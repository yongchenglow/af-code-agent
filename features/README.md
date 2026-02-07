# GitHub Code Review Agent - Features

This directory contains the feature modules for the GitHub Code Review Agent, organized following a feature-based architecture.

## Implemented Features

### Phase 1: Foundation ✅

#### Webhook Handler (`features/webhook/`)
- Receives and validates GitHub webhook events
- Authenticates webhook signatures (HMAC validation)
- Routes requests to appropriate agent workflows
- Supports debouncing for rapid commits
- CI/CD awareness (waits for pipeline completion)

**Key Files:**
- `webhook.go` - Main webhook handler
- `validator.go` - Signature validation
- `reasoners.go` - AgentField reasoners
- `types.go` - Webhook event types

#### Code Analyzer (`features/analyzer/`)
- Fetches PR content and file diffs from GitHub
- Parses code structure
- Identifies changed files and their relationships
- Extracts code metrics
- Language detection

**Key Files:**
- `analyzer.go` - PR analysis logic
- `reasoners.go` - AgentField reasoners
- `types.go` - Analysis result types

### Phase 2: Review Engine ✅

#### Code Reviewer (`features/reviewer/`)
- **AI-Powered Review**: Uses DeepSeek Chat via AgentField SDK for intelligent code analysis
- **Security Analysis**: Specialized security vulnerability detection (OWASP Top 10)
- **Issue Detection**: Identifies bugs, security vulnerabilities, performance problems, and maintainability concerns
- **GitHub Integration**: Posts review comments with proper formatting

**Key Components:**

1. **Reviewer (`reviewer.go`)**
   - `ReviewCode()` - Comprehensive code review using AI
   - `DetectSecurityIssues()` - Specialized security vulnerability detection
   - Smart prompting for DeepSeek with structured JSON output
   - Automatic file filtering (skips binaries, vendor, node_modules)

2. **Issue Prioritization (`prioritizer.go`)**
   - `PrioritizeIssues()` - Sorts issues by severity and category
   - `FilterByThreshold()` - Filters issues based on severity threshold
   - Deduplication of similar issues
   - Severity levels: Critical, High, Medium, Low
   - Categories: Security, Bug, Performance, Maintainability, Style

3. **Comment Posting (`comments.go`)**
   - `PostReview()` - Creates complete PR review with inline comments
   - `PostInlineComment()` - Adds single inline comment
   - `UpdateComment()` - Updates existing review comment
   - `PostSummaryComment()` - Posts summary comment on PR
   - Emoji-based severity indicators (🔴 Critical, 🟠 High, 🟡 Medium, 🔵 Low)

**Example Usage:**
```go
reviewer := NewReviewer(agentInstance)

// Perform review
report, err := reviewer.ReviewCode(ctx, files, prContext)

// Detect security issues
securityIssues, err := reviewer.DetectSecurityIssues(ctx, files)

// Prioritize all issues
prioritized := PrioritizeIssues(report.Issues, violations, securityIssues)

// Post review to GitHub
poster := NewCommentPoster(githubClient)
err = poster.PostReview(ctx, owner, repo, prNumber, prioritized, commitSHA)
```

#### Standards Validator (`features/standards/`)
- **Configurable Rules**: Validates code against configured coding standards
- **Multi-Language Support**: Go, Python, JavaScript/TypeScript
- **Built-in Rules**: Line length, function length, naming conventions, documentation, security

**Key Components:**

1. **Validator (`standards.go`)**
   - `ValidateStandards()` - Checks code against all enabled rules
   - Built-in rule engine with extensible architecture
   - Language-specific validation logic

2. **Built-in Rules:**
   - **Line Length**: Enforces maximum line length
   - **Function Length**: Checks function/method length limits
   - **Naming Conventions**: Validates identifier naming patterns
   - **Documentation**: Checks for missing docstrings/comments (Go, Python)
   - **Hardcoded Secrets**: Detects API keys, passwords, tokens

**Configuration (`.github/code-agent.yml`):**
```yaml
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

  security:
    check_dependencies: true
    check_secrets: true
    owasp_checks: true
```

**Example Usage:**
```go
validator := NewValidator(config)

// Validate files
report, err := validator.ValidateStandards(ctx, files)

// Check results
fmt.Printf("Found %d violations\n", report.TotalViolations)
for _, violation := range report.Violations {
    fmt.Printf("%s:%d - %s\n", violation.FilePath, violation.Line, violation.Message)
}
```

## AgentField Integration

All features are implemented as AgentField reasoners, allowing for:
- **Distributed Execution**: Run on multiple nodes
- **Workflow Orchestration**: Chain reasoners together
- **State Management**: Use AgentField's hierarchical memory
- **AI Integration**: Built-in DeepSeek access via `agent.AI()`

### Registered Reasoners

**Webhook:**
- `handle_webhook` - Main webhook handler
- `handle_pr_opened` - PR-specific workflow orchestration

**Analyzer:**
- `analyze_pr` - Analyzes PR files and extracts metrics
- `parse_code_structure` - Parses code into AST
- `calculate_complexity` - Calculates code metrics

**Reviewer (Phase 2):**
- `review_code` - AI-powered code review
- `detect_security_issues` - Security vulnerability detection

**Standards (Phase 2):**
- `validate_standards` - Standards validation

## Testing

All features include comprehensive unit tests:

```bash
# Run all tests
go test ./features/...

# Run specific feature tests
go test ./features/reviewer/...
go test ./features/standards/...

# Run with coverage
go test -cover ./features/...
```

**Current Test Coverage:**
- Webhook: ✅ Basic validation tests
- Analyzer: ✅ File parsing and analysis
- Reviewer: ✅ Filtering, prioritization, comment formatting
- Standards: ✅ Rule validation, secret detection, line/function length

## Architecture Patterns

### Feature-Based Organization
Each feature is self-contained with:
- Main logic (`*.go`)
- Type definitions (`types.go`)
- Reasoner registration (`reasoners.go`)
- Unit tests (`*_test.go`)

### AI Integration Pattern
```go
// Use AgentField's built-in AI method
response, err := agent.AI(ctx, prompt,
    ai.WithSystem("System prompt..."),
    ai.WithTemperature(0.2),
    ai.WithMaxTokens(4000))

// Parse structured response
report := parseReviewResponse(response.Text())
```

### GitHub API Pattern
```go
// Use go-github for API calls
client := github.NewClient(nil)

// Fetch PR files
files, _, err := client.PullRequests.ListFiles(ctx, owner, repo, prNumber, nil)

// Post review
review := &github.PullRequestReviewRequest{...}
client.PullRequests.CreateReview(ctx, owner, repo, prNumber, review)
```

## Next Steps

### Phase 3: Fix Generation (Upcoming)
- `features/fixer/` - AI-powered fix generation with validation loop
- `features/gitops/` - Git operations (branch, commit, push, PR creation)

### Phase 4: Polish & Testing
- Performance optimization
- Comprehensive integration tests
- End-to-end PR testing

### Phase 5: Advanced Features
- Multi-repository support
- Custom rule marketplace
- Learning from feedback

## References

- [AgentField Documentation](https://github.com/Agent-Field/agentfield)
- [Plan.md](../Plan.md) - Complete implementation plan
- [GitHub API](https://docs.github.com/en/rest)
- [DeepSeek API](https://api-docs.deepseek.com/)
