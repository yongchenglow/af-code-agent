package reviewer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "embed"
	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/Agent-Field/agentfield/sdk/go/ai"
	"github.com/yourorg/github-code-agent/agents/analyzer"
	"github.com/yourorg/github-code-agent/pkg/circuitbreaker"
	"github.com/yourorg/github-code-agent/pkg/constants"
	"github.com/yourorg/github-code-agent/pkg/logger"
	"github.com/yourorg/github-code-agent/pkg/retry"
	"github.com/yourorg/github-code-agent/pkg/utils"
)

//go:embed prompts/system.md
var reviewSystemPrompt string

//go:embed prompts/security.md
var securitySystemPrompt string

// ReviewerConfig holds configuration for the Reviewer
type ReviewerConfig struct {
	// AICircuitBreaker is the circuit breaker for AI calls
	AICircuitBreaker *circuitbreaker.CircuitBreaker
	// AIRetryer is the retryer for AI calls
	AIRetryer *retry.Retryer
	// SecurityCircuitBreaker is the circuit breaker for security analysis calls
	SecurityCircuitBreaker *circuitbreaker.CircuitBreaker
	// SecurityRetryer is the retryer for security analysis calls
	SecurityRetryer *retry.Retryer
}

// Reviewer handles AI-powered code review
type Reviewer struct {
	agent                  *agent.Agent
	aiCircuitBreaker       *circuitbreaker.CircuitBreaker
	aiRetryer              *retry.Retryer
	securityCircuitBreaker *circuitbreaker.CircuitBreaker
	securityRetryer        *retry.Retryer
}

// NewReviewer creates a new code reviewer with default configuration
func NewReviewer(a *agent.Agent) *Reviewer {
	return NewReviewerWithConfig(a, ReviewerConfig{})
}

// NewReviewerWithConfig creates a new code reviewer with custom configuration
func NewReviewerWithConfig(a *agent.Agent, config ReviewerConfig) *Reviewer {
	// Use provided circuit breakers or create defaults
	aiCB := config.AICircuitBreaker
	if aiCB == nil {
		aiCB = circuitbreaker.New(circuitbreaker.Config{
			Threshold:           5,
			Timeout:             1 * time.Minute,
			HalfOpenMaxRequests: 3,
			Name:                "ai-service",
		})
	}

	securityCB := config.SecurityCircuitBreaker
	if securityCB == nil {
		securityCB = circuitbreaker.New(circuitbreaker.Config{
			Threshold:           5,
			Timeout:             1 * time.Minute,
			HalfOpenMaxRequests: 3,
			Name:                "security-analysis",
		})
	}

	// Use provided retryers or create defaults
	aiRetryer := config.AIRetryer
	if aiRetryer == nil {
		aiRetryer = retry.New(retry.ConfigPreset{}.ForAI())
	}

	securityRetryer := config.SecurityRetryer
	if securityRetryer == nil {
		securityRetryer = retry.New(retry.ConfigPreset{}.ForAI())
	}

	return &Reviewer{
		agent:                  a,
		aiCircuitBreaker:       aiCB,
		aiRetryer:              aiRetryer,
		securityCircuitBreaker: securityCB,
		securityRetryer:        securityRetryer,
	}
}

// ReviewCode performs AI-powered comprehensive code review
func (r *Reviewer) ReviewCode(ctx context.Context, files []*analyzer.FileChange, prContext map[string]interface{}) (*ReviewReport, error) {
	// Filter out files that shouldn't be reviewed
	reviewableFiles := filterReviewableFiles(files)

	if len(reviewableFiles) == 0 {
		return &ReviewReport{
			Issues:           []*Issue{},
			Summary:          "No reviewable files found",
			TotalIssues:      0,
			IssuesBySeverity: make(map[string]int),
			IssuesByCategory: make(map[string]int),
		}, nil
	}

	// Build review prompt
	prompt := buildReviewPrompt(reviewableFiles, prContext)

	// Create context with timeout for large reviews
	aiCtx, cancel := context.WithTimeout(ctx, constants.DefaultAITimeout)
	defer cancel()

	// Get logger with context
	log := logger.Default().WithContext(ctx)

	// Execute with retry and circuit breaker
	var response interface{}
	
	err := r.aiRetryer.Execute(aiCtx, func() error {
		return r.aiCircuitBreaker.Execute(func() error {
			resp, err := r.agent.AI(aiCtx, prompt,
				ai.WithSystem(reviewSystemPrompt),
				ai.WithTemperature(constants.DefaultAITemperature),
				ai.WithMaxTokens(constants.ReviewAIMaxTokens))

			if err != nil {
				log.Warn("AI review attempt failed",
					"error", err,
					"circuit_state", r.aiCircuitBreaker.State().String())
				return err
			}

			response = resp
			return nil
		})
	})

	if err != nil {
		// Check for specific error types
		if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
			log.Error("AI review blocked by circuit breaker",
				"state", r.aiCircuitBreaker.State().String(),
				"metrics", r.aiCircuitBreaker.Metrics())
			return nil, fmt.Errorf("AI service unavailable (circuit breaker open): %w", err)
		}
		if errors.Is(err, retry.ErrMaxAttemptsExceeded) {
			log.Error("AI review failed after all retry attempts")
			return nil, fmt.Errorf("AI review failed after retries: %w", err)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("AI review timeout: request exceeded 10 minute limit: %w", err)
		}
		return nil, fmt.Errorf("AI review failed: %w", err)
	}

	// Type assert the response
	aiResponse, ok := response.(interface {
		Text() string
		Model() string
	})
	if !ok {
		return nil, fmt.Errorf("unexpected response type from AI")
	}

	// Parse AI response into structured review report
	report, err := parseReviewResponse(aiResponse.Text())
	if err != nil {
		return nil, fmt.Errorf("failed to parse review response: %w", err)
	}

	report.Model = aiResponse.Model()

	// Log success with metrics
	metrics := r.aiCircuitBreaker.Metrics()
	log.Debug("AI review completed",
		"files_reviewed", len(reviewableFiles),
		"issues_found", report.TotalIssues,
		"circuit_failures", metrics.FailureCount,
		"circuit_state", metrics.State.String())

	return report, nil
}

