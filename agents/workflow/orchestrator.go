package workflow

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/yourorg/github-code-agent/agents/analyzer"
	"github.com/yourorg/github-code-agent/agents/bugfix"
	"github.com/yourorg/github-code-agent/agents/planner"
	"github.com/yourorg/github-code-agent/agents/security"
	"github.com/yourorg/github-code-agent/agents/testexec"
)

// ExecutorOrchestrator orchestrates the planner-executor workflow
type ExecutorOrchestrator struct {
	planner      *planner.Planner
	securityExec *security.Executor
	bugfixExec   *bugfix.Executor
	testExec     *testexec.Executor
}

// NewExecutorOrchestrator creates a new orchestrator
func NewExecutorOrchestrator(a *agent.Agent) *ExecutorOrchestrator {
	return &ExecutorOrchestrator{
		planner:      planner.NewPlanner(a),
		securityExec: security.NewExecutor(a),
		bugfixExec:   bugfix.NewExecutor(a),
		testExec:     testexec.NewExecutor(a),
	}
}

// ExecuteReviewWorkflow executes the complete planner-executor workflow
func (o *ExecutorOrchestrator) ExecuteReviewWorkflow(ctx context.Context, files []*analyzer.FileChange, prContext map[string]any) (*WorkflowResult, error) {
	result := &WorkflowResult{
		Success: true,
	}

	// Step 1: Plan review (single AI call)
	log.Println("Starting review planning phase...")
	reviewPlan, err := o.planner.PlanReview(ctx, files, prContext)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("planning failed: %v", err)
		return result, err
	}

	result.Plan = reviewPlan
	log.Printf("Review planning completed: %d security, %d bugs, %d standards, %d test gaps",
		len(reviewPlan.SecurityIssues), len(reviewPlan.BugIssues),
		len(reviewPlan.StandardsViolations), len(reviewPlan.TestGaps))

	// If no issues found, approve
	if len(reviewPlan.SecurityIssues) == 0 && len(reviewPlan.BugIssues) == 0 &&
		len(reviewPlan.StandardsViolations) == 0 {
		result.Recommendation = "APPROVE"
		result.Summary = "No critical issues found"
		return result, nil
	}

	// Step 2: Execute fixes in parallel groups
	log.Println("Starting parallel fix execution...")
	o.executeParallelFixes(ctx, reviewPlan, files, result)

	// Step 3: Execute test generation (sequential, after fixes)
	if len(reviewPlan.TestGaps) > 0 {
		log.Println("Starting test generation...")
		o.executeTestGeneration(ctx, reviewPlan, result)
	}

	// Step 4: Generate summary
	result.Summary = o.generateSummary(result)
	result.Recommendation = o.determineRecommendation(result)

	return result, nil
}

// executeParallelFixes executes fix groups in parallel
func (o *ExecutorOrchestrator) executeParallelFixes(ctx context.Context, plan *planner.ReviewPlan, files []*analyzer.FileChange, result *WorkflowResult) {
	// Create file map for quick lookup
	fileMap := make(map[string]*analyzer.FileChange)
	for _, file := range files {
		fileMap[file.Filename] = file
	}

	// Execute each parallel group
	for groupIdx, group := range plan.FixPlan.ParallelGroups {
		log.Printf("Executing parallel group %d with %d tasks", groupIdx, len(group))

		// Execute group in parallel
		var wg sync.WaitGroup
		resultChan := make(chan *FixTaskResult, len(group))

		for _, task := range group {
			wg.Add(1)
			go func(task *planner.FixTask) {
				defer wg.Done()
				execResult := o.executeFixTask(ctx, task, fileMap)
				resultChan <- execResult
			}(task)
		}

		// Wait for all tasks in group to complete
		wg.Wait()
		close(resultChan)

		// Collect results
		for execResult := range resultChan {
			result.FixResults = append(result.FixResults, execResult)
			if !execResult.Success {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Task %s failed: %s", execResult.TaskID, execResult.Error))
			}
		}
	}
}

// executeFixTask executes a single fix task
func (o *ExecutorOrchestrator) executeFixTask(ctx context.Context, task *planner.FixTask, fileMap map[string]*analyzer.FileChange) *FixTaskResult {
	execResult := &FixTaskResult{
		TaskID:   task.ID,
		TaskType: task.Type,
		Success:  false,
	}

	_, ok := fileMap[task.File]
	if !ok {
		execResult.Error = fmt.Sprintf("file not found: %s", task.File)
		return execResult
	}

	switch task.Type {
	case "security":
		// Find the security issue - need plan context, simplified for now
		execResult.Error = "security executor needs plan context"

	case "bug":
		// Find the bug issue - need plan context, simplified for now
		execResult.Error = "bug executor needs plan context"

	case "standards":
		// Standards fixes would go here (often deterministic)
		execResult.Success = true
		execResult.Error = "standards fixes not yet implemented"

	default:
		execResult.Error = fmt.Sprintf("unknown task type: %s", task.Type)
	}

	return execResult
}

// executeTestGeneration executes test generation after fixes
func (o *ExecutorOrchestrator) executeTestGeneration(ctx context.Context, plan *planner.ReviewPlan, result *WorkflowResult) {
	// Collect fix patches for test context
	fixCodes := make(map[string]string)
	for _, fixResult := range result.FixResults {
		if fixResult.Success {
			fixCodes[fixResult.TaskID] = fixResult.Patch
		}
	}

	// Generate tests
	testResults, err := o.testExec.WriteTestsBatch(ctx, plan.TestGaps, fixCodes)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("test generation failed: %v", err))
		return
	}

	for _, testResult := range testResults {
		result.TestResults = append(result.TestResults, testResult)
		if !testResult.Success {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Test generation for %s failed: %s", testResult.GapID, testResult.Error))
		}
	}
}

func (o *ExecutorOrchestrator) generateSummary(result *WorkflowResult) string {
	summary := "Review Workflow Summary:\n"
	summary += fmt.Sprintf("- Total fix tasks: %d\n", len(result.FixResults))

	successCount := 0
	for _, r := range result.FixResults {
		if r.Success {
			successCount++
		}
	}

	summary += fmt.Sprintf("- Successful fixes: %d\n", successCount)
	summary += fmt.Sprintf("- Failed fixes: %d\n", len(result.FixResults)-successCount)
	summary += fmt.Sprintf("- Tests generated: %d\n", len(result.TestResults))

	if len(result.Warnings) > 0 {
		summary += fmt.Sprintf("- Warnings: %d\n", len(result.Warnings))
	}

	return summary
}

func (o *ExecutorOrchestrator) determineRecommendation(result *WorkflowResult) string {
	if !result.Success {
		return "BLOCK"
	}

	// Check if all critical fixes succeeded
	allCriticalSuccess := true
	for _, r := range result.FixResults {
		if r.TaskType == "security" && !r.Success {
			allCriticalSuccess = false
			break
		}
	}

	if allCriticalSuccess {
		return "APPROVE"
	}

	return "FIX"
}

// WorkflowResult contains the result of the workflow execution
type WorkflowResult struct {
	Plan          *planner.ReviewPlan
	FixResults    []*FixTaskResult
	TestResults   []*testexec.TestResult
	Success       bool
	Error         string
	Recommendation string
	Summary       string
	Warnings      []string
}

// FixTaskResult contains the result of a single fix task
type FixTaskResult struct {
	TaskID       string
	TaskType     string
	Success      bool
	Patch        string
	OriginalCode string
	Error        string
}
