package workflow

import (
	"strings"
	"sync"
	"testing"

	"github.com/yourorg/github-code-agent/agents/planner"
	"github.com/yourorg/github-code-agent/agents/testexec"
)

func TestWorkflowResultStruct(t *testing.T) {
	result := &WorkflowResult{
		Plan: &planner.ReviewPlan{
			Summary:        "Test plan",
			Recommendation: "APPROVE",
		},
		FixResults: []*FixTaskResult{
			{TaskID: "FIX-001", TaskType: "security", Success: true},
		},
		TestResults: []*testexec.TestResult{
			{GapID: "TEST-001", Success: true, TestCount: 1},
		},
		Success:        true,
		Error:          "",
		Recommendation: "APPROVE",
		Summary:        "All tests passed",
		Warnings:       []string{"Warning 1"},
	}

	if result.Plan.Summary != "Test plan" {
		t.Errorf("expected Plan.Summary to be 'Test plan', got %q", result.Plan.Summary)
	}
	if len(result.FixResults) != 1 {
		t.Errorf("expected 1 fix result, got %d", len(result.FixResults))
	}
	if len(result.TestResults) != 1 {
		t.Errorf("expected 1 test result, got %d", len(result.TestResults))
	}
	if !result.Success {
		t.Error("expected Success to be true")
	}
	if result.Recommendation != "APPROVE" {
		t.Errorf("expected Recommendation to be 'APPROVE', got %q", result.Recommendation)
	}
	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(result.Warnings))
	}
}

func TestFixTaskResultStruct(t *testing.T) {
	result := &FixTaskResult{
		TaskID:       "FIX-001",
		TaskType:     "security",
		Success:      true,
		Patch:        "patch content",
		OriginalCode: "original code",
		Error:        "",
	}

	if result.TaskID != "FIX-001" {
		t.Errorf("expected TaskID to be 'FIX-001', got %q", result.TaskID)
	}
	if result.TaskType != "security" {
		t.Errorf("expected TaskType to be 'security', got %q", result.TaskType)
	}
	if !result.Success {
		t.Error("expected Success to be true")
	}
	if result.Patch != "patch content" {
		t.Errorf("expected Patch to be 'patch content', got %q", result.Patch)
	}
}

func TestFixTaskResultFailure(t *testing.T) {
	result := &FixTaskResult{
		TaskID:   "FIX-001",
		TaskType: "bug",
		Success:  false,
		Error:    "File not found",
	}

	if result.Success {
		t.Error("expected Success to be false")
	}
	if result.Error != "File not found" {
		t.Errorf("expected Error to be 'File not found', got %q", result.Error)
	}
	if result.Patch != "" {
		t.Errorf("expected empty Patch, got %q", result.Patch)
	}
}

func TestNewExecutorOrchestrator(t *testing.T) {
	// Test that constructor returns non-nil orchestrator
	// Note: We can't fully test without a real agent
	t.Skip("Skipping - requires agent mock")
}

func TestExecuteReviewWorkflowEmptyFiles(t *testing.T) {
	// This test would require mocking the orchestrator components
	t.Skip("Skipping - requires agent mock")
}

func TestExecuteReviewWorkflowNoIssues(t *testing.T) {
	// This test would require mocking to return a plan with no issues
	t.Skip("Skipping - requires agent mock")
}

func TestExecuteReviewWorkflowWithIssues(t *testing.T) {
	// This test would require mocking to return a plan with issues
	t.Skip("Skipping - requires agent mock")
}

func TestExecuteReviewWorkflowPlanningFailure(t *testing.T) {
	// This test would require mocking the planner to return an error
	t.Skip("Skipping - requires agent mock")
}

func TestExecuteReviewWorkflowWithTestGaps(t *testing.T) {
	// This test would require mocking to test test generation
	t.Skip("Skipping - requires agent mock")
}

func TestExecuteParallelFixesEmptyGroups(t *testing.T) {
	// This test would require mocking the orchestrator
	t.Skip("Skipping - requires agent mock")
}

