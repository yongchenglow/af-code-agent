package testexec

import (
	"context"
	"errors"
	"fmt"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/Agent-Field/agentfield/sdk/go/ai"
	_ "embed"
	"github.com/yourorg/github-code-agent/agents/planner"
	"github.com/yourorg/github-code-agent/pkg/constants"
	"github.com/yourorg/github-code-agent/pkg/utils"
)

//go:embed prompts/system.md
var testExecutorPrompt string

//go:embed prompts/task.md
var testExecutorTask string

// Executor handles test generation
type Executor struct {
	agent *agent.Agent
}

// NewExecutor creates a new test executor
func NewExecutor(a *agent.Agent) *Executor {
	return &Executor{
		agent: a,
	}
}

// TestResult contains the result of test generation
type TestResult struct {
	GapID     string
	TestFile  string
	TestCode  string
	Success   bool
	Error     string
	TestCount int
}

// WriteTests writes tests for a single test gap
func (e *Executor) WriteTests(ctx context.Context, gap *planner.TestGap, fixCode string) (*TestResult, error) {
	// Build task prompt with template
	taskPrompt := fmt.Sprintf(testExecutorTask,
		gap.Description,
		gap.TestFile,
		gap.Framework,
		gap.TestCount,
	)

	// Add test cases if present
	if len(gap.TestCases) > 0 {
		taskPrompt += "\n**Test Cases**:\n"
		for i, tc := range gap.TestCases {
			taskPrompt += fmt.Sprintf("%d. %s\n", i+1, tc)
		}
	}

	// Add fixed code if present
	if fixCode != "" {
		taskPrompt += fmt.Sprintf("\n**Fixed Code**:\n```go\n%s\n```\n", fixCode)
	}

	taskPrompt += "\n## Your Task\n\nWrite tests that verify the fix works correctly. Include edge cases and error conditions.\n\nOutput the complete test file.\n"

	// Create context with timeout
	aiCtx, cancel := context.WithTimeout(ctx, constants.DefaultAITimeout)
	defer cancel()

	// Call AI to generate tests
	response, err := e.agent.AI(aiCtx, taskPrompt,
		ai.WithSystem(testExecutorPrompt),
		ai.WithTemperature(constants.DefaultAITemperature),
		ai.WithMaxTokens(constants.TestAIMaxTokens))

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("test generation timeout: %w", err)
		}
		return nil, fmt.Errorf("test generation failed: %w", err)
	}

	// Parse response to extract test code
	testCode := utils.ExtractCodeFromResponse(response.Text())

	// Count tests in generated code
	testCount := countTests(testCode)

	return &TestResult{
		GapID:     gap.ID,
		TestFile:  gap.TestFile,
		TestCode:  testCode,
		Success:   testCode != "",
		TestCount: testCount,
	}, nil
}

// WriteTestsBatch writes tests for multiple test gaps
func (e *Executor) WriteTestsBatch(ctx context.Context, gaps []*planner.TestGap, fixCodes map[string]string) ([]*TestResult, error) {
	results := make([]*TestResult, 0, len(gaps))

	// Write tests for each gap
	for _, gap := range gaps {
		fixCode := ""
		if fixCodes != nil {
			fixCode = fixCodes[gap.ID]
		}

		result, err := e.WriteTests(ctx, gap, fixCode)
		if err != nil {
			results = append(results, &TestResult{
				GapID:   gap.ID,
				Success: false,
				Error:   err.Error(),
			})
			continue
		}

		results = append(results, result)
	}

	return results, nil
}

// countTests counts the number of test functions in the code
func countTests(code string) int {
	if code == "" {
		return 0
	}

	// Simple heuristic: count lines starting with "func Test"
	count := 0
	lines := splitLines(code)
	for _, line := range lines {
		trimmed := trimSpace(line)
		if len(trimmed) >= 9 && trimmed[:9] == "func Test" {
			count++
		}
	}

	return count
}

// Helper functions

func splitLines(s string) []string {
	result := []string{}
	current := ""
	for _, ch := range s {
		if ch == '\n' {
			result = append(result, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
