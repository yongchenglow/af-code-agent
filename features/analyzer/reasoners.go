package analyzer

import (
	"context"
	"fmt"
	"log"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
)

// RegisterReasoners registers all analyzer reasoners
func RegisterReasoners(app *agent.Agent) {
	app.RegisterReasoner("analyze_pr", AnalyzePRReasoner,
		agent.WithDescription("Analyzes PR files and extracts code metrics"))

	app.RegisterReasoner("fetch_pr_files", FetchPRFilesReasoner,
		agent.WithDescription("Fetches changed files from a pull request"))
}

// AnalyzePRReasoner orchestrates the complete PR analysis
func AnalyzePRReasoner(ctx context.Context, input map[string]any) (any, error) {
	repo, ok := input["repo"].(string)
	if !ok {
		return nil, fmt.Errorf("repo is required")
	}

	prNumber, ok := input["pr_number"].(int)
	if !ok {
		// Try float64 (JSON numbers)
		if prNumberFloat, ok := input["pr_number"].(float64); ok {
			prNumber = int(prNumberFloat)
		} else {
			return nil, fmt.Errorf("pr_number is required")
		}
	}

	log.Printf("Analyzing PR #%d in repo %s", prNumber, repo)

	// Get agent from context
	agentInstance := GetAgentFromContext(ctx)
	if agentInstance == nil {
		return nil, fmt.Errorf("agent not found in context")
	}

	// Fetch PR files
	filesInput := map[string]any{
		"repo":      repo,
		"pr_number": prNumber,
	}

	filesResult, err := agentInstance.CallLocal(ctx, "fetch_pr_files", filesInput)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR files: %w", err)
	}

	files, ok := filesResult.(map[string]any)["files"].([]any)
	if !ok {
		return nil, fmt.Errorf("invalid files result")
	}

	log.Printf("Fetched %d files from PR #%d", len(files), prNumber)

	// Store results in workflow memory
	memKey := fmt.Sprintf("pr-%d-analysis", prNumber)
	analysisData := map[string]any{
		"repo":        repo,
		"pr_number":   prNumber,
		"files_count": len(files),
		"files":       files,
	}

	if err := agentInstance.Memory().WorkflowScope().Set(ctx, memKey, analysisData); err != nil {
		log.Printf("Warning: Failed to store analysis in memory: %v", err)
	}

	return map[string]any{
		"success":     true,
		"repo":        repo,
		"pr_number":   prNumber,
		"files_count": len(files),
		"files":       files,
	}, nil
}

// FetchPRFilesReasoner fetches changed files from a PR
func FetchPRFilesReasoner(ctx context.Context, input map[string]any) (any, error) {
	repo, ok := input["repo"].(string)
	if !ok {
		return nil, fmt.Errorf("repo is required")
	}

	prNumber, ok := input["pr_number"].(int)
	if !ok {
		if prNumberFloat, ok := input["pr_number"].(float64); ok {
			prNumber = int(prNumberFloat)
		} else {
			return nil, fmt.Errorf("pr_number is required")
		}
	}

	log.Printf("Fetching files for PR #%d in %s", prNumber, repo)

	// TODO: Use GitHub client to fetch actual PR files
	// For Phase 1, return mock data
	mockFiles := []any{
		map[string]any{
			"filename": "main.go",
			"status":   "modified",
			"additions": 10,
			"deletions": 5,
		},
		map[string]any{
			"filename": "config.go",
			"status":   "added",
			"additions": 50,
			"deletions": 0,
		},
	}

	return map[string]any{
		"repo":      repo,
		"pr_number": prNumber,
		"files":     mockFiles,
	}, nil
}

// GetAgentFromContext retrieves the agent instance from context
func GetAgentFromContext(ctx context.Context) *agent.Agent {
	if agentInstance, ok := ctx.Value("agent").(*agent.Agent); ok {
		return agentInstance
	}
	return nil
}