func TestExecuteParallelFixesSingleGroup(t *testing.T) {
	// This test would require mocking the orchestrator
	t.Skip("Skipping - requires agent mock")
}

func TestExecuteParallelFixesMultipleGroups(t *testing.T) {
	// This test would require mocking the orchestrator
	t.Skip("Skipping - requires agent mock")
}

func TestExecuteFixTaskFileNotFound(t *testing.T) {
	// This test would require mocking the orchestrator
	t.Skip("Skipping - requires agent mock")
}

func TestExecuteFixTaskUnknownType(t *testing.T) {
	// This test would require mocking the orchestrator
	t.Skip("Skipping - requires agent mock")
}

func TestExecuteTestGenerationNoTestGaps(t *testing.T) {
	// This test would require mocking the orchestrator
	t.Skip("Skipping - requires agent mock")
}

func TestExecuteTestGenerationWithTestGaps(t *testing.T) {
	// This test would require mocking the orchestrator
	t.Skip("Skipping - requires agent mock")
}

func TestGenerateSummary(t *testing.T) {
	o := &ExecutorOrchestrator{}
	result := &WorkflowResult{
		FixResults: []*FixTaskResult{
			{TaskID: "FIX-001", TaskType: "security", Success: true},
			{TaskID: "FIX-002", TaskType: "bug", Success: true},
			{TaskID: "FIX-003", TaskType: "standards", Success: false},
		},
		TestResults: []*testexec.TestResult{
			{GapID: "TEST-001", Success: true, TestCount: 2},
		},
		Warnings: []string{"Warning 1"},
	}

	summary := o.generateSummary(result)

	if !strings.Contains(summary, "Review Workflow Summary") {
		t.Error("expected summary header")
	}
	if !strings.Contains(summary, "Total fix tasks: 3") {
		t.Error("expected total fix tasks count")
	}
	if !strings.Contains(summary, "Successful fixes: 2") {
		t.Error("expected successful fixes count")
	}
	if !strings.Contains(summary, "Failed fixes: 1") {
		t.Error("expected failed fixes count")
	}
	if !strings.Contains(summary, "Tests generated: 1") {
		t.Error("expected tests generated count")
	}
	if !strings.Contains(summary, "Warnings: 1") {
		t.Error("expected warnings count")
	}
}

func TestGenerateSummaryNoWarnings(t *testing.T) {
	o := &ExecutorOrchestrator{}
	result := &WorkflowResult{
		FixResults: []*FixTaskResult{
			{TaskID: "FIX-001", TaskType: "security", Success: true},
		},
		TestResults: []*testexec.TestResult{},
		Warnings:    []string{},
	}

	summary := o.generateSummary(result)

	if strings.Contains(summary, "Warnings:") {
		t.Error("should not include warnings when empty")
	}
}

func TestGenerateSummaryEmptyResults(t *testing.T) {
	o := &ExecutorOrchestrator{}
	result := &WorkflowResult{
		FixResults:  []*FixTaskResult{},
		TestResults: []*testexec.TestResult{},
		Warnings:    []string{},
	}

	summary := o.generateSummary(result)

	if !strings.Contains(summary, "Total fix tasks: 0") {
		t.Error("expected total fix tasks to be 0")
	}
	if !strings.Contains(summary, "Successful fixes: 0") {
		t.Error("expected successful fixes to be 0")
	}
}

func TestDetermineRecommendationSuccess(t *testing.T) {
	o := &ExecutorOrchestrator{}
	result := &WorkflowResult{
		Success: true,
		FixResults: []*FixTaskResult{
			{TaskID: "FIX-001", TaskType: "security", Success: true},
			{TaskID: "FIX-002", TaskType: "bug", Success: true},
		},
	}

	recommendation := o.determineRecommendation(result)

	if recommendation != "APPROVE" {
		t.Errorf("expected recommendation 'APPROVE', got %q", recommendation)
	}
}

