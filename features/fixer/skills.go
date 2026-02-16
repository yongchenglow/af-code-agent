package fixer

import (
	"context"
	"fmt"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
)

// RegisterSkills registers all fixer skills (deterministic functions)
// Note: Go SDK uses RegisterReasoner for both skills and reasoners
func RegisterSkills(app *agent.Agent) {
	fixer := NewFixer(app)

	// validate_fix is deterministic validation logic
	app.RegisterReasoner("validate_fix",
		func(ctx context.Context, input map[string]any) (any, error) {
			return validateFixSkill(ctx, fixer, input)
		},
		agent.WithDescription("[SKILL] Validates a code fix against multiple criteria"))
}

// validateFixSkill is the skill for validating a single fix
func validateFixSkill(ctx context.Context, fixer *Fixer, input map[string]any) (any, error) {
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
