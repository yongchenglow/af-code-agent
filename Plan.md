# GitHub Code Review Agent - Implementation Plan

## Executive Summary

Build an autonomous GitHub code review agent using [AgentField](https://www.agentfield.ai) that automatically reviews pull requests, identifies issues, and applies fixes. The system will operate in two modes: **YOLO mode** (direct push to source branch) and **Safe mode** (creates new PR targeting source branch).

## 1. Architecture Overview

### 1.1 Multi-Agent System Design

The system will consist of specialized agents following AgentField's microservices approach:

```
┌─────────────────────────────────────────────────────────┐
│                   GitHub Webhook                         │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│            Orchestrator Agent (Main)                     │
│  - Receives webhook events                               │
│  - Routes to appropriate agents                          │
│  - Manages workflow state                                │
└────────┬────────────────────────────┬────────────────────┘
         │                            │
         ▼                            ▼
┌──────────────────┐        ┌──────────────────┐
│  Code Analyzer   │        │  Review Agent     │
│  Agent           │        │                   │
│  - Fetches PR    │◄───────┤  - Analyzes code  │
│  - Parses diff   │        │  - Generates      │
│  - Extracts      │        │    review         │
│    files         │        │    comments       │
└────────┬─────────┘        └──────────┬────────┘
         │                             │
         ▼                             ▼
┌──────────────────┐        ┌──────────────────┐
│  Standards       │        │  Fix Generator   │
│  Validator Agent │        │  Agent           │
│  - Coding style  │        │  - Creates fixes  │
│  - Docs checks   │        │  - Applies        │
│  - Best          │        │    patches        │
│    practices     │        │                   │
└──────────────────┘        └─────────┬─────────┘
                                      │
                                      ▼
                            ┌──────────────────┐
                            │  Git Operations  │
                            │  Agent           │
                            │  - Commits       │
                            │  - Pushes        │
                            │  - Creates PRs   │
                            └──────────────────┘
```

### 1.2 Technology Stack

- **Framework**: AgentField Go SDK (`github.com/Agent-Field/agentfield/sdk/go`)
- **Language**: Go 1.21+
- **GitHub Integration**: `google/go-github` + Webhooks
- **Code Analysis**: AST parsing, language-specific linters
- **AI Model**: DeepSeek Chat (deepseek-chat) via AgentField's built-in AI integration
- **AI Client**: AgentField native `agent.AI()` method with OpenRouter or OpenAI compatibility
- **Storage**: AgentField hierarchical memory (workflow, session, user, global scopes)
- **Deployment**: AgentField control plane with auto-scaling
- **Code Organization**: Feature-based structure with reasoner pattern

## 2. Project Structure (Feature-Based)

The codebase will follow a feature-based organization where each feature is self-contained:

```
github-code-agent/
├── cmd/
│   └── agent/
│       └── main.go                    # Entry point
├── features/
│   ├── webhook/                       # Webhook handling feature
│   │   ├── webhook.go                 # Skills and reasoners
│   │   ├── types.go                   # Feature-specific types
│   │   ├── validator.go               # Webhook validation
│   │   └── webhook_test.go
│   ├── analyzer/                      # Code analysis feature
│   │   ├── analyzer.go
│   │   ├── types.go
│   │   ├── parser.go                  # AST parsing
│   │   ├── metrics.go                 # Code metrics
│   │   └── analyzer_test.go
│   ├── reviewer/                      # Code review feature
│   │   ├── reviewer.go
│   │   ├── types.go
│   │   ├── security.go                # Security checks
│   │   ├── quality.go                 # Code quality
│   │   └── reviewer_test.go
│   ├── standards/                     # Standards validation feature
│   │   ├── standards.go
│   │   ├── types.go
│   │   ├── rules.go                   # Rule engine
│   │   ├── config.go                  # Config parser
│   │   └── standards_test.go
│   ├── fixer/                         # Fix generation feature
│   │   ├── fixer.go
│   │   ├── types.go
│   │   ├── generator.go               # Fix generation
│   │   ├── validator.go               # Fix validation
│   │   └── fixer_test.go
│   └── gitops/                        # Git operations feature
│       ├── gitops.go
│       ├── types.go
│       ├── branch.go                  # Branch operations
│       ├── commit.go                  # Commit operations
│       ├── pr.go                      # PR creation
│       └── gitops_test.go
├── pkg/
│   ├── github/                        # GitHub API client
│   │   └── client.go
│   ├── deepseek/                      # DeepSeek AI client wrapper
│   │   └── client.go
│   └── config/                        # Shared configuration
│       ├── config.go
│       └── types.go
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

**Benefits of Feature-Based Organization:**

- **High Cohesion**: All code related to a feature lives together
- **Easy Navigation**: Find all webhook-related code in `features/webhook/`
- **Independent Testing**: Each feature has its own test files
- **Clear Boundaries**: Features communicate through well-defined interfaces
- **Scalability**: Add new features without affecting existing ones

## 3. Agent Specifications

### 3.1 Webhook Handler (Orchestrator)

**Location**: `features/webhook/`

**Responsibilities:**

- Receive and validate GitHub webhook events
- Authenticate webhook signatures
- Route requests to appropriate agent workflows
- Manage execution state and error handling
- Coordinate responses back to GitHub

**Webhook Event Triggers:**

The agent will listen to the following GitHub webhook events:

1. **`pull_request.opened`** - When a new PR is created
2. **`pull_request.synchronize`** - When new commits are pushed to the PR
3. **`check_suite.completed`** - When CI/CD pipeline finishes (recommended)
4. **`workflow_run.completed`** - When GitHub Actions workflow completes
5. **`pull_request.reopened`** - When a closed PR is reopened

**Recommended Trigger Strategy:**

```yaml
# Preferred: Wait for CI/CD to complete before reviewing
trigger_on:
  - event: check_suite.completed
    conditions:
      - conclusion: success  # Only review if tests pass
      - conclusion: failure  # Review to help fix failures

  - event: pull_request.synchronize
    conditions:
      - wait_for_checks: true  # Wait for CI to finish
      - debounce: 30s  # Debounce multiple rapid commits

# Alternative: Immediate review on commit
trigger_on:
  - event: pull_request.synchronize
    immediate: true
```

**Why wait for CI/CD completion?**

- Avoid reviewing code that fails tests
- Get test results to inform review (e.g., "Tests are failing in auth.go:45")
- Reduce wasted review cycles on broken code
- Can suggest fixes based on test failures

**Reasoners:**

```go
// features/webhook/webhook.go
package webhook

import (
    "context"
    "github.com/Agent-Field/agentfield/sdk/go/agent"
)

// RegisterReasoners registers all webhook-related reasoners
func RegisterReasoners(app *agent.Agent) {
    // Main webhook handler
    app.RegisterReasoner("handle_webhook", HandleWebhook,
        agent.WithDescription("Processes incoming GitHub webhook events"))

    // PR-specific handler
    app.RegisterReasoner("handle_pr_opened", HandlePROpened,
        agent.WithDescription("Orchestrates code review workflow for new PRs"))
}

// HandleWebhook processes incoming webhook events
func HandleWebhook(ctx context.Context, input map[string]any) (any, error) {
    eventType := input["event_type"].(string)
    payload := input["payload"].(map[string]any)
    signature := input["signature"].(string)

    // Validate webhook signature
    if err := ValidateSignature(payload, signature); err != nil {
        return nil, err
    }

    // Route to appropriate handler
    return RouteEvent(ctx, eventType, payload)
}

// HandlePROpened is the main entry point for PR review workflow
func HandlePROpened(ctx context.Context, input map[string]any) (any, error) {
    prContext := ParsePRContext(input)

    // Orchestrate the review workflow using agent.Call()
    // This will be implemented to call other reasoners
    return &WorkflowResult{
        Success: true,
        Message: "Review workflow initiated",
    }, nil
}

// Helper functions (not reasoners)
func ValidateSignature(payload map[string]any, signature string) error {
    // HMAC validation logic
    return nil
}

func ParsePRContext(input map[string]any) *PRContext {
    // Extract PR metadata
    return &PRContext{}
}
```

### 3.2 Code Analyzer

**Location**: `features/analyzer/`

**Responsibilities:**

- Fetch PR content and file diffs from GitHub
- Parse code structure (AST analysis)
- Identify changed files and their relationships
- Extract code metrics (complexity, coverage, etc.)

**Reasoners:**

```go
// features/analyzer/analyzer.go
package analyzer

import (
    "context"
    "github.com/Agent-Field/agentfield/sdk/go/agent"
)

// RegisterReasoners registers all analyzer reasoners
func RegisterReasoners(app *agent.Agent) {
    app.RegisterReasoner("analyze_pr", AnalyzePR,
        agent.WithDescription("Analyzes PR files and extracts code metrics"))

    app.RegisterReasoner("parse_code_structure", ParseCodeStructure,
        agent.WithDescription("Parses code into AST for analysis"))

    app.RegisterReasoner("calculate_complexity", CalculateComplexity,
        agent.WithDescription("Calculates cyclomatic complexity and metrics"))
}

// AnalyzePR orchestrates the complete PR analysis
func AnalyzePR(ctx context.Context, input map[string]any) (any, error) {
    repo := input["repo"].(string)
    prNumber := int(input["pr_number"].(float64))

    // Fetch PR files (using GitHub client - deterministic)
    files, err := fetchPRFiles(repo, prNumber)
    if err != nil {
        return nil, err
    }

    // For each file, call parse_code_structure reasoner
    // (This could use CallLocal for same-agent calls)
    results := make([]*FileAnalysis, 0, len(files))
    for _, file := range files {
        // Use agent from context or pass agent instance
        result, err := analyzeFile(ctx, file)
        if err != nil {
            continue // Log and skip problematic files
        }
        results = append(results, result)
    }

    return map[string]any{
        "files": results,
        "total_files": len(files),
    }, nil
}

// ParseCodeStructure parses code into AST
func ParseCodeStructure(ctx context.Context, input map[string]any) (any, error) {
    content := input["content"].(string)
    language := input["language"].(string)

    ast, err := parseToAST(content, language)
    if err != nil {
        return nil, err
    }

    return map[string]any{
        "ast": ast,
        "language": language,
    }, nil
}

// CalculateComplexity calculates code metrics
func CalculateComplexity(ctx context.Context, input map[string]any) (any, error) {
    filePath := input["file_path"].(string)
    content := input["content"].(string)

    metrics := calculateMetrics(filePath, content)

    return map[string]any{
        "complexity": metrics.Complexity,
        "loc": metrics.LOC,
        "functions": metrics.FunctionCount,
    }, nil
}

// Helper functions (not reasoners - internal implementation)
func fetchPRFiles(repo string, prNumber int) ([]*FileChange, error) {
    // Use go-github to fetch PR files
    return nil, nil
}

func parseToAST(content, language string) (*CodeAST, error) {
    // Use go/parser for Go, tree-sitter for other languages
    return nil, nil
}

func analyzeFile(ctx context.Context, file *FileChange) (*FileAnalysis, error) {
    // Analyze individual file
    return nil, nil
}

func calculateMetrics(filePath, content string) *Metrics {
    return &Metrics{}
}
```

### 3.3 Standards Validator

**Location**: `features/standards/`

**Responsibilities:**

- Validate code against configurable standards
- Check documentation completeness
- Verify naming conventions
- Enforce architectural patterns

**Configuration Schema:**

```yaml
standards:
  coding:
    max_line_length: 100
    max_function_length: 50
    max_complexity: 10
    naming_conventions:
      functions: snake_case
      classes: PascalCase
      constants: UPPER_SNAKE_CASE
  documentation:
    require_docstrings: true
    docstring_style: google # google, numpy, sphinx
    require_type_hints: true
  architecture:
    prohibited_patterns:
      - circular_imports
      - god_classes
    required_patterns:
      - single_responsibility
```

**Implementation:**

```go
// features/standards/standards.go
package standards

func RegisterReasoners(app *agentfield.Agent) {
    app.Reasoner("validate_standards", ValidateStandards)
}

// ValidateStandards checks code against configured standards
func ValidateStandards(ctx context.Context, files []*FileChange, config *StandardsConfig) (*ValidationReport, error) {
    // Implement validation logic
    return nil, nil
}
```

### 3.4 Code Reviewer

**Location**: `features/reviewer/`

**Responsibilities:**

- Perform AI-powered code review using DeepSeek
- Identify bugs, security vulnerabilities, and code smells
- Generate actionable feedback
- Prioritize issues by severity

**Implementation:**

```go
// features/reviewer/reviewer.go
package reviewer

import (
    "context"
    "fmt"
    "github.com/Agent-Field/agentfield/sdk/go/agent"
    "github.com/Agent-Field/agentfield/sdk/go/ai"
)

// RegisterReasoners registers all review reasoners
func RegisterReasoners(app *agent.Agent) {
    app.RegisterReasoner("review_code", ReviewCode,
        agent.WithDescription("Performs AI-powered code review"))

    app.RegisterReasoner("detect_security_issues", DetectSecurityIssues,
        agent.WithDescription("Detects security vulnerabilities in code"))
}

// ReviewCode performs AI-powered comprehensive code review
func ReviewCode(ctx context.Context, input map[string]any) (any, error) {
    // Get agent from context (passed via middleware)
    agentInstance := GetAgentFromContext(ctx)

    files := input["files"].([]any)
    prContext := input["pr_context"].(map[string]any)

    // Build review prompt
    prompt := buildReviewPrompt(files, prContext)

    // Use AgentField's built-in AI method
    response, err := agentInstance.AI(ctx, prompt,
        ai.WithSystem("You are an expert code reviewer. Analyze code for bugs, security issues, performance problems, and maintainability concerns. Provide specific, actionable feedback."),
        ai.WithTemperature(0.2),
        ai.WithMaxTokens(4000))

    if err != nil {
        return nil, fmt.Errorf("AI review failed: %w", err)
    }

    // Parse AI response into structured review report
    report := parseReviewResponse(response.Text())

    return map[string]any{
        "report": report,
        "model": response.Model,
    }, nil
}

// DetectSecurityIssues performs specialized security vulnerability detection
func DetectSecurityIssues(ctx context.Context, input map[string]any) (any, error) {
    agentInstance := GetAgentFromContext(ctx)
    files := input["files"].([]any)

    // Build security-focused prompt
    prompt := buildSecurityPrompt(files)

    // Use AI for security analysis
    response, err := agentInstance.AI(ctx, prompt,
        ai.WithSystem("You are a security expert. Analyze code for OWASP Top 10 vulnerabilities, secret leaks, injection attacks, and other security issues."),
        ai.WithTemperature(0.1)) // Lower temp for deterministic security checks

    if err != nil {
        return nil, err
    }

    issues := parseSecurityIssues(response.Text())

    return map[string]any{
        "security_issues": issues,
        "count": len(issues),
    }, nil
}

// Helper functions
func buildReviewPrompt(files []any, prContext map[string]any) string {
    // Build comprehensive review prompt
    return ""
}

func buildSecurityPrompt(files []any) string {
    // Build security-focused prompt
    return ""
}

func parseReviewResponse(text string) *ReviewReport {
    // Parse AI response into structured data
    return &ReviewReport{}
}

func parseSecurityIssues(text string) []*SecurityIssue {
    // Parse security issues from AI response
    return []*SecurityIssue{}
}

// Context helper - store agent in context for access in reasoners
func GetAgentFromContext(ctx context.Context) *agent.Agent {
    // Implementation: retrieve agent from context
    return nil
}
```

**Review Categories:**

- **Critical**: Security vulnerabilities, breaking changes
- **High**: Bugs, performance issues, architectural violations
- **Medium**: Code smells, maintainability concerns
- **Low**: Style issues, minor improvements

### 3.5 Fix Generator with Validation Loop

**Location**: `features/fixer/`

**Responsibilities:**

- Generate code fixes for identified issues using DeepSeek
- Create minimal, targeted patches
- **Validate fixes don't introduce new problems**
- **Iteratively refine fixes until they pass validation**
- Apply fixes using git operations

**Implementation with Feedback Loop:**

```go
// features/fixer/fixer.go
package fixer

import (
    "context"
    "fmt"
    "github.com/Agent-Field/agentfield/sdk/go/agent"
    "github.com/Agent-Field/agentfield/sdk/go/ai"
)

const MaxFixAttempts = 3

func RegisterReasoners(app *agent.Agent) {
    app.RegisterReasoner("generate_fixes_with_validation", GenerateFixesWithValidation,
        agent.WithDescription("Generates and validates code fixes with retry loop"))

    app.RegisterReasoner("validate_fix", ValidateFix,
        agent.WithDescription("Validates a code fix against multiple criteria"))
}

// GenerateFixesWithValidation generates and validates fixes with retry loop
func GenerateFixesWithValidation(ctx context.Context, input map[string]any) (any, error) {
    agentInstance := GetAgentFromContext(ctx)
    issues := input["issues"].([]any)
    files := input["files"].([]any)

    var validatedPatches []*CodePatch

    for _, issueData := range issues {
        issue := issueData.(*Issue)
        var patch *CodePatch
        var validationErrors []string

        // Retry loop: attempt to fix up to MaxFixAttempts times
        for attempt := 1; attempt <= MaxFixAttempts; attempt++ {
            // Generate fix using AI
            patch, err := generateSingleFix(ctx, agentInstance, issue, files, validationErrors)
            if err != nil {
                return nil, fmt.Errorf("fix generation failed: %w", err)
            }

            // Validate the fix by calling validate_fix reasoner
            validationInput := map[string]any{
                "patch": patch,
                "issue": issue,
            }

            validationResult, err := agentInstance.CallLocal(ctx, "validate_fix", validationInput)
            if err != nil {
                return nil, fmt.Errorf("validation failed: %w", err)
            }

            result := validationResult.(map[string]any)
            isValid := result["is_valid"].(bool)

            // Check if fix is valid
            if isValid {
                validatedPatches = append(validatedPatches, patch)
                break // Success! Move to next issue
            }

            // Fix has problems, collect errors for next attempt
            validationErrors = result["errors"].([]string)

            if attempt == MaxFixAttempts {
                // Max attempts reached, log and skip this fix
                agentInstance.Notef(ctx, "Failed to generate valid fix for issue %s after %d attempts",
                    issue.ID, MaxFixAttempts)
                continue
            }

            agentInstance.Notef(ctx, "Fix attempt %d/%d failed validation, retrying with context...",
                attempt, MaxFixAttempts)
        }
    }

    return map[string]any{
        "validated_patches": validatedPatches,
        "count": len(validatedPatches),
    }, nil
}

// generateSingleFix creates a fix with context from previous validation errors
func generateSingleFix(ctx context.Context, agentInstance *agent.Agent, issue *Issue,
    files []any, previousErrors []string) (*CodePatch, error) {

    prompt := buildFixPrompt(issue, files)

    // If we have validation errors from previous attempt, include them
    if len(previousErrors) > 0 {
        prompt += fmt.Sprintf("\n\nPrevious fix attempt had these issues:\n%s\n"+
            "Please generate a fix that avoids these problems.",
            strings.Join(previousErrors, "\n"))
    }

    // Use AgentField's built-in AI method
    response, err := agentInstance.AI(ctx, prompt,
        ai.WithSystem("You are an expert code fixer. Generate minimal, targeted fixes that pass all validation checks (syntax, linting, formatting). Output ONLY the fixed code, no explanations."),
        ai.WithTemperature(0.1), // Lower temperature for deterministic fixes
        ai.WithMaxTokens(2000))

    if err != nil {
        return nil, err
    }

    return parseFixResponse(response.Text())
}

// ValidateFix ensures fix doesn't introduce new issues
func ValidateFix(ctx context.Context, input map[string]any) (any, error) {
    patch := input["patch"].(*CodePatch)
    issue := input["issue"].(*Issue)

    errors := []string{}

    // 1. Syntax validation
    if err := validateSyntax(patch); err != nil {
        errors = append(errors, fmt.Sprintf("Syntax error: %v", err))
    }

    // 2. Run linters on the fixed code
    if lintErrors := runLinters(patch); len(lintErrors) > 0 {
        errors = append(errors, lintErrors...)
    }

    // 3. Format validation
    if formatErrors := checkFormatting(patch); len(formatErrors) > 0 {
        errors = append(errors, formatErrors...)
    }

    // 4. Security checks on the fix
    if securityIssues := scanForSecurityIssues(patch); len(securityIssues) > 0 {
        errors = append(errors, securityIssues...)
    }

    // 5. Check if fix actually addresses the original issue
    if !doesFixAddressIssue(patch, issue) {
        errors = append(errors, "Fix does not address the original issue")
    }

    isValid := len(errors) == 0

    return map[string]any{
        "is_valid": isValid,
        "errors":   errors,
    }, nil
}

// Language-specific validation functions
func validateSyntax(patch *CodePatch) error {
    switch patch.Language {
    case "go":
        return validateGoSyntax(patch.Content)
    case "python":
        return validatePythonSyntax(patch.Content)
    case "javascript", "typescript":
        return validateJSSyntax(patch.Content)
    default:
        return nil // Skip syntax check for unknown languages
    }
}

func runLinters(patch *CodePatch) []string {
    var errors []string

    switch patch.Language {
    case "go":
        // Run golangci-lint
        if lintErrs := runGolangciLint(patch); len(lintErrs) > 0 {
            errors = append(errors, lintErrs...)
        }
    case "python":
        // Run pylint, flake8
        if lintErrs := runPylint(patch); len(lintErrs) > 0 {
            errors = append(errors, lintErrs...)
        }
    case "javascript", "typescript":
        // Run eslint
        if lintErrs := runESLint(patch); len(lintErrs) > 0 {
            errors = append(errors, lintErrs...)
        }
    }

    return errors
}

func checkFormatting(patch *CodePatch) []string {
    var errors []string

    switch patch.Language {
    case "go":
        // Check gofmt
        if !isGoFormatted(patch.Content) {
            errors = append(errors, "Code is not properly formatted (gofmt)")
        }
    case "python":
        // Check black formatting
        if !isPythonFormatted(patch.Content) {
            errors = append(errors, "Code is not properly formatted (black)")
        }
    case "javascript", "typescript":
        // Check prettier
        if !isJSFormatted(patch.Content) {
            errors = append(errors, "Code is not properly formatted (prettier)")
        }
    }

    return errors
}

type ValidationResult struct {
    IsValid bool
    Errors  []string
}
```

**Validation Pipeline:**

```
Fix Generated
     ↓
[1] Syntax Check → Parse code, ensure it's valid
     ↓
[2] Linting → Run language-specific linters
     ↓
[3] Formatting → Check code formatting standards
     ↓
[4] Security Scan → Check for new vulnerabilities
     ↓
[5] Issue Verification → Confirm fix addresses original problem
     ↓
Valid? → Yes → Apply Fix
       → No → Feed errors back to DeepSeek → Retry (max 3x)
```

### 3.6 Git Operations & Comment Linking

**Location**: `features/gitops/`

**Responsibilities:**

- Clone repositories and manage branches
- Apply patches and create commits
- Push changes (YOLO mode) or create PRs (Safe mode)
- Add review comments to GitHub with proper linking
- Update review comments with fix PR/commit links

**Comment Workflow:**

1. **Post initial review comments** with issue details
2. **Generate and validate fixes**
3. **Update comments with links** to fixes (PR or commit)

**Implementation:**

```go
// features/gitops/gitops.go
package gitops

import (
    "context"
    "fmt"
    "github.com/Agent-Field/agentfield/sdk/go/agent"
    "github.com/google/go-github/v57/github"
    "github.com/go-git/go-git/v5"
)

func RegisterReasoners(app *agent.Agent) {
    // Git operations reasoners
    app.RegisterReasoner("create_branch", CreateBranch,
        agent.WithDescription("Creates a new Git branch"))

    app.RegisterReasoner("apply_patches", ApplyPatches,
        agent.WithDescription("Applies code patches and creates commits"))

    app.RegisterReasoner("create_pull_request", CreatePullRequest,
        agent.WithDescription("Creates a GitHub pull request"))

    // Review comment reasoners
    app.RegisterReasoner("add_review_comment", AddReviewComment,
        agent.WithDescription("Adds a code review comment to a PR"))

    app.RegisterReasoner("update_review_comment", UpdateReviewComment,
        agent.WithDescription("Updates an existing review comment"))

    // Main orchestration reasoner
    app.RegisterReasoner("post_review_with_fixes", PostReviewWithFixes,
        agent.WithDescription("Orchestrates posting review comments and applying fixes"))
}

// CreateBranch creates a new branch for fixes
func CreateBranch(ctx context.Context, input map[string]any) (any, error) {
    repo := input["repo"].(string)
    baseBranch := input["base_branch"].(string)
    newBranch := input["new_branch"].(string)

    // Use go-git or GitHub API
    branch, err := createGitBranch(repo, baseBranch, newBranch)
    if err != nil {
        return nil, err
    }

    return map[string]any{
        "branch_name": branch.Name,
        "sha": branch.SHA,
    }, nil
}

// ApplyPatches applies code patches and commits
func ApplyPatches(ctx context.Context, repoPath string, patches []*CodePatch) (*CommitResult, error) {
    // Apply patches to local repo and commit
    return nil, nil
}

// CreatePullRequest creates PR with fixes
func CreatePullRequest(ctx context.Context, repo, headBranch, baseBranch, title, body string) (*github.PullRequest, error) {
    // Use go-github to create PR
    return nil, nil
}

// ReviewComment represents a code review comment with optional fix link
type ReviewComment struct {
    FilePath  string
    Line      int
    Body      string
    IssueID   string
    Severity  string
    FixCommit string // Commit SHA (YOLO mode)
    FixPR     int    // PR number (Safe mode)
}

// AddReviewComment posts a review comment to GitHub PR
func AddReviewComment(ctx context.Context, repo string, prNumber int, comment *ReviewComment) (int64, error) {
    // Build comment body with issue details
    body := formatCommentBody(comment)

    // Create review comment using GitHub API
    ghComment, _, err := ghClient.PullRequests.CreateComment(ctx, owner, repo, prNumber, &github.PullRequestComment{
        Path:     github.String(comment.FilePath),
        Line:     github.Int(comment.Line),
        Body:     github.String(body),
    })

    if err != nil {
        return 0, err
    }

    return ghComment.GetID(), nil
}

// UpdateReviewComment updates an existing comment with fix link
func UpdateReviewComment(ctx context.Context, repo string, commentID int64, fixLink string) error {
    // Get existing comment
    comment, _, err := ghClient.PullRequests.GetComment(ctx, owner, repo, commentID)
    if err != nil {
        return err
    }

    // Append fix link to comment body
    updatedBody := fmt.Sprintf("%s\n\n✅ **Fix available:** %s", comment.GetBody(), fixLink)

    // Update comment
    _, _, err = ghClient.PullRequests.EditComment(ctx, owner, repo, commentID, &github.PullRequestComment{
        Body: github.String(updatedBody),
    })

    return err
}

// PostReviewWithFixes orchestrates the complete review + fix workflow
func PostReviewWithFixes(ctx context.Context, repo string, prNumber int,
    issues []*Issue, mode string) error {

    // Step 1: Post initial review comments for all issues
    commentIDs := make(map[string]int64) // issueID -> commentID

    for _, issue := range issues {
        comment := &ReviewComment{
            FilePath: issue.FilePath,
            Line:     issue.Line,
            Body:     formatIssueComment(issue),
            IssueID:  issue.ID,
            Severity: issue.Severity,
        }

        commentID, err := AddReviewComment(ctx, repo, prNumber, comment)
        if err != nil {
            return fmt.Errorf("failed to post comment: %w", err)
        }

        commentIDs[issue.ID] = commentID
    }

    // Step 2: Generate and validate fixes
    fixes, err := GenerateFixesWithValidation(ctx, issues, files)
    if err != nil {
        return fmt.Errorf("failed to generate fixes: %w", err)
    }

    // Step 3: Apply fixes based on mode
    if mode == "yolo" {
        // YOLO mode: Push directly to PR branch
        commitSHA, err := applyFixesDirectly(ctx, repo, prNumber, fixes)
        if err != nil {
            return fmt.Errorf("failed to apply fixes: %w", err)
        }

        // Step 4: Update comments with commit links
        for _, fix := range fixes {
            commentID := commentIDs[fix.IssueID]
            commitURL := fmt.Sprintf("https://github.com/%s/commit/%s", repo, commitSHA)
            fixLink := fmt.Sprintf("[View fix commit](%s)", commitURL)

            if err := UpdateReviewComment(ctx, repo, commentID, fixLink); err != nil {
                log.Warnf("Failed to update comment %d: %v", commentID, err)
            }
        }

    } else {
        // Safe mode: Create fix PR
        fixPR, err := createFixPR(ctx, repo, prNumber, fixes)
        if err != nil {
            return fmt.Errorf("failed to create fix PR: %w", err)
        }

        // Step 4: Update comments with PR links
        for _, fix := range fixes {
            commentID := commentIDs[fix.IssueID]
            prURL := fmt.Sprintf("https://github.com/%s/pull/%d", repo, fixPR.GetNumber())
            fixLink := fmt.Sprintf("[View fix PR #%d](%s)", fixPR.GetNumber(), prURL)

            if err := UpdateReviewComment(ctx, repo, commentID, fixLink); err != nil {
                log.Warnf("Failed to update comment %d: %v", commentID, err)
            }
        }

        // Post summary comment on original PR with link to fix PR
        summaryComment := fmt.Sprintf(
            "🤖 **Automated Code Review Complete**\n\n"+
            "Found %d issues. Fixes are available in PR #%d\n\n"+
            "👉 [Review fixes](%s)",
            len(issues), fixPR.GetNumber(), prURL,
        )

        _, _, err = ghClient.Issues.CreateComment(ctx, owner, repo, prNumber, &github.IssueComment{
            Body: github.String(summaryComment),
        })
    }

    return nil
}

// formatIssueComment creates a well-formatted comment body
func formatIssueComment(issue *Issue) string {
    emoji := getSeverityEmoji(issue.Severity)

    return fmt.Sprintf(
        "%s **%s**: %s\n\n"+
        "**Details:**\n"+
        "%s\n\n"+
        "_Automated review by GitHub Code Agent_",
        emoji, issue.Severity, issue.Title,
        issue.Description,
    )
}

func getSeverityEmoji(severity string) string {
    switch severity {
    case "Critical":
        return "🔴"
    case "High":
        return "🟠"
    case "Medium":
        return "🟡"
    case "Low":
        return "🔵"
    default:
        return "⚪"
    }
}
```

**Comment Format Examples:**

**Initial Review Comment:**

```markdown
🔴 **Critical**: Missing null pointer check

**Details:**
The function `GetUser` doesn't check if the user exists before accessing
properties, which could cause a nil pointer dereference.

_Automated review by GitHub Code Agent_
```

**Updated Comment (YOLO mode):**

```markdown
🔴 **Critical**: Missing null pointer check

**Details:**
The function `GetUser` doesn't check if the user exists before accessing
properties, which could cause a nil pointer dereference.

_Automated review by GitHub Code Agent_

✅ **Fix available:** [View fix commit](https://github.com/org/repo/commit/abc123)
```

**Updated Comment (Safe mode):**

```markdown
🟡 **Medium**: Line exceeds maximum length

**Details:**
Line 102 has 145 characters, exceeding the configured limit of 100.

_Automated review by GitHub Code Agent_

✅ **Fix available:** [View fix PR #124](https://github.com/org/repo/pull/124)
```

## 4. Feedback Loop System

### 4.1 Fix Validation Loop

The agent implements a **validation-retry feedback loop** to ensure generated fixes don't introduce new problems. This prevents common issues like:

- Fixes that introduce syntax errors
- Fixes that violate linting rules
- Fixes that break code formatting
- Fixes that create new security vulnerabilities
- Fixes that don't actually solve the original issue

**Loop Configuration:**

```go
const (
    MaxFixAttempts = 3  // Maximum retry attempts per fix
    ValidationTimeout = 30 * time.Second
)
```

**Loop Flow:**

```
Attempt 1: Generate Fix
    ↓
Validate Fix (syntax, lint, format, security)
    ↓
Valid? → YES → Success! Apply fix
    ↓ NO
Attempt 2: Regenerate with validation errors as context
    ↓
Validate Fix
    ↓
Valid? → YES → Success! Apply fix
    ↓ NO
Attempt 3: Final attempt with all previous errors
    ↓
Validate Fix
    ↓
Valid? → YES → Success! Apply fix
       → NO → Skip this fix, log failure, notify in PR comment
```

**Why limit to 3 attempts?**

- **Prevent infinite loops**: Some issues may be unfixable automatically
- **Cost control**: Each attempt costs AI API tokens
- **Time bounds**: Keep review cycle under 2 minutes
- **Graceful degradation**: Better to report "couldn't fix" than hang forever

**Validation Checks:**

Each fix goes through these validations:

1. **Syntax Validation**: Parse code AST to ensure valid syntax
2. **Linting**: Run language-specific linters (golangci-lint, pylint, eslint)
3. **Formatting**: Verify code follows formatting standards (gofmt, black, prettier)
4. **Security Scanning**: Check for new vulnerabilities
5. **Issue Verification**: Confirm fix addresses the original problem

### 4.2 CI/CD Integration Loop

The agent can optionally wait for CI/CD pipelines to complete before reviewing:

**Pipeline-Aware Workflow:**

```
PR Created/Updated
    ↓
Agent Receives Webhook
    ↓
Check CI/CD Status
    ↓
Running? → Wait for check_suite.completed event
    ↓
Completed?
    ↓
Success → Review code + suggest improvements
Failed  → Review code + analyze test failures + suggest fixes
```

**Benefits:**

- Don't waste time reviewing code that fails tests
- Can provide context-aware fixes based on test failures
- Reduce noise (wait for force-push series to settle)
- Better signal-to-noise ratio in review comments

## 5. Operational Modes

### 4.1 YOLO Mode

**Behavior:**

- Posts review comments first (with issue details)
- Generates and validates fixes
- Directly pushes fixes to the source PR branch
- Updates review comments with commit links
- Suitable for trusted repos with good test coverage

**Configuration:**

```yaml
mode: yolo
auto_fix:
  enabled: true
  severity_threshold: medium # auto-fix medium and below
  require_tests_passing: true
```

**Workflow:**

```
PR Opened
    ↓
Review Code → Find Issues
    ↓
Post Review Comments (with issue details) ← Comment IDs saved
    ↓
Generate & Validate Fixes (feedback loop)
    ↓
Push Fixes to PR Branch → Get Commit SHA
    ↓
Update Review Comments with commit links
```

**Comment Linking Example:**

Initial comment:

```markdown
🔴 **Critical**: SQL injection vulnerability
_Automated review by GitHub Code Agent_
```

After fix is pushed:

```markdown
🔴 **Critical**: SQL injection vulnerability
_Automated review by GitHub Code Agent_

✅ **Fix available:** [View fix commit](https://github.com/org/repo/commit/def456)
```

### 4.2 Safe Mode

**Behavior:**

- Posts review comments first (with issue details)
- Generates and validates fixes
- Creates a new branch with fixes (e.g., `agent-fixes/pr-123`)
- Opens a new PR targeting the original PR branch
- Updates review comments with fix PR links
- Posts summary comment on original PR linking to fix PR
- Allows human review before merging fixes

**Configuration:**

```yaml
mode: safe
auto_fix:
  enabled: true
  create_pr: true
  pr_title_template: "🤖 Automated fixes for PR #{pr_number}"
  pr_description_template: |
    ## Automated Code Review Fixes

    This PR contains automated fixes for issues found in PR #{pr_number}.

    ### Issues Fixed:
    {issue_list}

    ### Validation Results:
    All fixes have been validated for:
    - ✅ Syntax correctness
    - ✅ Linting compliance
    - ✅ Code formatting
    - ✅ Security checks

    Please review and merge if changes are acceptable.
```

**Workflow:**

```
PR Opened
    ↓
Review Code → Find Issues
    ↓
Post Review Comments (with issue details) ← Comment IDs saved
    ↓
Generate & Validate Fixes (feedback loop)
    ↓
Create Fix Branch → Apply Patches → Push
    ↓
Create Fix PR → Get PR Number & URL
    ↓
Update Review Comments with PR links
    ↓
Post Summary Comment on Original PR with link to Fix PR
```

**Comment Linking Example:**

Initial comment on original PR #123:

```markdown
🟡 **Medium**: Unused variable detected
_Automated review by GitHub Code Agent_
```

After fix PR #124 is created:

```markdown
🟡 **Medium**: Unused variable detected
_Automated review by GitHub Code Agent_

✅ **Fix available:** [View fix PR #124](https://github.com/org/repo/pull/124)
```

Summary comment on original PR #123:

```markdown
🤖 **Automated Code Review Complete**

Found 5 issues. Fixes are available in PR #124

👉 [Review fixes](https://github.com/org/repo/pull/124)

**Issues addressed:**

- 🔴 1 Critical
- 🟠 1 High
- 🟡 3 Medium
```

## 5. Configuration System

### 5.1 Repository-Level Configuration

Store configuration in `.github/code-agent.yml`:

```yaml
# Agent configuration
agent:
  enabled: true
  mode: safe # yolo or safe

# Webhook triggers
webhooks:
  triggers:
    - pull_request.opened
    - pull_request.synchronize
    - check_suite.completed # Wait for CI/CD to finish
    - workflow_run.completed
  wait_for_ci: true # Wait for CI/CD before reviewing
  debounce_seconds: 30 # Wait 30s for rapid commits to settle

# Review settings
review:
  auto_review: true
  auto_fix: true
  severity_threshold: medium # Only auto-fix medium and below
  ignore_paths:
    - "*.md"
    - "docs/**"
    - "tests/fixtures/**"

# Feedback loop settings
validation:
  enabled: true
  max_fix_attempts: 3 # Maximum retry attempts per fix
  checks:
    - syntax # Validate syntax
    - linting # Run linters
    - formatting # Check code formatting
    - security # Security vulnerability scan
  timeout_seconds: 30 # Timeout per validation attempt
  auto_format: true # Automatically format code before validation

# Coding standards
standards:
  coding:
    max_line_length: 100
    max_function_length: 50
    max_complexity: 10
    naming_conventions:
      functions: snake_case
      classes: PascalCase

  documentation:
    require_docstrings: true
    docstring_style: google
    require_type_hints: true
    require_module_docs: true

  security:
    check_dependencies: true
    check_secrets: true
    owasp_checks: true

# Language-specific settings
languages:
  python:
    linters:
      - pylint
      - black
      - mypy
    min_test_coverage: 80

  javascript:
    linters:
      - eslint
      - prettier
    frameworks:
      - react

# AI model configuration
ai:
  provider: deepseek
  model: deepseek-chat
  base_url: https://api.deepseek.com
  temperature: 0.2
  max_tokens: 4000

# Notification settings
notifications:
  on_review_complete: true
  on_fixes_applied: true
  mention_author: true
```

### 5.2 Environment Variables

Create a `.env` file for local development:

```bash
# AgentField Configuration
AGENTFIELD_URL=http://localhost:8080

# GitHub Configuration
GITHUB_APP_ID=123456
GITHUB_PRIVATE_KEY_PATH=./github-app.pem
GITHUB_WEBHOOK_SECRET=your-webhook-secret

# AI Configuration (Option 1: DeepSeek via OpenRouter - Recommended)
OPENROUTER_API_KEY=sk-or-v1-your-openrouter-api-key
AI_MODEL=deepseek/deepseek-chat

# AI Configuration (Option 2: OpenAI)
# OPENOPENAI_API_KEY=sk-proj-your-openai-api-key
# AI_MODEL=gpt-4o

# Application Settings
LOG_LEVEL=info
```

### 5.3 User Preferences

Allow users to customize behavior via PR labels:

- `agent:skip` - Skip automated review
- `agent:yolo` - Override to YOLO mode
- `agent:safe` - Override to Safe mode
- `agent:review-only` - Review but don't auto-fix

## 6. Implementation Phases

### Phase 1: Foundation (Week 1-2)

**Goal**: Basic webhook handling and PR analysis

- [ ] Initialize Go project with feature-based structure
  - Set up `go.mod` with dependencies
  - Create feature directories
  - Set up AgentField Go SDK
- [x] Set up AI integration via AgentField SDK
  - AgentField SDK handles AI client (OpenAI-compatible APIs)
  - Configure via environment variables (API_KEY, BASE_URL, MODEL)
  - No custom wrapper needed
- [ ] Implement Webhook Handler (`features/webhook/`)
  - Webhook signature validation
  - Event parsing (pull_request, check_suite, workflow_run)
  - **CI/CD status checking**
  - **Event debouncing for rapid commits**
  - Routing logic
- [ ] Implement Code Analyzer (`features/analyzer/`)
  - GitHub API integration (`google/go-github`)
  - File diff parsing
  - Basic code metrics
  - **CI/CD test result parsing**
- [ ] Create configuration system (`pkg/config/`)
  - YAML parser for `.github/code-agent.yml`
  - Environment variable loading
  - Config validation
  - **Feedback loop configuration**
- [ ] Set up GitHub App registration
  - Configure webhook events (pull_request, check_suite, workflow_run)
- [ ] Create main entry point (`cmd/agent/main.go`)

**Deliverables:**

- Working Go project structure
- Webhook receiver with CI/CD awareness
- PR content extraction
- Configuration system with feedback loop settings
- DeepSeek integration

### Phase 2: Review Engine (Week 3-4)

**Goal**: Intelligent code review capabilities

- [ ] Implement Code Reviewer (`features/reviewer/`)
  - DeepSeek-powered code analysis
  - Prompt engineering for code review
  - Security vulnerability detection (OWASP checks)
  - Code smell identification
  - Response parsing and structuring
- [ ] Implement Standards Validator (`features/standards/`)
  - Configurable rule engine
  - Multiple language support (Go, Python, JS, etc.)
  - Custom rule definitions
  - Naming convention checks
- [ ] Build issue prioritization system
  - Severity classification (Critical, High, Medium, Low)
  - Issue deduplication
- [ ] Implement review comment posting
  - GitHub review API integration
  - Comment formatting
  - Inline code suggestions

**Deliverables:**

- DeepSeek-powered code review
- GitHub review comments with suggestions
- Issue severity classification
- Standards validation

### Phase 3: Fix Generation (Week 5-6)

**Goal**: Automated fix application with validation loop

- [ ] Implement Fix Generator with Validation Loop (`features/fixer/`)
  - DeepSeek-powered fix generation
  - Prompt engineering for code fixes
  - Patch creation (diff generation)
  - **Validation-retry feedback loop (max 3 attempts)**
  - **Syntax validation per language**
  - **Linting integration** (golangci-lint, pylint, eslint)
  - **Formatting checks** (gofmt, black, prettier)
  - **Security scanning on fixes**
  - **Auto-formatting before validation**
  - Multi-issue batch fixing
- [ ] Implement Validation Engine (`features/fixer/validator.go`)
  - Language-specific syntax parsers (Go AST, Python AST, JS parser)
  - Linter runners (execute external linters, parse output)
  - Formatter checkers
  - Security scanners
  - Issue verification (confirm fix addresses original problem)
- [ ] Implement Git Operations (`features/gitops/`)
  - Repository cloning (using `go-git`)
  - Branch management
  - Commit creation with proper messages
  - Push operations
  - PR creation via GitHub API
- [ ] Implement Safe mode workflow
  - Create fix branch (e.g., `agent-fixes/pr-123`)
  - Apply validated patches to fix branch
  - Create PR targeting original branch
  - Add explanatory comments with validation results
- [ ] Implement YOLO mode workflow
  - Apply validated patches directly to source branch
  - Push to remote
  - Add review comments with validation status
- [ ] Add safety checks
  - **CI/CD test pass validation**
  - Conflict detection
  - Rollback capabilities
  - **Max attempt limits to prevent infinite loops**

**Deliverables:**

- DeepSeek-powered fix generation with validation loop
- Comprehensive validation engine
- Both operational modes (YOLO & Safe)
- Git operations integration
- Safety mechanisms with max 3 retry attempts

### Phase 4: Polish & Testing (Week 7-8)

**Goal**: Production-ready system

- [ ] Comprehensive testing
  - Unit tests for each agent
  - Integration tests for workflows
  - End-to-end PR testing
- [ ] Performance optimization
  - Parallel agent execution
  - Caching strategies
  - Rate limit handling
- [ ] Documentation
  - User guide
  - Configuration reference
  - API documentation
- [ ] Monitoring & observability
  - AgentField DAG visualization
  - Metrics dashboard
  - Error tracking

**Deliverables:**

- Production-ready agent system
- Complete documentation
- Monitoring setup

### Phase 5: Advanced Features (Week 9-10)

**Goal**: Enhanced capabilities

- [ ] Multi-repository support
- [ ] Custom rule marketplace
- [ ] Learning from feedback
  - Track fix acceptance rate
  - Improve suggestions over time
- [ ] Advanced security features
  - Dependency vulnerability scanning
  - Secret detection
  - License compliance
- [ ] Integration with CI/CD
  - Wait for test results
  - Block merge on critical issues

**Deliverables:**

- Advanced feature set
- Feedback learning system
- CI/CD integration

## 7. Complete Workflow Examples

### 7.1 End-to-End Workflow with Feedback Loop

**Scenario: Developer pushes commits to PR**

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Developer pushes 3 commits to PR #123                    │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. GitHub sends pull_request.synchronize webhook            │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. Agent receives webhook                                   │
│    - Debounces for 30s (wait for more commits)              │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ 4. Check CI/CD status                                       │
│    - GitHub Actions workflow is running                     │
│    - Agent subscribes to check_suite.completed event        │
│    - Wait...                                                │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ 5. CI/CD completes (check_suite.completed event)            │
│    - Status: FAILURE (tests failed in auth.go)              │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ 6. Start Review Workflow                                    │
│    ├─> Fetch PR files (3 changed files)                     │
│    ├─> Analyze code structure                               │
│    ├─> Validate against standards                           │
│    └─> Review code + analyze test failures                  │
│         Issues found:                                        │
│         - Missing null check in auth.go:45 (High)            │
│         - Unused import in auth.go:5 (Low)                   │
│         - Line too long in user.go:102 (Low)                 │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ 7. Post Review Comments FIRST                               │
│    ├─> Comment 1 (ID: 98765): Missing null check            │
│    ├─> Comment 2 (ID: 98766): Unused import                 │
│    └─> Comment 3 (ID: 98767): Line too long                 │
│    Save comment IDs for later linking                       │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ 8. Generate Fixes (Feedback Loop Start)                     │
│                                                              │
│    Issue 1: Missing null check                              │
│    Attempt 1:                                                │
│    ├─> DeepSeek generates fix                               │
│    ├─> Validation:                                           │
│    │   ✓ Syntax: OK                                          │
│    │   ✓ Lint: OK                                            │
│    │   ✗ Format: Line too long (91 chars)                    │
│    │   ✓ Security: OK                                        │
│    └─> INVALID → Retry with error context                   │
│                                                              │
│    Attempt 2:                                                │
│    ├─> DeepSeek generates fix (with formatting constraint)  │
│    ├─> Validation:                                           │
│    │   ✓ Syntax: OK                                          │
│    │   ✓ Lint: OK                                            │
│    │   ✓ Format: OK                                          │
│    │   ✓ Security: OK                                        │
│    │   ✓ Addresses issue: OK                                 │
│    └─> VALID ✓ → Keep this fix                              │
│                                                              │
│    Issue 2: Unused import                                   │
│    Attempt 1:                                                │
│    ├─> DeepSeek generates fix                               │
│    ├─> Validation: All checks pass ✓                         │
│    └─> VALID → Keep this fix                                │
│                                                              │
│    Issue 3: Line too long                                   │
│    Attempt 1:                                                │
│    ├─> DeepSeek generates fix                               │
│    ├─> Validation: All checks pass ✓                         │
│    └─> VALID → Keep this fix                                │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ 9. Apply Fixes (Safe Mode)                                  │
│    ├─> Create branch: agent-fixes/pr-123                    │
│    ├─> Apply all validated patches                          │
│    ├─> Commit: "🤖 Auto-fix: 3 issues resolved"             │
│    ├─> Push to remote                                       │
│    └─> Create PR #124 → targets PR #123 branch              │
│         Get PR URL: https://github.com/org/repo/pull/124    │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ 10. Update Review Comments with Fix PR Links                │
│     ├─> Update Comment 98765:                               │
│     │   Add: "✅ Fix available: [View fix PR #124](...)"    │
│     ├─> Update Comment 98766:                               │
│     │   Add: "✅ Fix available: [View fix PR #124](...)"    │
│     └─> Update Comment 98767:                               │
│         Add: "✅ Fix available: [View fix PR #124](...)"    │
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ 11. Post Summary Comment on Original PR #123                │
│     "🤖 Automated Code Review Complete                      │
│      Found 3 issues. Fixes available in PR #124             │
│      👉 [Review fixes](https://github.com/org/repo/pull/124)│
└───────────────────────┬─────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────────┐
│ 12. Developer reviews fix PR #124                           │
│     - Sees explanation of fixes                             │
│     - Reviews validation results                            │
│     - Approves and merges into PR #123                      │
│     - CI/CD runs again → PASS ✓                             │
└─────────────────────────────────────────────────────────────┘
```

### 7.2 Workflow with Max Retry Exhaustion

**Scenario: Fix cannot be validated after 3 attempts**

```
Issue: Complex refactoring needed in service.go:234

Attempt 1:
├─> DeepSeek generates fix
├─> Validation: Syntax error (missing closing brace)
└─> INVALID → Retry

Attempt 2:
├─> DeepSeek generates fix (with previous error context)
├─> Validation: Lint error (unused variable)
└─> INVALID → Retry

Attempt 3 (FINAL):
├─> DeepSeek generates fix (with all previous errors)
├─> Validation: Still has lint error
└─> INVALID → Max attempts reached

Result:
├─> Skip this fix
├─> Log warning: "Failed to auto-fix issue XXX after 3 attempts"
└─> Add PR comment:
    "⚠️ Could not automatically fix issue in service.go:234
    Manual review required. Attempted fixes had validation errors."
```

### 7.3 YOLO Mode Workflow

**Scenario: Trusted repo with good test coverage, immediate fixes**

```
PR Created → Review → Generate & Validate Fixes → Push directly to PR branch
                                                 ↓
                                    CI/CD runs automatically
                                                 ↓
                                    Pass ✓ → Ready to merge
                                    Fail ✗ → New review cycle
```

## 8. Technical Considerations

### 7.1 Go-Specific Considerations

**Dependencies:**

```go
// go.mod
module github.com/yourorg/github-code-agent

go 1.21

require (
    github.com/Agent-Field/agentfield/sdk/go v0.x.x
    github.com/google/go-github/v57 v57.0.0
    github.com/go-git/go-git/v5 v5.x.x
    gopkg.in/yaml.v3 v3.x.x
    github.com/joho/godotenv v1.x.x
)

// Note: DeepSeek is accessed via OpenRouter or OpenAI-compatible API
// using AgentField's built-in AI client - no separate client needed
```

**Error Handling:**

- Use Go's idiomatic error handling
- Wrap errors with context using `fmt.Errorf`
- Log errors with structured logging (e.g., `log/slog`)

**Concurrency:**

- Use goroutines for parallel file analysis
- Leverage channels for agent communication
- Implement proper context cancellation
- Use `sync.WaitGroup` for coordinating parallel operations

**Testing:**

- Table-driven tests for each feature
- Mock DeepSeek client for testing
- Mock GitHub API for integration tests
- Use `testify` for assertions

### 7.2 GitHub API Rate Limits

- **REST API**: 5,000 requests/hour for authenticated apps
- **GraphQL API**: 5,000 points/hour (more efficient for complex queries)
- **Strategy**:
  - Use conditional requests with ETags
  - Implement request caching (15-min TTL)
  - Use GraphQL for fetching PR data
  - Monitor rate limit headers

### 7.3 Security

- **Webhook Security**: Verify all webhook signatures using HMAC
- **Token Management**: Use GitHub App installation tokens (1-hour expiry)
- **Permissions**: Request minimal necessary permissions
  - `pull_requests`: read/write
  - `contents`: write
  - `metadata`: read
- **Secret Storage**: Use environment variables, never commit tokens
- **Code Execution**: Sandbox all code analysis (no eval/exec)

### 7.4 Performance

- **Parallel Processing**: Leverage AgentField's async execution
  - Analyze files in parallel
  - Run multiple reviewers concurrently
- **Timeout Handling**: Set reasonable timeouts (5 min for review)
- **Large PRs**:
  - Limit to 400 LOC per PR (best practice)
  - Batch processing for large changesets
  - Incremental review for synchronize events

### 7.5 Reliability

- **Webhook Retries**: GitHub retries failed webhooks
- **Idempotency**: Handle duplicate webhook events
- **Error Recovery**:
  - AgentField automatic retry on failure
  - Graceful degradation (comment even if fixes fail)
- **State Management**: Use AgentField shared memory for workflow state

### 7.6 AI Integration via AgentField Best Practices

**Accessing DeepSeek via OpenRouter:**

AgentField's built-in AI client supports DeepSeek through OpenRouter, providing a unified interface:

```bash
# Environment setup
export OPENROUTER_API_KEY="sk-or-v1-..."
export AI_MODEL="deepseek/deepseek-chat"
```

**Prompt Engineering:**

- Use clear, structured prompts for code review
- Include language and framework context
- Use `ai.WithSystem()` for role instructions
- Set appropriate temperature (0.1-0.2 for fixes, 0.2-0.3 for reviews)
- Use structured outputs with `ai.WithSchema()` for JSON parsing

**Cost Optimization:**

- DeepSeek Chat is cost-effective (~$0.14 per 1M input tokens, $0.28 per 1M output tokens)
- AgentField automatically caches similar requests (15-min TTL)
- Batch similar review tasks
- Limit context to relevant code sections
- Use lower temperatures for deterministic tasks

**Example Integration:**

```go
import (
    "github.com/Agent-Field/agentfield/sdk/go/agent"
    "github.com/Agent-Field/agentfield/sdk/go/ai"
)

// Inside a reasoner function
response, err := agentInstance.AI(ctx,
    fmt.Sprintf("Review this %s code:\n\n%s", language, code),
    ai.WithSystem("You are an expert code reviewer focusing on security, performance, and best practices."),
    ai.WithTemperature(0.2),
    ai.WithMaxTokens(2000))

if err != nil {
    return nil, err
}

reviewText := response.Text()
modelUsed := response.Model // "deepseek/deepseek-chat"
```

**Structured Outputs:**

```go
type CodeReview struct {
    Issues []struct {
        Severity    string `json:"severity"`
        Line        int    `json:"line"`
        Description string `json:"description"`
        Suggestion  string `json:"suggestion"`
    } `json:"issues"`
}

response, err := agentInstance.AI(ctx, prompt,
    ai.WithSystem("Respond with JSON only"),
    ai.WithSchema(CodeReview{}),
    ai.WithTemperature(0.1))

// Response will be structured as CodeReview
```

**Streaming for Long Responses:**

```go
chunks, errs := agentInstance.AIStream(ctx, prompt,
    ai.WithSystem("Generate detailed code review"),
    ai.WithTemperature(0.2))

for chunk := range chunks {
    if len(chunk.Choices) > 0 {
        content := chunk.Choices[0].Delta.Content
        // Process streaming content
        agentInstance.Notef(ctx, "Review progress: %s", content)
    }
}

if err := <-errs; err != nil {
    return nil, err
}
```

### 7.7 AgentField Integration Patterns

**Agent Communication:**

```go
// Orchestrator invokes Code Analyzer (same agent, local call)
analyzerResult, err := app.CallLocal(ctx, "analyze_pr", map[string]any{
    "repo": repo,
    "pr_number": prNumber,
})

// Or invoke remote agent (different node)
analyzerResult, err := app.Call(ctx, "github-code-agent.analyze_pr", map[string]any{
    "repo": repo,
    "pr_number": prNumber,
})

// Use hierarchical memory for state coordination
// Session scope - shared across this PR review session
err = app.Memory().Set(ctx, fmt.Sprintf("pr-%d-status", prNumber), "reviewing")

// Workflow scope - isolated to this execution
err = app.Memory().WorkflowScope().Set(ctx, "current_step", "analysis")

// Global scope - shared across all sessions
err = app.Memory().GlobalScope().Set(ctx, "total_prs_reviewed", count)
```

**Workflow DAG with Feedback Loop:**

```
webhook.handle_event
  │
  ├─> [Get CI/CD Status] → Wait if running
  │
  ├─> analyzer.analyze_pr (parallel)
  ├─> standards.validate_standards (parallel)
  │
  └─> reviewer.review_code
       │
       ├─> [Issues Found?] → Post review comments
       │
       └─> fixer.generate_fixes
            │
            └─> gitops.apply_fixes
                 │
                 └─> validator.validate_fixes ←─┐
                      │                          │
                      ├─> [Lint/Format Check]    │
                      ├─> [Syntax Validation]    │
                      ├─> [New Issues?] ─────────┘ (Loop back to fixer)
                      │                    Max 3 iterations
                      └─> [Success] → Commit & Push/PR
```

**Feedback Loop Details:**

The system implements a **validation-retry loop** to ensure fixes don't introduce new problems:

1. **Fix Generation** → Generate code fixes
2. **Validation** → Run linters, formatters, syntax checks
3. **Issue Detection** → Check if fixes introduce new problems
4. **Retry** → If new issues found, regenerate fixes (max 3 attempts)
5. **Success** → Apply fixes and create PR/push

This prevents scenarios like:

- Fix introduces syntax errors
- Fix fails linting rules
- Fix causes formatting violations
- Fix creates new security issues

**Main Entry Point:**

```go
// cmd/agent/main.go
package main

import (
    "context"
    "log"
    "os"

    "github.com/Agent-Field/agentfield/sdk/go/agent"
    "github.com/Agent-Field/agentfield/sdk/go/ai"
    "github.com/joho/godotenv"

    "github.com/yourorg/github-code-agent/features/webhook"
    "github.com/yourorg/github-code-agent/features/analyzer"
    "github.com/yourorg/github-code-agent/features/reviewer"
    "github.com/yourorg/github-code-agent/features/standards"
    "github.com/yourorg/github-code-agent/features/fixer"
    "github.com/yourorg/github-code-agent/features/gitops"
)

func main() {
    // Load environment variables
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found, using environment variables")
    }

    // Configure AI (supports OpenAI or OpenRouter)
    // For DeepSeek via OpenRouter:
    // export OPENROUTER_API_KEY="sk-or-v1-..."
    // export AI_MODEL="deepseek/deepseek-chat"
    aiConfig := ai.DefaultConfig() // Reads from environment

    // Create AgentField agent
    app, err := agent.New(agent.Config{
        NodeID:        "github-code-agent",
        Version:       "1.0.0",
        TeamID:        "code-review",
        AgentFieldURL: os.Getenv("AGENTFIELD_URL"), // defaults to http://localhost:8080
        AIConfig:      aiConfig,
    })
    if err != nil {
        log.Fatalf("Failed to create agent: %v", err)
    }

    // Register all feature reasoners
    webhook.RegisterReasoners(app)
    analyzer.RegisterReasoners(app)
    reviewer.RegisterReasoners(app)
    standards.RegisterReasoners(app)
    fixer.RegisterReasoners(app)
    gitops.RegisterReasoners(app)

    // Start the agent service
    log.Println("Starting GitHub Code Agent...")
    log.Printf("AgentField URL: %s", os.Getenv("AGENTFIELD_URL"))
    log.Printf("AI Model: %s", aiConfig.Model)

    // Run will automatically handle CLI mode or server mode
    if err := app.Run(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

## 8. Deployment Architecture

### 8.1 Infrastructure

```
┌─────────────────────────────────────────┐
│          GitHub Webhook                  │
│        (webhook.github.com)              │
└────────────────┬────────────────────────┘
                 │ HTTPS
                 ▼
┌─────────────────────────────────────────┐
│       Load Balancer / Ingress            │
│           (your-domain.com)              │
└────────────────┬────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────┐
│      AgentField Control Plane            │
│         (Port 8080)                      │
│  - Webhook routing                       │
│  - Agent discovery                       │
│  - Load balancing                        │
└────────────────┬────────────────────────┘
                 │
        ┌────────┴────────┐
        ▼                 ▼
┌──────────────┐  ┌──────────────┐
│ Agent Pod 1  │  │ Agent Pod 2  │
│ - Orchestrator│  │ - Code Analyzer│
│ - Reviewer   │  │ - Fix Generator│
└──────────────┘  └──────────────┘
```

### 8.2 Scaling Strategy

- **Horizontal Scaling**: Multiple agent instances behind AgentField
- **Queue Management**: Built-in backpressure handling
- **Resource Allocation**:
  - Review Agent: High CPU (AI inference)
  - Code Analyzer: Medium CPU/Memory
  - Git Operations: Low CPU, Medium I/O

### 8.3 Environment Setup

```bash
# Production environment variables
export AGENTFIELD_URL="http://localhost:8080"
export GITHUB_APP_ID="123456"
export GITHUB_PRIVATE_KEY_PATH="/secrets/github-app.pem"
export GITHUB_WEBHOOK_SECRET="your-webhook-secret"
export OPENROUTER_API_KEY="sk-or-v1-your-key"
export AI_MODEL="deepseek/deepseek-chat"
export LOG_LEVEL="info"
```

### 8.4 Docker Deployment

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o github-code-agent ./cmd/agent

FROM alpine:latest
RUN apk --no-cache add ca-certificates git

WORKDIR /root/
COPY --from=builder /app/github-code-agent .

EXPOSE 8080
CMD ["./github-code-agent"]
```

## 9. Monitoring & Observability

### 9.1 Key Metrics

- **Performance**:
  - PR review completion time (target: <2 minutes)
  - Fix generation time
  - GitHub API latency

- **Quality**:
  - Issues detected per PR
  - Fix acceptance rate
  - False positive rate

- **Usage**:
  - PRs reviewed per day
  - Active repositories
  - Agent invocation count

### 9.2 Observability Tools

- **AgentField Dashboard**: Built-in workflow DAG visualization
- **Prometheus Metrics**: Agent performance metrics
- **Structured Logging**: JSON logs for aggregation
- **Verifiable Credentials**: Audit trail for all actions

### 9.3 Alerting

- Review failure rate > 5%
- GitHub API rate limit < 20%
- Average review time > 5 minutes
- Agent crash/restart events

## 10. User Documentation Plan

### 10.1 Getting Started Guide

1. **Installation**: Register GitHub App in your organization
2. **Configuration**: Create `.github/code-agent.yml`
3. **First PR**: Test with a sample PR
4. **Customization**: Adjust standards and rules

### 10.2 Configuration Reference

- Complete YAML schema documentation
- Example configurations for different languages
- Common patterns and recipes

### 10.3 Troubleshooting

- Common issues and solutions
- How to disable agent for specific PRs
- Contact and support information

## 11. Success Criteria

### 11.1 Functional Requirements

- ✅ Automatically reviews PRs within 2 minutes of creation
- ✅ Generates actionable feedback with severity levels
- ✅ Applies fixes in both YOLO and Safe modes
- ✅ Supports configurable coding standards
- ✅ Handles multiple programming languages
- ✅ Posts review comments on GitHub

### 11.2 Quality Requirements

- ✅ <5% false positive rate on issue detection
- ✅ >80% fix acceptance rate (fixes not reverted)
- ✅ Zero security incidents from agent actions
- ✅ 99.5% uptime for webhook handling

### 11.3 Performance Requirements

- ✅ Review completion: <2 minutes for PRs <400 LOC
- ✅ Fix generation: <30 seconds per issue
- ✅ GitHub API rate limit: <50% utilization
- ✅ Memory usage: <512MB per agent pod

## 12. Future Enhancements

### 12.1 Short-term (3-6 months)

- **Learning System**: Train on accepted/rejected fixes
- **Team Preferences**: Per-team configuration overrides
- **IDE Integration**: VS Code extension for local review
- **Merge Conflict Resolution**: Auto-resolve simple conflicts

### 12.2 Long-term (6-12 months)

- **Cross-Repository Analysis**: Detect changes affecting multiple repos
- **Refactoring Suggestions**: Proactive code improvement proposals
- **Architecture Validation**: Ensure changes align with system design
- **Performance Prediction**: Estimate impact of changes on runtime performance
- **Auto-Documentation**: Generate/update docs from code changes

## 13. Risk Assessment

### 13.1 Technical Risks

| Risk                          | Impact | Mitigation                                            |
| ----------------------------- | ------ | ----------------------------------------------------- |
| Bad fixes merged in YOLO mode | High   | Default to Safe mode, require test passage            |
| GitHub API rate limits        | Medium | Implement caching, use GraphQL, monitor limits        |
| AI model hallucinations       | High   | Validate all fixes, require human review for Critical |
| Large PR performance          | Medium | Set LOC limits, incremental processing                |

### 13.2 Operational Risks

| Risk                                 | Impact | Mitigation                         |
| ------------------------------------ | ------ | ---------------------------------- |
| Agent downtime during PR surge       | Medium | Auto-scaling, queue management     |
| Configuration errors breaking builds | High   | Config validation, dry-run mode    |
| Security token compromise            | High   | Rotate tokens, minimal permissions |
| Cost overruns (AI API)               | Low    | Set spending limits, monitor usage |

## 14. Cost Estimation

### 14.1 AI API Costs (DeepSeek)

**Assumptions:**

- 50 PRs/day
- Average 200 LOC per PR
- ~4000 tokens per review (2000 input + 2000 output)

**DeepSeek Chat Pricing:**

- Input: $0.14 per 1M tokens
- Output: $0.28 per 1M tokens
- Cache hits: $0.014 per 1M tokens (90% discount)

**Monthly costs:**

- Without caching: ~$31.50/month
  - Input: (50 PR × 30 days × 2000 tokens × $0.14) / 1M = $0.42
  - Output: (50 PR × 30 days × 2000 tokens × $0.28) / 1M = $0.84
  - Total per day: ~$1.26/day = $37.80/month

- With 50% cache hit rate: ~$20-25/month

**DeepSeek is 85-90% cheaper than GPT-4o and Claude Opus 4.5!**

### 14.2 Infrastructure Costs

- **AgentField Control Plane**: Free (self-hosted)
- **Compute**: ~$50-100/month (2-4 Go agent pods on cloud VMs)
  - Go has lower memory footprint than Python
  - Can handle more concurrent requests
- **Storage**: <$10/month (logs, state)

**Total estimated monthly cost: $80-135** (significantly cheaper with DeepSeek)

## 15. References & Resources

### Documentation

- [GitHub Webhooks Guide](https://www.magicbell.com/blog/github-webhooks-guide)
- [GitHub Webhook Events](https://docs.github.com/en/webhooks/webhook-events-and-payloads)
- [Building GitHub Apps](https://docs.github.com/en/apps/creating-github-apps/writing-code-for-a-github-app/building-a-github-app-that-responds-to-webhook-events)
- [Best Automated Code Review Tools 2026](https://www.qodo.ai/blog/best-automated-code-review-tools-2026/)
- [AI Code Review Best Practices](https://www.qodo.ai/blog/best-ai-code-review-tools-2026/)
- [Code Review Best Practices 2026](https://zencoder.ai/blog/code-review-best-practices)
- [Golang Microservices Best Practices](https://medium.com/@amin-softtech/why-golang-is-ideal-for-microservices-architecture-2026-and-beyond-fd0775c97d4c)
- [Feature-Based Organization](https://medium.com/@smart_byte_labs/organize-like-a-pro-a-simple-guide-to-go-project-folder-structures-e85e9c1769c2)

### AgentField Resources

- [AgentField Documentation](https://github.com/Agent-Field/agentfield)
- [AgentField Website](https://www.agentfield.ai)
- [AgentField Go Example](https://github.com/yongchenglow/agentfield)

### DeepSeek Resources

- [DeepSeek API Documentation](https://api-docs.deepseek.com/)
- [go-deepseek/deepseek - Go Client](https://github.com/go-deepseek/deepseek)
- [DeepSeek Go Package Documentation](https://pkg.go.dev/github.com/go-deepseek/deepseek)

### Go Libraries

- [google/go-github](https://github.com/google/go-github) - GitHub API client
- [go-git/go-git](https://github.com/go-git/go-git) - Git operations in Go
- [gopkg.in/yaml.v3](https://github.com/go-yaml/yaml) - YAML parser

---

**Plan Version**: 2.0
**Last Updated**: 2026-02-07
**Status**: Ready for Implementation
**Technology**: Go 1.21+ with DeepSeek Chat
**Code Organization**: Feature-based structure
**Estimated Timeline**: 10 weeks
**Team Size**: 2-3 developers
