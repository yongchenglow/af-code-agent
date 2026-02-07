package performance

import (
	"context"
	"fmt"
	"sync"
)

// ParallelExecutor executes tasks in parallel with concurrency control
type ParallelExecutor struct {
	maxConcurrency int
	semaphore      chan struct{}
}

// NewParallelExecutor creates a new parallel executor
func NewParallelExecutor(maxConcurrency int) *ParallelExecutor {
	if maxConcurrency <= 0 {
		maxConcurrency = 10 // default
	}

	return &ParallelExecutor{
		maxConcurrency: maxConcurrency,
		semaphore:      make(chan struct{}, maxConcurrency),
	}
}

// Task represents a task to be executed
type Task func(ctx context.Context) (any, error)

// Result represents the result of a task execution
type Result struct {
	Value any
	Error error
	Index int
}

// Execute runs multiple tasks in parallel and returns results in order
func (p *ParallelExecutor) Execute(ctx context.Context, tasks []Task) ([]Result, error) {
	if len(tasks) == 0 {
		return []Result{}, nil
	}

	results := make([]Result, len(tasks))
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Context for cancellation
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i, task := range tasks {
		wg.Add(1)

		go func(index int, t Task) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case p.semaphore <- struct{}{}:
				defer func() { <-p.semaphore }()
			case <-execCtx.Done():
				mu.Lock()
				results[index] = Result{
					Error: execCtx.Err(),
					Index: index,
				}
				mu.Unlock()
				return
			}

			// Execute task
			value, err := t(execCtx)

			// Store result
			mu.Lock()
			results[index] = Result{
				Value: value,
				Error: err,
				Index: index,
			}
			mu.Unlock()
		}(i, task)
	}

	wg.Wait()
	return results, nil
}

// ExecuteWithCallback runs tasks in parallel and calls callback for each result
func (p *ParallelExecutor) ExecuteWithCallback(
	ctx context.Context,
	tasks []Task,
	callback func(Result),
) error {
	if len(tasks) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i, task := range tasks {
		wg.Add(1)

		go func(index int, t Task) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case p.semaphore <- struct{}{}:
				defer func() { <-p.semaphore }()
			case <-execCtx.Done():
				callback(Result{
					Error: execCtx.Err(),
					Index: index,
				})
				return
			}

			// Execute task
			value, err := t(execCtx)

			// Call callback with result
			callback(Result{
				Value: value,
				Error: err,
				Index: index,
			})
		}(i, task)
	}

	wg.Wait()
	return nil
}

// FileAnalysisTask represents a file analysis task
type FileAnalysisTask struct {
	FilePath string
	Content  string
	Language string
}

// BatchFileAnalyzer analyzes multiple files in parallel
type BatchFileAnalyzer struct {
	executor *ParallelExecutor
}

// NewBatchFileAnalyzer creates a new batch file analyzer
func NewBatchFileAnalyzer(maxConcurrency int) *BatchFileAnalyzer {
	return &BatchFileAnalyzer{
		executor: NewParallelExecutor(maxConcurrency),
	}
}

// AnalyzeFiles analyzes multiple files in parallel
func (b *BatchFileAnalyzer) AnalyzeFiles(
	ctx context.Context,
	files []FileAnalysisTask,
	analyzeFunc func(context.Context, FileAnalysisTask) (any, error),
) ([]Result, error) {
	// Convert to generic tasks
	tasks := make([]Task, len(files))
	for i, file := range files {
		file := file // capture loop variable
		tasks[i] = func(ctx context.Context) (any, error) {
			return analyzeFunc(ctx, file)
		}
	}

	return b.executor.Execute(ctx, tasks)
}

// WorkerPool manages a pool of workers for task execution
type WorkerPool struct {
	workers    int
	taskQueue  chan Task
	resultChan chan Result
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(ctx context.Context, numWorkers int) *WorkerPool {
	execCtx, cancel := context.WithCancel(ctx)

	return &WorkerPool{
		workers:    numWorkers,
		taskQueue:  make(chan Task, numWorkers*2),
		resultChan: make(chan Result, numWorkers*2),
		ctx:        execCtx,
		cancel:     cancel,
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// worker is the worker goroutine
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	for {
		select {
		case <-wp.ctx.Done():
			return
		case task, ok := <-wp.taskQueue:
			if !ok {
				return
			}

			// Execute task
			value, err := task(wp.ctx)

			// Send result
			select {
			case wp.resultChan <- Result{
				Value: value,
				Error: err,
				Index: id,
			}:
			case <-wp.ctx.Done():
				return
			}
		}
	}
}

// Submit submits a task to the worker pool
func (wp *WorkerPool) Submit(task Task) error {
	select {
	case wp.taskQueue <- task:
		return nil
	case <-wp.ctx.Done():
		return fmt.Errorf("worker pool is closed")
	}
}

// Results returns the result channel
func (wp *WorkerPool) Results() <-chan Result {
	return wp.resultChan
}

// Stop stops the worker pool
func (wp *WorkerPool) Stop() {
	close(wp.taskQueue)
	wp.wg.Wait()
	close(wp.resultChan)
	wp.cancel()
}
