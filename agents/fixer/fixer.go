package fixer

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/Agent-Field/agentfield/sdk/go/ai"
	"github.com/yourorg/github-code-agent/pkg/constants"
	"github.com/yourorg/github-code-agent/pkg/utils"
)

//go:embed prompts/system.md
var fixSystemPrompt string

// GenerateFixesWithValidation generates and validates fixes with retry loop
func GenerateFixesWithValidation(ctx context.Context, agentInstance *agent.Agent, issues []map[string]any, files []map[string]any, config *ValidationConfig) (*BatchFixResult, error) {
	if config == nil {
		config = DefaultValidationConfig()
	}

	result := &BatchFixResult{
		SuccessfulFixes: []*CodePatch{},
		FailedIssues:    []string{},
		TotalIssues:     len(issues),
	}

	for _, issueData := range issues {
		issueID := issueData["id"].(string)

		// Attempt to fix with validation loop
		fixResult := attemptFixWithRetry(ctx, agentInstance, issueData, files, config)

		if fixResult.Success {
			result.SuccessfulFixes = append(result.SuccessfulFixes, fixResult.Patch)
			result.SuccessCount++
		} else {
			result.FailedIssues = append(result.FailedIssues, issueID)
			result.FailureCount++

			// Log failure
			if agentInstance != nil {
				agentInstance.Notef(ctx, "Failed to generate valid fix for issue %s after %d attempts: %s",
					issueID, fixResult.TotalAttempts, fixResult.FinalError)
			}
		}
	}

	return result, nil
}

// attemptFixWithRetry attempts to fix an issue with retry loop
func attemptFixWithRetry(ctx context.Context, agentInstance *agent.Agent, issue map[string]any, files []map[string]any, config *ValidationConfig) *FixResult {
	issueID := issue["id"].(string)
	result := &FixResult{
		IssueID:  issueID,
		Success:  false,
		Attempts: []*FixAttempt{},
	}

	var previousErrors []string

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		attemptStart := time.Now()

		// Generate fix
		patch, err := generateSingleFix(ctx, agentInstance, issue, files, previousErrors)
		if err != nil {
			result.FinalError = fmt.Sprintf("fix generation failed: %v", err)
			result.TotalAttempts = attempt
			return result
		}

		// Validate the fix
		validation, err := ValidateFix(ctx, patch, config)
		if err != nil {
			result.FinalError = fmt.Sprintf("validation failed: %v", err)
			result.TotalAttempts = attempt
			return result
		}

		// Record attempt
		fixAttempt := &FixAttempt{
			AttemptNumber: attempt,
			Patch:         patch,
			Validation:    validation,
			Timestamp:     attemptStart,
		}
		result.Attempts = append(result.Attempts, fixAttempt)

		// Check if fix is valid
		if validation.IsValid {
			result.Success = true
			result.Patch = patch
			result.TotalAttempts = attempt

			if agentInstance != nil {
				agentInstance.Notef(ctx, "Successfully generated valid fix for issue %s on attempt %d",
					issueID, attempt)
			}
			return result
		}

		// Fix failed validation, collect errors for next attempt
		previousErrors = validation.Errors

		if attempt < config.MaxAttempts {
			if agentInstance != nil {
				agentInstance.Notef(ctx, "Fix attempt %d/%d failed validation for issue %s, retrying with context...",
					attempt, config.MaxAttempts, issueID)
			}
		} else {
			result.FinalError = fmt.Sprintf("max attempts reached with errors: %s",
				strings.Join(validation.Errors, "; "))
		}
	}

	result.TotalAttempts = config.MaxAttempts
	return result
}

