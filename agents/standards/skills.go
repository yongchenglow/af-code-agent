package standards

import (
	"context"
	"fmt"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/yourorg/github-code-agent/agents/analyzer"
	"github.com/yourorg/github-code-agent/pkg/config"
)

// RegisterSkills registers all standards validation skills (deterministic functions)
// Note: Go SDK uses RegisterReasoner for both skills and reasoners
func RegisterSkills(app *agent.Agent, cfg *config.Config) {
	validator := NewValidator(cfg)

	// validate_standards is rule-based validation (deterministic pattern matching)
	app.RegisterReasoner("validate_standards",
		func(ctx context.Context, input map[string]any) (any, error) {
			return validateStandardsSkill(ctx, validator, input)
		},
		agent.WithDescription("[SKILL] Validates code against configured standards using rule-based checks"))
}

// validateStandardsSkill is the skill function for standards validation
func validateStandardsSkill(ctx context.Context, validator *Validator, input map[string]any) (any, error) {
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
				Content:   getString(fileMap, "content"),
				Language:  getString(fileMap, "language"),
				Additions: getInt(fileMap, "additions"),
				Deletions: getInt(fileMap, "deletions"),
			}
			files = append(files, file)
		}
	}

	// Validate standards
	report, err := validator.ValidateStandards(ctx, files)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"report":             report,
		"total_violations":   report.TotalViolations,
		"violations_by_type": report.ViolationsByType,
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
