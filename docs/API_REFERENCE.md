# API Reference

Complete reference for all AgentField reasoners (APIs) in the GitHub Code Review Agent.

## Table of Contents

1. [Overview](#overview)
2. [Webhook Reasoners](#webhook-reasoners)
3. [Analyzer Reasoners](#analyzer-reasoners)
4. [Reviewer Reasoners](#reviewer-reasoners)
5. [Standards Reasoners](#standards-reasoners)
6. [Fixer Reasoners](#fixer-reasoners)
7. [GitOps Reasoners](#gitops-reasoners)
8. [Integration Patterns](#integration-patterns)
9. [Error Handling](#error-handling)

---

## Overview

The GitHub Code Review Agent uses AgentField's reasoner pattern. Each reasoner is a function that:

- Accepts `context.Context` and `map[string]any` input
- Returns `(any, error)`
- Is registered with a unique name
- Can be invoked locally or remotely

### Calling Reasoners

```go
// Local call (same agent)
result, err := app.CallLocal(ctx, "reasoner_name", input)

// Remote call (different agent/node)
result, err := app.Call(ctx, "agent.reasoner_name", input)
```

---

## Webhook Reasoners

### `webhook.handle_webhook`

Processes incoming GitHub webhook events.

**Input:**

```go
{
    "event_type": string,      // GitHub event type
    "payload": map[string]any, // Webhook payload
    "signature": string        // Webhook signature
}
```

**Output:**

```go
{
    "success": bool,
    "message": string
}
```

**Example:**

```go
input := map[string]any{
    "event_type": "pull_request",
    "payload": webhookPayload,
    "signature": "sha256=...",
}

result, err := app.CallLocal(ctx, "handle_webhook", input)
```

**Errors:**

- `ErrInvalidSignature`: Webhook signature validation failed
- `ErrUnsupportedEvent`: Event type not supported

---

### `webhook.handle_pr_opened`

Orchestrates the complete PR review workflow.

**Input:**

```go
{
    "owner": string,
    "repo": string,
    "pr_number": int,
    "installation_id": int64
}
```

**Output:**

```go
{
    "success": bool,
    "issues_found": int,
    "fixes_applied": int,
    "review_url": string
}
```

**Example:**

```go
input := map[string]any{
    "owner": "yourorg",
    "repo": "yourrepo",
    "pr_number": 123,
    "installation_id": 12345,
}

result, err := app.CallLocal(ctx, "handle_pr_opened", input)
```

---

## Analyzer Reasoners

### `analyzer.analyze_pr`

Analyzes all files in a pull request.

**Input:**

```go
{
    "owner": string,
    "repo": string,
    "pr_number": int
}
```

**Output:**

```go
{
    "files": []FileAnalysis,
    "total_files": int,
    "total_loc": int
}

type FileAnalysis struct {
    FilePath   string
    Language   string
    LOC        int
    Complexity int
    Functions  int
}
```

**Example:**

```go
input := map[string]any{
    "owner": "yourorg",
    "repo": "yourrepo",
    "pr_number": 123,
}

result, err := app.CallLocal(ctx, "analyze_pr", input)
data := result.(map[string]any)
files := data["files"].([]FileAnalysis)
```

---

### `analyzer.parse_code_structure`

Parses code into AST for analysis.

**Input:**

```go
{
    "content": string,
    "language": string
}
```

**Output:**

```go
{
    "ast": CodeAST,
    "language": string,
    "functions": []FunctionInfo,
    "classes": []ClassInfo
}
```

**Example:**

```go
input := map[string]any{
    "content": "package main\n\nfunc main() {}",
    "language": "go",
}

result, err := app.CallLocal(ctx, "parse_code_structure", input)
```

---

### `analyzer.calculate_complexity`

Calculates code complexity metrics.

**Input:**

```go
{
    "file_path": string,
    "content": string
}
```

**Output:**

```go
{
    "complexity": int,
    "loc": int,
    "functions": int,
    "cyclomatic": int
}
```

---

## Reviewer Reasoners

### `reviewer.review_code`

Performs AI-powered comprehensive code review.

**Input:**

```go
{
    "files": []FileAnalysis,
    "pr_context": map[string]any
}
```

**Output:**

```go
{
    "report": ReviewReport,
    "model": string
}

type ReviewReport struct {
    Issues []Issue
    Summary string
    Stats ReviewStats
}

type Issue struct {
    ID          string
    FilePath    string
    Line        int
    Severity    string  // Critical, High, Medium, Low
    Title       string
    Description string
    Suggestion  string
    Category    string  // bug, security, performance, style
}
```

**Example:**

```go
input := map[string]any{
    "files": analyzedFiles,
    "pr_context": map[string]any{
        "title": "Add new feature",
        "author": "developer",
    },
}

result, err := app.CallLocal(ctx, "review_code", input)
data := result.(map[string]any)
report := data["report"].(ReviewReport)
```

---

### `reviewer.detect_security_issues`

Performs specialized security vulnerability detection.

**Input:**

```go
{
    "files": []FileAnalysis
}
```

**Output:**

```go
{
    "security_issues": []SecurityIssue,
    "count": int
}

type SecurityIssue struct {
    Severity     string  // Critical, High, Medium, Low
    Line         int
    FilePath     string
    Title        string
    Description  string
    CVE          string  // If applicable
    Remediation  string
}
```

**Example:**

```go
input := map[string]any{
    "files": analyzedFiles,
}

result, err := app.CallLocal(ctx, "detect_security_issues", input)
```

---

## Standards Reasoners

### `standards.validate_standards`

Validates code against configured standards.

**Input:**

```go
{
    "files": []FileAnalysis,
    "config": Config
}
```

**Output:**

```go
{
    "violations": []Violation,
    "passed": bool
}

type Violation struct {
    Rule        string
    FilePath    string
    Line        int
    Message     string
    Severity    string
    CanAutoFix  bool
}
```

**Example:**

```go
input := map[string]any{
    "files": analyzedFiles,
    "config": cfg,
}

result, err := app.CallLocal(ctx, "validate_standards", input)
```

**Built-in Rules:**

1. `max_line_length` - Line length limit
2. `function_length` - Function length limit
3. `complexity` - Cyclomatic complexity limit
4. `naming_convention` - Naming style enforcement
5. `require_docstring` - Documentation requirement

---

## Fixer Reasoners

### `fixer.generate_fixes_with_validation`

Generates and validates code fixes with retry loop.

**Input:**

```go
{
    "issues": []Issue,
    "files": []FileAnalysis
}
```

**Output:**

```go
{
    "validated_patches": []CodePatch,
    "count": int,
    "failed": []FailedFix
}

type CodePatch struct {
    IssueID     string
    FilePath    string
    OldCode     string
    NewCode     string
    Language    string
    StartLine   int
    EndLine     int
    Validated   bool
    Attempts    int
}

type FailedFix struct {
    IssueID string
    Reason  string
    Attempts int
}
```

**Example:**

```go
input := map[string]any{
    "issues": reviewIssues,
    "files": analyzedFiles,
}

result, err := app.CallLocal(ctx, "generate_fixes_with_validation", input)
data := result.(map[string]any)
patches := data["validated_patches"].([]CodePatch)
```

**Validation Loop:**

1. Generate fix using AI
2. Validate syntax
3. Run linters
4. Check formatting
5. Scan for security issues
6. If invalid, retry (max 3 attempts)
7. Return validated patch or failure

---

### `fixer.validate_fix`

Validates a single code fix.

**Input:**

```go
{
    "patch": CodePatch,
    "issue": Issue
}
```

**Output:**

```go
{
    "is_valid": bool,
    "errors": []string,
    "warnings": []string
}
```

**Validation Checks:**

- Syntax correctness
- Linting compliance
- Code formatting
- Security vulnerabilities
- Addresses original issue

**Example:**

```go
input := map[string]any{
    "patch": generatedPatch,
    "issue": originalIssue,
}

result, err := app.CallLocal(ctx, "validate_fix", input)
```

---

## GitOps Reasoners

### `gitops.create_branch`

Creates a new Git branch for fixes.

**Input:**

```go
{
    "owner": string,
    "repo": string,
    "base_branch": string,
    "new_branch": string
}
```

**Output:**

```go
{
    "branch_name": string,
    "sha": string
}
```

**Example:**

```go
input := map[string]any{
    "owner": "yourorg",
    "repo": "yourrepo",
    "base_branch": "feature-branch",
    "new_branch": "agent-fixes/pr-123",
}

result, err := app.CallLocal(ctx, "create_branch", input)
```

---

### `gitops.apply_patches`

Applies code patches and creates a commit.

**Input:**

```go
{
    "owner": string,
    "repo": string,
    "branch": string,
    "patches": []CodePatch,
    "commit_message": string
}
```

**Output:**

```go
{
    "commit_sha": string,
    "files_changed": int
}
```

**Example:**

```go
input := map[string]any{
    "owner": "yourorg",
    "repo": "yourrepo",
    "branch": "agent-fixes/pr-123",
    "patches": validatedPatches,
    "commit_message": "🤖 Auto-fix: 3 issues resolved",
}

result, err := app.CallLocal(ctx, "apply_patches", input)
```

---

### `gitops.create_pull_request`

Creates a GitHub pull request.

**Input:**

```go
{
    "owner": string,
    "repo": string,
    "head_branch": string,
    "base_branch": string,
    "title": string,
    "body": string
}
```

**Output:**

```go
{
    "pr_number": int,
    "pr_url": string
}
```

**Example:**

```go
input := map[string]any{
    "owner": "yourorg",
    "repo": "yourrepo",
    "head_branch": "agent-fixes/pr-123",
    "base_branch": "feature-branch",
    "title": "🤖 Automated fixes for PR #123",
    "body": "## Summary\n- Fixed 3 issues\n...",
}

result, err := app.CallLocal(ctx, "create_pull_request", input)
```

---

### `gitops.add_review_comment`

Adds a code review comment to a PR.

**Input:**

```go
{
    "owner": string,
    "repo": string,
    "pr_number": int,
    "comment": ReviewComment
}

type ReviewComment struct {
    FilePath  string
    Line      int
    Body      string
    Severity  string
}
```

**Output:**

```go
{
    "comment_id": int64
}
```

**Example:**

```go
input := map[string]any{
    "owner": "yourorg",
    "repo": "yourrepo",
    "pr_number": 123,
    "comment": ReviewComment{
        FilePath: "main.go",
        Line: 42,
        Body: "🔴 **Critical**: Potential nil pointer...",
        Severity: "Critical",
    },
}

result, err := app.CallLocal(ctx, "add_review_comment", input)
```

---

### `gitops.update_review_comment`

Updates an existing review comment with fix link.

**Input:**

```go
{
    "owner": string,
    "repo": string,
    "comment_id": int64,
    "fix_link": string
}
```

**Output:**

```go
{
    "success": bool
}
```

**Example:**

```go
input := map[string]any{
    "owner": "yourorg",
    "repo": "yourrepo",
    "comment_id": 98765,
    "fix_link": "[View fix PR #124](https://github.com/org/repo/pull/124)",
}

result, err := app.CallLocal(ctx, "update_review_comment", input)
```

---

### `gitops.post_review_with_fixes`

Orchestrates the complete review + fix workflow.

**Input:**

```go
{
    "owner": string,
    "repo": string,
    "pr_number": int,
    "issues": []Issue,
    "mode": string  // "yolo" or "safe"
}
```

**Output:**

```go
{
    "comments_posted": int,
    "fixes_applied": int,
    "fix_pr_number": int,     // Safe mode only
    "fix_commit_sha": string   // YOLO mode only
}
```

**Workflow:**

1. Post initial review comments
2. Generate and validate fixes
3. Apply fixes based on mode:
   - **Safe**: Create PR with fixes
   - **YOLO**: Push directly to branch
4. Update comments with fix links

**Example:**

```go
input := map[string]any{
    "owner": "yourorg",
    "repo": "yourrepo",
    "pr_number": 123,
    "issues": reviewIssues,
    "mode": "safe",
}

result, err := app.CallLocal(ctx, "post_review_with_fixes", input)
```

---

## Integration Patterns

### Pattern 1: Simple Review Workflow

```go
// 1. Analyze PR
analyzeResult, _ := app.CallLocal(ctx, "analyze_pr", map[string]any{
    "owner": "org",
    "repo": "repo",
    "pr_number": 123,
})

// 2. Review code
reviewResult, _ := app.CallLocal(ctx, "review_code", map[string]any{
    "files": analyzeResult["files"],
})

// 3. Post comments
issues := reviewResult["report"].(ReviewReport).Issues
for _, issue := range issues {
    app.CallLocal(ctx, "add_review_comment", map[string]any{
        "owner": "org",
        "repo": "repo",
        "pr_number": 123,
        "comment": issue,
    })
}
```

### Pattern 2: Review + Auto-Fix

```go
// 1. Analyze + Review (parallel)
analyzeResult, _ := app.CallLocal(ctx, "analyze_pr", input)
reviewResult, _ := app.CallLocal(ctx, "review_code", input)

// 2. Generate and validate fixes
fixResult, _ := app.CallLocal(ctx, "generate_fixes_with_validation", map[string]any{
    "issues": reviewResult["issues"],
    "files": analyzeResult["files"],
})

// 3. Apply fixes
app.CallLocal(ctx, "post_review_with_fixes", map[string]any{
    "issues": reviewResult["issues"],
    "patches": fixResult["validated_patches"],
    "mode": "safe",
})
```

### Pattern 3: Security-Focused Review

```go
// 1. Analyze
analyzeResult, _ := app.CallLocal(ctx, "analyze_pr", input)

// 2. Security scan only
securityResult, _ := app.CallLocal(ctx, "detect_security_issues", map[string]any{
    "files": analyzeResult["files"],
})

// 3. Block PR if critical issues found
securityIssues := securityResult["security_issues"].([]SecurityIssue)
for _, issue := range securityIssues {
    if issue.Severity == "Critical" {
        // Post blocking comment
        // Fail CI check
    }
}
```

---

## Error Handling

### Standard Errors

All reasoners return Go errors following these conventions:

```go
// Validation errors
ErrInvalidInput       = errors.New("invalid input")
ErrMissingRequired    = errors.New("required field missing")

// GitHub API errors
ErrGitHubAPI          = errors.New("GitHub API error")
ErrRateLimit          = errors.New("rate limit exceeded")
ErrPermissionDenied   = errors.New("permission denied")

// AI errors
ErrAIFailure          = errors.New("AI request failed")
ErrAITimeout          = errors.New("AI request timeout")

// Validation errors
ErrValidationFailed   = errors.New("validation failed")
ErrMaxAttemptsReached = errors.New("max retry attempts reached")
```

### Error Handling Pattern

```go
result, err := app.CallLocal(ctx, "review_code", input)
if err != nil {
    switch {
    case errors.Is(err, ErrRateLimit):
        // Wait and retry
        time.Sleep(time.Minute)
        result, err = app.CallLocal(ctx, "review_code", input)

    case errors.Is(err, ErrAIFailure):
        // Log and notify
        log.Errorf("AI failure: %v", err)
        notifyAdmin(err)

    default:
        // Generic error handling
        return fmt.Errorf("review failed: %w", err)
    }
}
```

---

## Performance Optimization

### Parallel Execution

Use the performance package for parallel reasoner calls:

```go
import "github.com/yourorg/github-code-agent/pkg/performance"

executor := performance.NewParallelExecutor(5)

tasks := []performance.Task{
    func(ctx context.Context) (any, error) {
        return app.CallLocal(ctx, "analyze_pr", input)
    },
    func(ctx context.Context) (any, error) {
        return app.CallLocal(ctx, "validate_standards", input)
    },
}

results, err := executor.Execute(ctx, tasks)
```

### Caching

Use caching for expensive operations:

```go
import "github.com/yourorg/github-code-agent/pkg/performance"

cache := performance.NewAIResponseCache(15 * time.Minute)

// Check cache first
response, exists := cache.Get("review", prompt)
if !exists {
    // Call AI reasoner
    result, _ := app.CallLocal(ctx, "review_code", input)
    response = result["response"].(string)

    // Store in cache
    cache.Set("review", prompt, response)
}
```

---

## See Also

- [User Guide](USER_GUIDE.md)
- [Configuration Reference](CONFIGURATION_REFERENCE.md)
- [AgentField Documentation](https://www.agentfield.ai)
