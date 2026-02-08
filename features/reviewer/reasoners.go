package reviewer

import (
	"context"
	"fmt"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/yourorg/github-code-agent/features/analyzer"
)

// RegisterReasoners registers all review reasoners
func RegisterReasoners(app *agent.Agent) {
	reviewer := NewReviewer(app)

	app.RegisterReasoner("review_code",
		func(ctx context.Context, input map[string]any) (any, error) {
			return reviewCodeReasoner(ctx, reviewer, input)
		},
		agent.WithDescription("Performs AI-powered code review"))

	app.RegisterReasoner("detect_security_issues",
		func(ctx context.Context, input map[string]any) (any, error) {
			return detectSecurityIssuesReasoner(ctx, reviewer, input)
		},
		agent.WithDescription("Detects security vulnerabilities in code"))
}

// reviewCodeReasoner is the reasoner function for code review
func reviewCodeReasoner(ctx context.Context, reviewer *Reviewer, input map[string]any) (any, error) {
	// Extract files from input
	filesData, ok := input["files"].([]any)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'files' parameter")
	}

	// Convert to FileChange array
	files := make([]*analyzer.FileChange, 0, len(filesData))
	for _, f := range filesData {
		if fileMap, ok := f.(map[string]any); ok {
			file := &analyzer.FileChange{
				Filename:  getString(fileMap, "filename"),
				Status:    getString(fileMap, "status"),
				Additions: getInt(fileMap, "additions"),
				Deletions: getInt(fileMap, "deletions"),
				Changes:   getInt(fileMap, "changes"),
				Patch:     getString(fileMap, "patch"),
				Content:   getString(fileMap, "content"),
				Language:  getString(fileMap, "language"),
			}
			files = append(files, file)
		}
	}

	// Extract PR context
	prContext, _ := input["pr_context"].(map[string]any)
	if prContext == nil {
		prContext = make(map[string]any)
	}

	// Perform review
	report, err := reviewer.ReviewCode(ctx, files, prContext)
	if err != nil {
		return nil, err
	}

	// Convert issues to []any for easier processing downstream
	// Note: Use float64 for numeric values since JSON unmarshaling uses float64
	issuesAny := make([]any, len(report.Issues))
	for i, issue := range report.Issues {
		issuesAny[i] = map[string]any{
			"id":          issue.ID,
			"file_path":   issue.FilePath,
			"line":        float64(issue.Line), // Convert to float64 for JSON compatibility
			"severity":    issue.Severity,
			"category":    issue.Category,
			"title":       issue.Title,
			"description": issue.Description,
			"suggestion":  issue.Suggestion,
		}
	}

	return map[string]any{
		"report": map[string]any{
			"issues":             issuesAny,
			"summary":            report.Summary,
			"total_issues":       report.TotalIssues,
			"issues_by_severity": report.IssuesBySeverity,
			"issues_by_category": report.IssuesByCategory,
			"model":              report.Model,
		},
		"model": report.Model,
	}, nil
}

// detectSecurityIssuesReasoner is the reasoner function for security detection
func detectSecurityIssuesReasoner(ctx context.Context, reviewer *Reviewer, input map[string]any) (any, error) {
	// Extract files from input
	filesData, ok := input["files"].([]any)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'files' parameter")
	}

	// Convert to FileChange array
	files := make([]*analyzer.FileChange, 0, len(filesData))
	for _, f := range filesData {
		if fileMap, ok := f.(map[string]any); ok {
			file := &analyzer.FileChange{
				Filename: getString(fileMap, "filename"),
				Status:   getString(fileMap, "status"),
				Content:  getString(fileMap, "content"),
				Language: getString(fileMap, "language"),
			}
			files = append(files, file)
		}
	}

	// Detect security issues
	issues, err := reviewer.DetectSecurityIssues(ctx, files)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"security_issues": issues,
		"count":           len(issues),
	}, nil
}

// Helper functions to safely extract values from maps
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key].(int); ok {
		return v
	}
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}
