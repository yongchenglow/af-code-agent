package reviewer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/Agent-Field/agentfield/sdk/go/ai"
	"github.com/yourorg/github-code-agent/features/analyzer"
	"github.com/yourorg/github-code-agent/pkg/constants"
	"github.com/yourorg/github-code-agent/pkg/utils"
)

// Reviewer handles AI-powered code review
type Reviewer struct {
	agent *agent.Agent
}

// NewReviewer creates a new code reviewer
func NewReviewer(a *agent.Agent) *Reviewer {
	return &Reviewer{
		agent: a,
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

	// Use AgentField's built-in AI method
	response, err := r.agent.AI(aiCtx, prompt,
		ai.WithSystem(buildReviewSystemPrompt()),
		ai.WithTemperature(constants.DefaultAITemperature),
		ai.WithMaxTokens(constants.ReviewAIMaxTokens))

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("AI review timeout: request exceeded 10 minute limit: %w", err)
		}
		return nil, fmt.Errorf("AI review failed: %w", err)
	}

	// Parse AI response into structured review report
	report, err := parseReviewResponse(response.Text())
	if err != nil {
		return nil, fmt.Errorf("failed to parse review response: %w", err)
	}

	report.Model = response.Model

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

	// Use AI for security analysis with lower temperature for consistency
	response, err := r.agent.AI(aiCtx, prompt,
		ai.WithSystem(buildSecuritySystemPrompt()),
		ai.WithTemperature(constants.LowAITemperature),
		ai.WithMaxTokens(constants.SecurityAIMaxTokens))

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("security analysis timeout: request exceeded 10 minute limit: %w", err)
		}
		return nil, fmt.Errorf("security analysis failed: %w", err)
	}

	// Parse security issues from AI response
	issues, err := parseSecurityIssues(response.Text())
	if err != nil {
		return nil, fmt.Errorf("failed to parse security issues: %w", err)
	}

	return issues, nil
}

// buildReviewSystemPrompt creates the system prompt for code review
func buildReviewSystemPrompt() string {
	return `You are an expert code reviewer with deep knowledge of software engineering best practices.

Your role is to analyze code for:
- Bugs and potential runtime errors
- Security vulnerabilities (SQL injection, XSS, authentication issues, etc.)
- Performance problems (N+1 queries, inefficient algorithms, memory leaks)
- Maintainability concerns (code complexity, naming, structure)
- Best practice violations

Provide specific, actionable feedback. For each issue:
1. Clearly identify the problem
2. Explain why it's problematic
3. Suggest a concrete fix

Output your review as a JSON object with this structure:
{
  "issues": [
    {
      "file_path": "path/to/file.go",
      "line": 42,
      "severity": "High|Medium|Low|Critical",
      "category": "bug|security|performance|maintainability|style",
      "title": "Brief title",
      "description": "Detailed explanation of the issue",
      "suggestion": "How to fix it"
    }
  ],
  "summary": "Overall assessment of the code quality"
}

Severity guidelines:
- Critical: Security vulnerabilities, data loss risks, breaking changes
- High: Bugs that will cause failures, serious performance issues
- Medium: Code smells, maintainability concerns, minor bugs
- Low: Style issues, minor improvements`
}

// buildSecuritySystemPrompt creates the system prompt for security analysis
func buildSecuritySystemPrompt() string {
	return `You are a security expert specializing in application security and OWASP Top 10 vulnerabilities.

Analyze code for security issues including:
- Injection attacks (SQL, NoSQL, Command, LDAP)
- Cross-Site Scripting (XSS)
- Authentication and session management flaws
- Insecure direct object references
- Security misconfigurations
- Sensitive data exposure
- Missing access controls
- Cross-Site Request Forgery (CSRF)
- Using components with known vulnerabilities
- Insufficient logging and monitoring
- Hardcoded secrets, API keys, passwords
- Cryptographic failures

Output findings as a JSON array:
[
  {
    "file_path": "path/to/file.go",
    "line": 42,
    "severity": "Critical|High|Medium|Low",
    "type": "sql_injection|xss|secrets|etc",
    "title": "Brief title",
    "description": "Detailed security concern",
    "cwe": "CWE-89",
    "owasp": "A03:2021-Injection",
    "remediation": "How to fix the vulnerability"
  }
]

Only report actual security issues. Do not include general code quality concerns.`
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