// DetectSecurityIssues performs specialized security vulnerability detection
func (r *Reviewer) DetectSecurityIssues(ctx context.Context, files []*analyzer.FileChange) ([]*SecurityIssue, error) {
	// Filter to code files only
	codeFiles := filterCodeFiles(files)

	if len(codeFiles) == 0 {
		return []*SecurityIssue{}, nil
	}

	// Build security-focused prompt
	prompt := buildSecurityPrompt(codeFiles)

	// Create context with timeout for security analysis
	aiCtx, cancel := context.WithTimeout(ctx, constants.DefaultAITimeout)
	defer cancel()

	// Get logger with context
	log := logger.Default().WithContext(ctx)

	// Execute with retry and circuit breaker
	var response interface{}
	err := r.securityRetryer.Execute(aiCtx, func() error {
		return r.securityCircuitBreaker.Execute(func() error {
			resp, err := r.agent.AI(aiCtx, prompt,
				ai.WithSystem(securitySystemPrompt),
				ai.WithTemperature(constants.LowAITemperature),
				ai.WithMaxTokens(constants.SecurityAIMaxTokens))

			if err != nil {
				log.Warn("Security analysis attempt failed",
					"error", err,
					"circuit_state", r.securityCircuitBreaker.State().String())
				return err
			}

			response = resp
			return nil
		})
	})

	if err != nil {
		// Check for specific error types
		if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
			log.Error("Security analysis blocked by circuit breaker",
				"state", r.securityCircuitBreaker.State().String(),
				"metrics", r.securityCircuitBreaker.Metrics())
			return nil, fmt.Errorf("security analysis unavailable (circuit breaker open): %w", err)
		}
		if errors.Is(err, retry.ErrMaxAttemptsExceeded) {
			log.Error("Security analysis failed after all retry attempts")
			return nil, fmt.Errorf("security analysis failed after retries: %w", err)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("security analysis timeout: request exceeded 10 minute limit: %w", err)
		}
		return nil, fmt.Errorf("security analysis failed: %w", err)
	}

	// Type assert the response
	aiResponse, ok := response.(interface {
		Text() string
		Model() string
	})
	if !ok {
		return nil, fmt.Errorf("unexpected response type from AI")
	}

	// Parse security issues from AI response
	issues, err := parseSecurityIssues(aiResponse.Text())
	if err != nil {
		return nil, fmt.Errorf("failed to parse security issues: %w", err)
	}

	// Log success with metrics
	metrics := r.securityCircuitBreaker.Metrics()
	log.Debug("Security analysis completed",
		"files_analyzed", len(codeFiles),
		"issues_found", len(issues),
		"circuit_failures", metrics.FailureCount,
		"circuit_state", metrics.State.String())

	return issues, nil
}

