package security

import (
	"context"
	"errors"
	"fmt"

	_ "embed"
	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/Agent-Field/agentfield/sdk/go/ai"
	"github.com/yourorg/github-code-agent/agents/analyzer"
	"github.com/yourorg/github-code-agent/agents/planner"
	"github.com/yourorg/github-code-agent/pkg/constants"
	"github.com/yourorg/github-code-agent/pkg/utils"
)

//go:embed prompts/system.md
var securityExecutorPrompt string

//go:embed prompts/task.md
var securityExecutorTask string

// Executor handles security vulnerability fixes
type Executor struct {
	agent *agent.Agent
}

// NewExecutor creates a new security executor
func NewExecutor(a *agent.Agent) *Executor {
	return &Executor{
		agent: a,
	}
}

// FixResult contains the result of a security fix
type FixResult struct {
	IssueID      string
	FilePath     string
	Language     string
	OriginalCode string
	FixedCode    string
	Success      bool
	Error        string
}

// FixSecurityIssue fixes a single security vulnerability
func (e *Executor) FixSecurityIssue(ctx context.Context, issue *planner.SecurityIssue, fileContent string, language string) (*FixResult, error) {
	// Extract relevant code section
	originalCode := utils.ExtractCodeSection(fileContent, issue.Line, constants.MaxFileContextLines)

	// Build task prompt with template
	taskPrompt := fmt.Sprintf(securityExecutorTask,
		issue.Type,
		issue.Severity,
		issue.CWE,
		issue.OWASP,
		issue.FilePath,
		issue.Line,
		issue.Description,
		issue.Remediation,
	)
	taskPrompt += fmt.Sprintf("\n\n**Original Code**:\n```%s\n%s\n```\n\n**Fixed Code**:\n", language, originalCode)

	// Create context with timeout
	aiCtx, cancel := context.WithTimeout(ctx, constants.DefaultAITimeout)
	defer cancel()

	// Call AI to generate fix
	response, err := e.agent.AI(aiCtx, taskPrompt,
		ai.WithSystem(securityExecutorPrompt),
		ai.WithTemperature(constants.LowAITemperature), // Lower temperature for security fixes
		ai.WithMaxTokens(constants.FixerAIMaxTokens))

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("security fix timeout: %w", err)
		}
		return nil, fmt.Errorf("security fix generation failed: %w", err)
	}

	// Parse response to extract fixed code
	fixedCode := utils.ExtractCodeFromResponse(response.Text())

	return &FixResult{
		IssueID:      issue.ID,
		FilePath:     issue.FilePath,
		Language:     language,
		OriginalCode: originalCode,
		FixedCode:    fixedCode,
		Success:      fixedCode != "",
	}, nil
}

// FixSecurityIssuesBatch fixes multiple security issues
func (e *Executor) FixSecurityIssuesBatch(ctx context.Context, issues []*planner.SecurityIssue, files []*analyzer.FileChange) ([]*FixResult, error) {
	results := make([]*FixResult, 0, len(issues))

	// Create a map for quick file lookup
	fileMap := make(map[string]*analyzer.FileChange)
	for _, file := range files {
		fileMap[file.Filename] = file
	}

	// Fix each issue
	for _, issue := range issues {
		file, ok := fileMap[issue.FilePath]
		if !ok {
			results = append(results, &FixResult{
				IssueID: issue.ID,
				Success: false,
				Error:   fmt.Sprintf("file not found: %s", issue.FilePath),
			})
			continue
		}

		result, err := e.FixSecurityIssue(ctx, issue, file.Content, file.Language)
		if err != nil {
			results = append(results, &FixResult{
				IssueID: issue.ID,
				Success: false,
				Error:   err.Error(),
			})
			continue
		}

		results = append(results, result)
	}

	return results, nil
}
