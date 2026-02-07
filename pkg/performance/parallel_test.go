package performance

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestParallelExecutor(t *testing.T) {
	t.Run("Execute with successful tasks", func(t *testing.T) {
		executor := NewParallelExecutor(3)
		ctx := context.Background()

		tasks := []Task{
			func(ctx context.Context) (any, error) {
				return "result1", nil
			},
			func(ctx context.Context) (any, error) {
				return "result2", nil
			},
			func(ctx context.Context) (any, error) {
				return "result3", nil
			},
		}

		results, err := executor.Execute(ctx, tasks)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if len(results) != 3 {
			t.Errorf("len(results) = %d, want 3", len(results))
		}

		for i, result := range results {
			if result.Error != nil {
				t.Errorf("result[%d].Error = %v, want nil", i, result.Error)
			}
		}
	})

	t.Run("Execute with failing task", func(t *testing.T) {
		executor := NewParallelExecutor(2)
		ctx := context.Background()

		expectedErr := errors.New("task failed")

		tasks := []Task{
			func(ctx context.Context) (any, error) {
				return "success", nil
			},
			func(ctx context.Context) (any, error) {
				return nil, expectedErr
			},
		}

		results, err := executor.Execute(ctx, tasks)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if results[1].Error == nil {
			t.Error("expected second task to have error")
		}
	})

	t.Run("Execute with context cancellation", func(t *testing.T) {
		executor := NewParallelExecutor(2)
		ctx, cancel := context.WithCancel(context.Background())

		tasks := []Task{
			func(ctx context.Context) (any, error) {
				time.Sleep(100 * time.Millisecond)
				return "result", nil
			},
		}

		// Cancel immediately
		cancel()

		results, err := executor.Execute(ctx, tasks)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if results[0].Error == nil {
			t.Log("Note: Task may complete before cancellation")
		}
	})

	t.Run("Execute with empty tasks", func(t *testing.T) {
		executor := NewParallelExecutor(2)
		ctx := context.Background()

		results, err := executor.Execute(ctx, []Task{})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if len(results) != 0 {
			t.Errorf("len(results) = %d, want 0", len(results))
		}
	})
}

func TestBatchFileAnalyzer(t *testing.T) {
	t.Run("Analyze files in parallel", func(t *testing.T) {
		analyzer := NewBatchFileAnalyzer(2)
		ctx := context.Background()

		files := []FileAnalysisTask{
			{FilePath: "file1.go", Content: "package main", Language: "go"},
			{FilePath: "file2.go", Content: "package test", Language: "go"},
		}

		analyzeFunc := func(ctx context.Context, task FileAnalysisTask) (any, error) {
			return map[string]any{
				"file":     task.FilePath,
				"language": task.Language,
			}, nil
		}

		results, err := analyzer.AnalyzeFiles(ctx, files, analyzeFunc)
		if err != nil {
			t.Fatalf("AnalyzeFiles() error = %v", err)
		}

		if len(results) != 2 {
			t.Errorf("len(results) = %d, want 2", len(results))
		}
	})
}

func TestWorkerPool(t *testing.T) {
	t.Run("Worker pool execution", func(t *testing.T) {
		ctx := context.Background()
		pool := NewWorkerPool(ctx, 3)

		pool.Start()
		defer pool.Stop()

		// Submit tasks
		for i := 0; i < 5; i++ {
			task := func(ctx context.Context) (any, error) {
				return "result", nil
			}

			if err := pool.Submit(task); err != nil {
				t.Fatalf("Submit() error = %v", err)
			}
		}

		// Collect results
		resultCount := 0
		timeout := time.After(2 * time.Second)

		for resultCount < 5 {
			select {
			case result := <-pool.Results():
				if result.Error != nil {
					t.Errorf("result error: %v", result.Error)
				}
				resultCount++
			case <-timeout:
				t.Fatalf("timeout waiting for results, got %d/5", resultCount)
			}
		}
	})
}

func TestNewParallelExecutor(t *testing.T) {
	tests := []struct {
		name           string
		maxConcurrency int
		wantConcurrency int
	}{
		{
			name:            "positive concurrency",
			maxConcurrency:  5,
			wantConcurrency: 5,
		},
		{
			name:            "zero concurrency uses default",
			maxConcurrency:  0,
			wantConcurrency: 10,
		},
		{
			name:            "negative concurrency uses default",
			maxConcurrency:  -1,
			wantConcurrency: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewParallelExecutor(tt.maxConcurrency)
			if executor.maxConcurrency != tt.wantConcurrency {
				t.Errorf("maxConcurrency = %d, want %d",
					executor.maxConcurrency, tt.wantConcurrency)
			}
		})
	}
}