// buildReviewPrompt constructs the review prompt from file changes
func buildReviewPrompt(files []*analyzer.FileChange, prContext map[string]interface{}) string {
	var builder strings.Builder

	builder.WriteString("Please review the following code changes:\n\n")

	// Add PR context if available
	if title, ok := prContext["title"].(string); ok {
		builder.WriteString(fmt.Sprintf("PR Title: %s\n", title))
	}
	if description, ok := prContext["description"].(string); ok && description != "" {
		builder.WriteString(fmt.Sprintf("PR Description: %s\n", description))
	}
	builder.WriteString("\n")

	// Add file changes
	for i, file := range files {
		if i >= constants.MaxReviewableFiles {
			builder.WriteString(fmt.Sprintf("\n... and %d more files\n", len(files)-constants.MaxReviewableFiles))
			break
		}

		builder.WriteString(fmt.Sprintf("File: %s (Language: %s, +%d -%d)\n",
			file.Filename, file.Language, file.Additions, file.Deletions))

		// Use patch (diff) instead of full content when available - much smaller
		if file.Patch != "" {
			patch := utils.TruncateContent(file.Patch, constants.MaxPatchLength)
			builder.WriteString("Diff:\n```diff\n")
			builder.WriteString(patch)
			builder.WriteString("\n```\n\n")
		} else if file.Content != "" {
			content := utils.TruncateContent(file.Content, constants.MaxContentLength)
			builder.WriteString("```" + file.Language + "\n")
			builder.WriteString(content)
			builder.WriteString("\n```\n\n")
		}
	}

	return builder.String()
}

// buildSecurityPrompt constructs the security analysis prompt
func buildSecurityPrompt(files []*analyzer.FileChange) string {
	var builder strings.Builder

	builder.WriteString("Analyze the following code for security vulnerabilities:\n\n")

	for i, file := range files {
		if i >= constants.MaxReviewableFiles {
			break
		}

		builder.WriteString(fmt.Sprintf("File: %s (Language: %s)\n", file.Filename, file.Language))

		if file.Content != "" {
			content := utils.TruncateContent(file.Content, constants.MaxSecurityContentLength)
			builder.WriteString("```" + file.Language + "\n")
			builder.WriteString(content)
			builder.WriteString("\n```\n\n")
		}
	}

	return builder.String()
}

// parseReviewResponse parses the AI response into a ReviewReport
func parseReviewResponse(text string) (*ReviewReport, error) {
	// Try to extract JSON from the response
	jsonText := utils.ExtractJSON(text)

	var response struct {
		Issues  []*Issue `json:"issues"`
		Summary string   `json:"summary"`
	}

	if err := json.Unmarshal([]byte(jsonText), &response); err != nil {
		// If JSON parsing fails, create a simple report
		return &ReviewReport{
			Issues:           []*Issue{},
			Summary:          "Unable to parse AI response",
			TotalIssues:      0,
			IssuesBySeverity: make(map[string]int),
			IssuesByCategory: make(map[string]int),
		}, nil
	}

	// Build report with statistics
	report := &ReviewReport{
		Issues:           response.Issues,
		Summary:          response.Summary,
		TotalIssues:      len(response.Issues),
		IssuesBySeverity: make(map[string]int),
		IssuesByCategory: make(map[string]int),
	}

	// Calculate statistics
	for _, issue := range response.Issues {
		// Assign unique ID if not present
		if issue.ID == "" {
			issue.ID = fmt.Sprintf("%s-%d", issue.FilePath, issue.Line)
		}

		// Count by severity
		report.IssuesBySeverity[issue.Severity]++

		// Count by category
		report.IssuesByCategory[issue.Category]++
	}

	return report, nil
}

// parseSecurityIssues parses security issues from AI response
func parseSecurityIssues(text string) ([]*SecurityIssue, error) {
	jsonText := utils.ExtractJSON(text)

	var issues []*SecurityIssue

	if err := json.Unmarshal([]byte(jsonText), &issues); err != nil {
		// If JSON parsing fails, return empty list
		return []*SecurityIssue{}, nil
	}

	// Assign IDs if missing
	for i, issue := range issues {
		if issue.ID == "" {
			issue.ID = fmt.Sprintf("sec-%s-%d", issue.FilePath, i)
		}
	}

	return issues, nil
}

// filterReviewableFiles filters out files that shouldn't be reviewed
func filterReviewableFiles(files []*analyzer.FileChange) []*analyzer.FileChange {
	var reviewable []*analyzer.FileChange

	for _, file := range files {
		// Skip deleted files
		if file.Status == constants.FileStatusRemoved {
			continue
		}

		// Skip binary files and common non-code files
		if utils.ShouldSkipFile(file.Filename) {
			continue
		}

		reviewable = append(reviewable, file)
	}

	return reviewable
}

// filterCodeFiles filters to only code files for security analysis
func filterCodeFiles(files []*analyzer.FileChange) []*analyzer.FileChange {
	var codeFiles []*analyzer.FileChange

	for _, file := range files {
		if file.Status == constants.FileStatusRemoved {
			continue
		}

		// Only include known programming languages
		if utils.IsCodeLanguage(file.Language) && file.Content != "" {
			codeFiles = append(codeFiles, file)
		}
	}

	return codeFiles
}