// generateSingleFix creates a fix with context from previous validation errors
func generateSingleFix(ctx context.Context, agentInstance *agent.Agent, issue map[string]any, files []map[string]any, previousErrors []string) (*CodePatch, error) {
	// Extract issue details
	filePath := issue["file_path"].(string)
	line := int(issue["line"].(float64))
	description := issue["description"].(string)

	// Find the file content
	var fileContent string
	var language string
	for _, file := range files {
		if file["path"].(string) == filePath {
			fileContent = file["content"].(string)
			language = file["language"].(string)
			break
		}
	}

	if fileContent == "" {
		return nil, fmt.Errorf("file content not found for %s", filePath)
	}

	// Extract relevant code section (context around the issue)
	originalCode := utils.ExtractCodeSection(fileContent, line, constants.MaxFileContextLines)

	// Build fix prompt
	prompt := buildFixPrompt(issue, originalCode, language)

	// Include previous validation errors if any
	if len(previousErrors) > 0 {
		prompt += fmt.Sprintf("\n\nPrevious fix attempt had these validation issues:\n%s\n"+
			"Please generate a fix that avoids these problems.",
			strings.Join(previousErrors, "\n"))
	}

	// Create context with timeout for fix generation
	aiCtx, cancel := context.WithTimeout(ctx, constants.DefaultAITimeout)
	defer cancel()

	// Call AI to generate fix
	response, err := agentInstance.AI(aiCtx, prompt,
		ai.WithSystem(fixSystemPrompt),
		ai.WithTemperature(constants.LowAITemperature),
		ai.WithMaxTokens(constants.FixerAIMaxTokens))

	if err != nil {
		return nil, fmt.Errorf("AI fix generation failed: %w", err)
	}

	// Parse response to extract fixed code
	fixedCode := utils.ExtractCodeFromResponse(response.Text())

	patch := &CodePatch{
		IssueID:      issue["id"].(string),
		FilePath:     filePath,
		Language:     language,
		OriginalCode: originalCode,
		FixedCode:    fixedCode,
		Description:  description,
		Line:         line,
	}

	return patch, nil
}

// buildFixPrompt builds the prompt for fix generation
func buildFixPrompt(issue map[string]any, originalCode, language string) string {
	title := issue["title"].(string)
	description := issue["description"].(string)
	suggestion := ""
	if s, ok := issue["suggestion"].(string); ok {
		suggestion = s
	}

	prompt := fmt.Sprintf(`Fix the following %s code issue:

**Issue**: %s
**Description**: %s`, language, title, description)

	if suggestion != "" {
		prompt += fmt.Sprintf("\n**Suggested Fix**: %s", suggestion)
	}

	prompt += fmt.Sprintf(`

**Original Code:**
%s

**Instructions:**
1. Generate ONLY the fixed code, no explanations
2. Maintain the same code structure and indentation
3. Ensure the fix is minimal and targeted
4. The code must be syntactically correct
5. Follow %s formatting conventions
6. Do not introduce new security vulnerabilities
7. Output only the complete fixed code block

**Fixed Code:**`, "```"+language+"\n"+originalCode+"\n```", language)

	return prompt
}

// CreateBatchFixSummary creates a summary of batch fix results
func CreateBatchFixSummary(result *BatchFixResult) string {
	if result.SuccessCount == 0 && result.FailureCount == 0 {
		return "No fixes were attempted."
	}

	summary := "Fix Generation Summary:\n"
	summary += fmt.Sprintf("- Total issues: %d\n", result.TotalIssues)
	summary += fmt.Sprintf("- Successfully fixed: %d\n", result.SuccessCount)
	summary += fmt.Sprintf("- Failed to fix: %d\n", result.FailureCount)

	if result.SuccessCount > 0 {
		summary += "\nSuccessful fixes:\n"
		for i, patch := range result.SuccessfulFixes {
			summary += fmt.Sprintf("%d. %s (line %d)\n", i+1, patch.FilePath, patch.Line)
		}
	}

	if len(result.FailedIssues) > 0 {
		summary += "\nFailed to fix:\n"
		for i, issueID := range result.FailedIssues {
			summary += fmt.Sprintf("%d. Issue %s\n", i+1, issueID)
		}
	}

	return summary
}
