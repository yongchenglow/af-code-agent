package prompts

import (
	"strings"

	"github.com/yourorg/github-code-agent/agents/analyzer"
)

// ReviewContext provides dynamic context to prompts
type ReviewContext struct {
	Files       []*analyzer.FileChange
	PRContext   map[string]any
	PriorIssues []*Issue
}

// Issue represents a code issue for prompting
type Issue struct {
	ID          string
	FilePath    string
	Line        int
	Severity    string
	Category    string
	Title       string
	Description string
	Suggestion  string
}

// ReviewerPrompts contains all reviewer prompt templates
type ReviewerPrompts struct {
	SystemPrompt string
	TaskPrompt   func(ctx ReviewContext) string
}

// NewReviewerPrompts creates reviewer prompts with templates
func NewReviewerPrompts() *ReviewerPrompts {
	return &ReviewerPrompts{
		SystemPrompt: systemReviewerPrompt,
		TaskPrompt:   buildReviewerTaskPrompt,
	}
}

// systemReviewerPrompt is the enhanced system prompt based on SWE-AF patterns
const systemReviewerPrompt = `You are a senior Code Reviewer who has reviewed millions of lines of production code. Your reviews catch bugs before they reach users, identify security vulnerabilities aligned with OWASP Top 10, and improve code maintainability without slowing down engineering velocity.

## Your Responsibilities
You own code quality at the PR level. Your review is the last line of defense before code reaches production. You balance rigor with pragmatism — catching real issues without nitpicking.

## What Makes You Exceptional
You study the codebase before reviewing. You understand established patterns, conventions, and architectural decisions. Your feedback feels like it comes from a teammate who knows the codebase, not an outsider imposing foreign standards.

## Your Quality Standards
- **Specificity**: Every issue names exact files, functions, and line numbers
- **Actionability**: Every issue includes a concrete fix suggestion
- **Prioritization**: Issues are ordered by severity (Critical → High → Medium → Low)
- **Evidence**: Security issues reference CWE and OWASP classifications
- **Signal-to-Noise**: You skip style nits that linters catch. Focus on what matters.

## Decision Framework
APPROVE when: code is correct, secure, and maintainable. Minor debt items are acceptable if tracked.
REQUEST_CHANGES when: bugs, security issues, or significant maintainability concerns exist.

## Output Format
Return a JSON object with:
{
  "issues": [...],
  "summary": "...",
  "recommendation": "APPROVE|REQUEST_CHANGES"
}`

// buildReviewerTaskPrompt builds the task prompt with context
func buildReviewerTaskPrompt(ctx ReviewContext) string {
	var b strings.Builder

	b.WriteString("## Code Review Task\n\n")

	// Add PR context
	if title, ok := ctx.PRContext["title"].(string); ok {
		b.WriteString("**PR Title**: ")
		b.WriteString(title)
		b.WriteString("\n\n")
	}

	if description, ok := ctx.PRContext["description"].(string); ok && description != "" {
		b.WriteString("**PR Description**: ")
		b.WriteString(description)
		b.WriteString("\n\n")
	}

	// Add files to review
	b.WriteString("## Files to Review\n\n")
	for i, file := range ctx.Files {
		if i >= 20 { // Limit for token efficiency
			b.WriteString("\n... and more files\n")
			break
		}

		b.WriteString("### File: ")
		b.WriteString(file.Filename)
		b.WriteString(" (")
		b.WriteString(file.Language)
		b.WriteString(")\n")
		b.WriteString("Changes: +")
		b.WriteString(string(rune(file.Additions)))
		b.WriteString(" -")
		b.WriteString(string(rune(file.Deletions)))
		b.WriteString("\n\n")

		if file.Patch != "" {
			b.WriteString("```diff\n")
			b.WriteString(file.Patch)
			b.WriteString("\n```\n\n")
		}
	}

	// Add prior issues context if any
	if len(ctx.PriorIssues) > 0 {
		b.WriteString("## Prior Issues to Consider\n\n")
		for _, issue := range ctx.PriorIssues {
			b.WriteString("- **")
			b.WriteString(issue.Title)
			b.WriteString("** (")
			b.WriteString(issue.FilePath)
			b.WriteString(":")
			b.WriteString(string(rune(issue.Line)))
			b.WriteString("): ")
			b.WriteString(issue.Description)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## Review Instructions\n\n")
	b.WriteString("Analyze the code changes above and identify:\n")
	b.WriteString("1. **Bugs**: Logic errors, null dereferences, type mismatches, edge cases\n")
	b.WriteString("2. **Security**: SQL injection, XSS, authentication flaws, secrets, input validation\n")
	b.WriteString("3. **Performance**: N+1 queries, inefficient algorithms, memory leaks\n")
	b.WriteString("4. **Maintainability**: Complex code, poor naming, missing tests\n")
	b.WriteString("\nFor each issue, provide:\n")
	b.WriteString("- Exact file path and line number\n")
	b.WriteString("- Clear description of the problem\n")
	b.WriteString("- Concrete fix suggestion\n")
	b.WriteString("- Severity level (Critical/High/Medium/Low)\n")

	return b.String()
}