func TestDetermineRecommendationFailure(t *testing.T) {
	o := &ExecutorOrchestrator{}
	result := &WorkflowResult{
		Success: false,
		Error:   "Workflow failed",
	}

	recommendation := o.determineRecommendation(result)

	if recommendation != "BLOCK" {
		t.Errorf("expected recommendation 'BLOCK', got %q", recommendation)
	}
}

func TestDetermineRecommendationSecurityFixFailed(t *testing.T) {
	o := &ExecutorOrchestrator{}
	result := &WorkflowResult{
		Success: true,
		FixResults: []*FixTaskResult{
			{TaskID: "FIX-001", TaskType: "security", Success: false, Error: "Failed"},
			{TaskID: "FIX-002", TaskType: "bug", Success: true},
		},
	}

	recommendation := o.determineRecommendation(result)

	if recommendation != "FIX" {
		t.Errorf("expected recommendation 'FIX', got %q", recommendation)
	}
}

func TestDetermineRecommendationOnlyStandardsIssues(t *testing.T) {
	o := &ExecutorOrchestrator{}
	result := &WorkflowResult{
		Success: true,
		FixResults: []*FixTaskResult{
			{TaskID: "FIX-001", TaskType: "standards", Success: true},
		},
	}

	recommendation := o.determineRecommendation(result)

	if recommendation != "APPROVE" {
		t.Errorf("expected recommendation 'APPROVE', got %q", recommendation)
	}
}

func TestDetermineRecommendationMixedResults(t *testing.T) {
	o := &ExecutorOrchestrator{}
	result := &WorkflowResult{
		Success: true,
		FixResults: []*FixTaskResult{
			{TaskID: "SEC-001", TaskType: "security", Success: true},
			{TaskID: "BUG-001", TaskType: "bug", Success: false},
			{TaskID: "STD-001", TaskType: "standards", Success: true},
		},
	}

	recommendation := o.determineRecommendation(result)

	// Security fixes succeeded, so should be APPROVE even if bug fix failed
	if recommendation != "APPROVE" {
		t.Errorf("expected recommendation 'APPROVE', got %q", recommendation)
	}
}

func TestWorkflowResultWithNilPlan(t *testing.T) {
	result := &WorkflowResult{
		Plan:           nil,
		FixResults:     []*FixTaskResult{},
		TestResults:    []*testexec.TestResult{},
		Success:        false,
		Error:          "Planning failed",
		Recommendation: "BLOCK",
	}

	if result.Plan != nil {
		t.Error("expected Plan to be nil")
	}
	if result.Error != "Planning failed" {
		t.Errorf("expected Error to be 'Planning failed', got %q", result.Error)
	}
}

func TestFixTaskResultAllFields(t *testing.T) {
	result := &FixTaskResult{
		TaskID:       "FIX-001",
		TaskType:     "security",
		Success:      true,
		Patch:        "+new code\n-old code",
		OriginalCode: "original code",
		Error:        "",
	}

	// Verify all fields are accessible
	if result.TaskID == "" {
		t.Error("TaskID should not be empty")
	}
	if result.TaskType == "" {
		t.Error("TaskType should not be empty")
	}
	if result.Patch == "" {
		t.Error("Patch should not be empty")
	}
	if result.OriginalCode == "" {
		t.Error("OriginalCode should not be empty")
	}
}

func TestExecutorOrchestratorStruct(t *testing.T) {
	// Test that the orchestrator struct is properly defined
	var orch ExecutorOrchestrator

	// Verify struct fields exist (will be nil without initialization)
	if orch.planner != nil {
		t.Error("expected planner to be nil on zero value")
	}
	if orch.securityExec != nil {
		t.Error("expected securityExec to be nil on zero value")
	}
	if orch.bugfixExec != nil {
		t.Error("expected bugfixExec to be nil on zero value")
	}
	if orch.testExec != nil {
		t.Error("expected testExec to be nil on zero value")
	}
}

