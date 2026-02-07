package fixer

import (
	"context"
	"fmt"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
)

// Fixer handles fix generation with validation
type Fixer struct {
	agent *agent.Agent
}

// NewFixer creates a new Fixer instance
func NewFixer(app *agent.Agent) *Fixer {
	return &Fixer{
		agent: app,
	}
}

// RegisterReasoners registers all fixer reasoners with the agent
func RegisterReasoners(app *agent.Agent) {
	fixer := NewFixer(app)

	app.RegisterReasoner("generate_fixes_with_validation",
		func(ctx context.Context, input map[string]any) (any, error) {
			return generateFixesWithValidationReasoner(ctx, fixer, input)
		},
		agent.WithDescription("Generates and validates code fixes with retry loop"))

	app.RegisterReasoner("validate_fix",
		func(ctx context.Context, input map[string]any) (any, error) {
			return validateFixReasoner(ctx, fixer, input)
		},
		agent.WithDescription("Validates a code fix against multiple criteria"))
}

// generateFixesWithValidationReasoner is the reasoner for generating fixes with validation
func generateFixesWithValidationReasoner(ctx context.Context, fixer *Fixer, input map[string]any) (any, error) {
	// Extract inputs
	issues, ok := input["issues"].([]map[string]any)
	if !ok {
		// Try to convert []any to []map[string]any
		issuesAny, ok := input["issues"].([]any)
		if !ok {
			return nil, fmt.Errorf("issues must be a list of objects")
		}
		issues = make([]map[string]any, len(issuesAny))
		for i, item := range issuesAny {
			issues[i] = item.(map[string]any)
		}
	}

	files, ok := input["files"].([]map[string]any)
	if !ok {
		// Try to convert []any to []map[string]any
		filesAny, ok := input["files"].([]any)
		if !ok {
			return nil, fmt.Errorf("files must be a list of objects")
		}
		files = make([]map[string]any, len(filesAny))
		for i, item := range filesAny {
			files[i] = item.(map[string]any)
		}
	}

	// Get validation config
	config := DefaultValidationConfig()
	if configInput, ok := input["config"].(map[string]any); ok {
		if maxAttempts, ok := configInput["max_attempts"].(float64); ok {
			config.MaxAttempts = int(maxAttempts)
		}
		if timeoutSeconds, ok := configInput["timeout_seconds"].(float64); ok {
			config.TimeoutSeconds = int(timeoutSeconds)
		}
		if enableSyntax, ok := configInput["enable_syntax_check"].(bool); ok {
			config.EnableSyntaxCheck = enableSyntax
		}
		if enableLinting, ok := configInput["enable_linting"].(bool); ok {
			config.EnableLinting = enableLinting
		}
		if enableFormatting, ok := configInput["enable_formatting"].(bool); ok {
			config.EnableFormatting = enableFormatting
		}
		if enableSecurity, ok := configInput["enable_security_scan"].(bool); ok {
			config.EnableSecurityScan = enableSecurity
		}
		if autoFormat, ok := configInput["auto_format"].(bool); ok {
			config.AutoFormat = autoFormat
		}
	}

	// Generate fixes with validation
	result, err := GenerateFixesWithValidation(ctx, fixer.agent, issues, files, config)
	if err != nil {
		return nil, err
	}

	// Convert to map for output
	return map[string]any{
		"successful_fixes": result.SuccessfulFixes,
		"failed_issues":    result.FailedIssues,
		"total_issues":     result.TotalIssues,
		"success_count":    result.SuccessCount,
		"failure_count":    result.FailureCount,
		"summary":          CreateBatchFixSummary(result),
	}, nil
}

// validateFixReasoner is the reasoner for validating a single fix
func validateFixReasoner(ctx context.Context, fixer *Fixer, input map[string]any) (any, error) {
	// Extract patch
	patchData, ok := input["patch"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("patch must be an object")
	}

	// Convert to CodePatch
	patch := &CodePatch{
		IssueID:      patchData["issue_id"].(string),
		FilePath:     patchData["file_path"].(string),
		Language:     patchData["language"].(string),
		OriginalCode: patchData["original_code"].(string),
		FixedCode:    patchData["fixed_code"].(string),
		Description:  patchData["description"].(string),
	}

	if line, ok := patchData["line"].(float64); ok {
		patch.Line = int(line)
	}

	// Get validation config
	config := DefaultValidationConfig()
	if configInput, ok := input["config"].(map[string]any); ok {
		if enableSyntax, ok := configInput["enable_syntax_check"].(bool); ok {
			config.EnableSyntaxCheck = enableSyntax
		}
		if enableLinting, ok := configInput["enable_linting"].(bool); ok {
			config.EnableLinting = enableLinting
		}
		if enableFormatting, ok := configInput["enable_formatting"].(bool); ok {
			config.EnableFormatting = enableFormatting
		}
		if enableSecurity, ok := configInput["enable_security_scan"].(bool); ok {
			config.EnableSecurityScan = enableSecurity
		}
		if autoFormat, ok := configInput["auto_format"].(bool); ok {
			config.AutoFormat = autoFormat
		}
	}

	// Validate fix
	result, err := ValidateFix(ctx, patch, config)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"is_valid": result.IsValid,
		"errors":   result.Errors,
		"warnings": result.Warnings,
	}, nil
}
