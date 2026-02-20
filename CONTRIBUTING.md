# Contributing to GitHub Code Review Agent

Thank you for your interest in contributing to the GitHub Code Review Agent project! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Code Style](#code-style)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)
- [Commit Guidelines](#commit-guidelines)
- [Architecture](#architecture)
- [Questions?](#questions)

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Welcome newcomers and help them learn
- Keep discussions professional and on-topic

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally
   ```bash
   git clone https://github.com/YOUR_USERNAME/af-code-agent.git
   cd af-code-agent
   ```
3. **Add upstream remote**
   ```bash
   git remote add upstream https://github.com/yongchenglow/af-code-agent.git
   ```
4. **Create a branch** for your work
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Setup

### Prerequisites

- Go 1.24 or higher
- Git
- Docker (optional, for containerized testing)
- Kubernetes (optional, for deployment testing)

### Install Dependencies

```bash
make install
```

### Set Up Environment

```bash
cp .env.example .env
# Edit .env with your configuration
```

### Verify Setup

```bash
# Run tests
make test

# Run linter
make lint-check

# Build the project
make build
```

## Code Style

### Go Conventions

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting (automatically run by `make fmt`)
- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use meaningful variable and function names
- Keep functions focused and small (< 50 lines preferred)

### Linting

We use `golangci-lint` with the following enabled linters:
- errcheck
- goconst
- gocritic
- gofmt
- gosec
- gosimple
- govet
- ineffassign
- misspell
- staticcheck
- unused
- revive (for exported rules)

Run linting:
```bash
# Auto-fix issues
make lint

# Check without fixes (CI mode)
make lint-check
```

### Documentation

- All exported functions must have godoc comments
- Comments should explain **why**, not **what**
- Include examples for complex functions
- Update README for user-visible changes

Example:
```go
// PlanReview performs comprehensive code review and creates a fix plan.
//
// The planner analyzes code changes using AI to identify:
//   - Security vulnerabilities (OWASP Top 10)
//   - Logic bugs and edge cases
//   - Standards violations
//   - Test gaps
//
// Returns a prioritized fix plan with dependency tracking.
func (p *Planner) PlanReview(ctx context.Context, files []*analyzer.FileChange, prContext map[string]any) (*ReviewPlan, error) {
```

## Testing

### Test Requirements

- **Minimum coverage**: 60% overall
- **New code**: 80% coverage minimum
- **Critical paths**: Must be tested
- **Table-driven tests**: Use for functions with multiple cases

### Running Tests

```bash
# All tests
make test

# With coverage
make test-coverage

# Check coverage threshold
make test-coverage-check

# Integration tests
make test-integration

# With race detector
make test-race

# Specific package
go test ./agents/reviewer/...

# Verbose output
go test -v ./...
```

### Writing Tests

#### Table-Driven Tests (Preferred)

```go
func TestDetectLanguage(t *testing.T) {
    tests := []struct {
        name     string
        filename string
        expected string
    }{
        {"go file", "main.go", "go"},
        {"python file", "script.py", "python"},
        {"unknown", "README.md", "unknown"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := DetectLanguage(tt.filename)
            if got != tt.expected {
                t.Errorf("DetectLanguage(%q) = %q, want %q", tt.filename, got, tt.expected)
            }
        })
    }
}
```

#### Test Naming

- Use descriptive names: `TestFunctionName_Condition_ExpectedBehavior`
- Examples:
  - `TestParseReviewResponse_ValidJSON`
  - `TestParseReviewResponse_InvalidJSON`
  - `TestBuildReviewPrompt_WithPRTitle`

#### Mocking

For external dependencies (AI, GitHub API), use mocks:

```go
type MockAgent struct {
    AIResponse string
    AIError    error
}

func (m *MockAgent) AI(ctx context.Context, prompt string, opts ...any) (any, error) {
    if m.AIError != nil {
        return nil, m.AIError
    }
    return &MockResponse{Text: m.AIResponse}, nil
}
```

## Pull Request Process

### Before Submitting

1. **Update documentation** for user-visible changes
2. **Add tests** for new functionality
3. **Run all tests** and ensure they pass
4. **Run linter** and fix all issues
5. **Check coverage** meets threshold
6. **Update CHANGELOG.md** with your changes

### PR Title Format

Use conventional commits format:
- `feat: Add secret scanning for AWS keys`
- `fix: Handle nil pointer in reviewer`
- `docs: Update README with examples`
- `test: Add tests for planner package`
- `refactor: Simplify validation logic`
- `chore: Update dependencies`

### PR Description Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to change)
- [ ] Documentation update

## Testing
- [ ] Tests added/updated
- [ ] Tests pass locally
- [ ] Integration tests pass

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No new warnings
- [ ] CHANGELOG updated
```

### Review Process

1. **Automated Checks**: CI must pass (tests, lint, coverage)
2. **Code Review**: At least one maintainer review required
3. **Changes**: Address all review comments
4. **Merge**: Squash and merge by maintainer

### Review Response Time

- We aim to review PRs within 48 hours
- If no response after 3 days, feel free to ping @yongchenglow

## Commit Guidelines

### Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Formatting (non-functional)
- `refactor`: Code restructuring
- `test`: Tests
- `chore`: Maintenance

### Examples

```
feat(security): Add secret scanning for GitHub tokens

Implement regex-based detection for:
- GitHub PAT tokens (ghp_*)
- GitHub OAuth tokens (gho_*)
- GitHub App tokens (ghs_*, ghru_*)

Fixes: #123

Reviewed-by: @reviewer
```

```
fix(reviewer): Handle empty file list in PlanReview

Return empty review plan instead of panicking when
no reviewable files are found.

Fixes: #456
```

## Architecture

### Project Structure

```
af-code-agent/
├── cmd/agent/           # Entry point
├── agents/              # Feature-based agents
│   ├── analyzer/        # PR analysis
│   ├── planner/         # Review planning
│   ├── reviewer/        # AI review
│   ├── standards/       # Standards validation
│   ├── fixer/           # Fix generation
│   ├── security/        # Security scanning
│   ├── testexec/        # Test generation
│   ├── workflow/        # Orchestration
│   └── webhook/         # Webhook handling
├── pkg/                 # Shared packages
│   ├── app/             # Application bootstrap
│   ├── config/          # Configuration
│   ├── context/         # Context management
│   ├── constants/       # Constants
│   ├── github/          # GitHub client
│   ├── logger/          # Logging
│   ├── middleware/      # HTTP middleware
│   ├── utils/           # Utilities
│   ├── validator/       # Validation
│   ├── circuitbreaker/  # Circuit breaker
│   └── retry/           # Retry logic
├── tests/               # Integration tests
├── docs/                # Documentation
│   └── adr/             # Architecture Decision Records
└── helm/                # Kubernetes charts
```

### Key Design Decisions

See [docs/adr/](docs/adr/) for Architecture Decision Records:
- ADR-001: AI Model Selection
- ADR-002: Agent Architecture
- ADR-003: GitHub App Authentication
- ADR-004: Validation Loop Design

## Questions?

- **General questions**: Open a [Discussion](https://github.com/yongchenglow/af-code-agent/discussions)
- **Bug reports**: Open an [Issue](https://github.com/yongchenglow/af-code-agent/issues)
- **Security issues**: Email security@example.com (do not open public issue)

## Thank You!

Your contributions make this project better for everyone. We appreciate your time and effort!

---

*Last updated: 2026-02-19*