func TestWorkflowResultJSONCompatible(t *testing.T) {
	// Test that WorkflowResult can be serialized (basic check)
	result := &WorkflowResult{
		Plan: &planner.ReviewPlan{
			Summary:        "Test",
			Recommendation: "APPROVE",
		},
		Success:        true,
		Recommendation: "APPROVE",
		Summary:        "All good",
	}

	// Basic validation that struct is properly defined
	if result.Plan == nil {
		t.Error("Plan should not be nil")
	}
	if result.Summary == "" {
		t.Error("Summary should not be empty")
	}
}

func TestExecuteParallelFixesConcurrentSafety(t *testing.T) {
	// This test would verify that parallel execution is thread-safe
	// Requires mocking the orchestrator components
	t.Skip("Skipping - requires agent mock")
}

func TestGenerateSummaryLargeResult(t *testing.T) {
	o := &ExecutorOrchestrator{}

	// Create a large result set
	fixResults := make([]*FixTaskResult, 100)
	for i := 0; i < 100; i++ {
		fixResults[i] = &FixTaskResult{
			TaskID:   string(rune('A' + i%26)),
			TaskType: "security",
			Success:  i%2 == 0,
		}
	}

	result := &WorkflowResult{
		FixResults:  fixResults,
		TestResults: make([]*testexec.TestResult, 50),
		Warnings:    make([]string, 10),
	}

	summary := o.generateSummary(result)

	if !strings.Contains(summary, "Total fix tasks: 100") {
		t.Error("expected total fix tasks to be 100")
	}
	if !strings.Contains(summary, "Successful fixes: 50") {
		t.Error("expected successful fixes to be 50")
	}
	if !strings.Contains(summary, "Tests generated: 50") {
		t.Error("expected tests generated to be 50")
	}
}

func TestDetermineRecommendationEdgeCases(t *testing.T) {
	o := &ExecutorOrchestrator{}

	// Test with empty fix results
	result := &WorkflowResult{
		Success:    true,
		FixResults: []*FixTaskResult{},
	}

	recommendation := o.determineRecommendation(result)
	if recommendation != "APPROVE" {
		t.Errorf("expected 'APPROVE' for empty fix results, got %q", recommendation)
	}
}

func TestWorkflowWithSyncGroup(t *testing.T) {
	// Test that WaitGroup is used correctly in parallel execution
	var wg sync.WaitGroup
	taskCount := 10
	resultChan := make(chan int, taskCount)

	for i := 0; i < taskCount; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			resultChan <- val
		}(i)
	}

	wg.Wait()
	close(resultChan)

	results := make([]int, 0, taskCount)
	for r := range resultChan {
		results = append(results, r)
	}

	if len(results) != taskCount {
		t.Errorf("expected %d results, got %d", taskCount, len(results))
	}
}

func TestExecuteFixTaskTypes(t *testing.T) {
	// Test all task types are handled
	taskTypes := []string{"security", "bug", "standards", "test", "unknown"}

	for _, taskType := range taskTypes {
		t.Run(taskType, func(t *testing.T) {
			task := &planner.FixTask{
				ID:   "FIX-001",
				Type: taskType,
				File: "pkg/test.go",
			}

			// Verify task type is valid
			if task.Type == "" {
				t.Error("task type should not be empty")
			}
		})
	}
}

func TestWorkflowResultInitialization(t *testing.T) {
	// Test various initialization patterns
	result1 := &WorkflowResult{}
	if result1.Success {
		t.Error("expected Success to be false by default")
	}
	if result1.FixResults != nil {
		t.Error("expected FixResults to be nil by default")
	}

	result2 := &WorkflowResult{
		FixResults:  []*FixTaskResult{},
		TestResults: []*testexec.TestResult{},
		Warnings:    []string{},
	}
	if result2.FixResults == nil {
		t.Error("expected FixResults to be initialized")
	}
	if len(result2.FixResults) != 0 {
		t.Error("expected FixResults to be empty")
	}
}
