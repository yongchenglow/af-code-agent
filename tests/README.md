# Testing Infrastructure

This directory contains the testing infrastructure for the GitHub Code Review Agent.

## Directory Structure

```
tests/
├── mocks/                      # Mock implementations
│   ├── github_mock.go          # Mock GitHub API client
│   └── ai_mock.go              # Mock AI client
├── integration/                # Integration tests
│   └── workflow_test.go.example # Example integration test
├── e2e/                        # End-to-end tests
│   └── pr_review_test.go.example # Example E2E test
└── README.md                   # This file
```

## Mock Infrastructure

### GitHub Mock (`mocks/github_mock.go`)

Provides a mock GitHub API client for testing without making real API calls.

**Features:**
- Mock PR data
- Mock file changes
- Mock comments
- Mock CI status
- Configurable responses
- Error simulation

**Usage:**

```go
import "github.com/yourorg/github-code-agent/tests/mocks"

func TestMyFunction(t *testing.T) {
    // Create mock client with default data
    githubClient := mocks.NewMockGitHubClient()

    // Customize data
    githubClient.CIStatus = "failure"
    githubClient.PRErr = errors.New("API error")

    // Use in tests
    pr, err := githubClient.GetPR(ctx, "owner", "repo", 123)
}
```

### AI Mock (`mocks/ai_mock.go`)

Provides a mock AI client for testing without calling real AI APIs.

**Features:**
- Canned responses for review/security/fix
- Custom response configuration
- Call tracking
- Response parsing helpers

**Usage:**

```go
import "github.com/yourorg/github-code-agent/tests/mocks"

func TestReview(t *testing.T) {
    // Create mock AI client
    aiClient := mocks.NewMockAIClient()

    // Set custom response
    aiClient.SetResponse("review", `{"issues": [...]}`, nil)

    // Use in tests
    response, err := aiClient.Call(ctx, "review", "Review this code")

    // Check call count
    if aiClient.GetCallCount() != 1 {
        t.Error("expected 1 AI call")
    }
}
```

## Integration Tests (Examples)

Integration tests verify that multiple components work together correctly.

**Location**: `integration/workflow_test.go.example`

**Test Scenarios:**

1. **Complete PR Review Flow**
   - Analyzes PR files
   - Runs standards validation
   - Performs AI code review
   - Verifies issue prioritization

2. **YOLO Mode Workflow**
   - Reviews code
   - Generates fixes
   - Simulates direct push

3. **Safe Mode Workflow**
   - Reviews code
   - Generates fixes
   - Simulates PR creation

4. **Issue Deduplication**
   - Tests duplicate removal
   - Verifies unique issues preserved

5. **Severity Prioritization**
   - Tests issue sorting
   - Verifies Critical → High → Medium → Low order

6. **Configuration Validation**
   - Tests valid config acceptance
   - Tests invalid config rejection

**Running Integration Tests:**

```bash
# Rename example to active test
mv integration/workflow_test.go.example integration/workflow_test.go

# Run integration tests
go test ./tests/integration/... -v

# Rename back to example
mv integration/workflow_test.go integration/workflow_test.go.example
```

## End-to-End Tests (Examples)

E2E tests simulate complete real-world scenarios from webhook to PR comment.

**Location**: `e2e/pr_review_test.go.example`

**Test Scenarios:**

1. **Complete PR Review with Fixes**
   - Simulates webhook event
   - Fetches PR details
   - Analyzes files
   - Checks CI status
   - Performs code review
   - Posts review comments
   - Generates fixes
   - Creates fix PR (Safe mode)
   - Updates comments with links

2. **YOLO Mode - Direct Push**
   - Reviews code
   - Generates fixes
   - Pushes directly to PR branch

3. **Security Scan Workflow**
   - Performs security vulnerability detection
   - Flags critical issues

4. **Large Repo Performance**
   - Simulates 50-file PR
   - Measures processing time
   - Verifies performance targets

**Running E2E Tests:**

```bash
# Rename example to active test
mv e2e/pr_review_test.go.example e2e/pr_review_test.go

# Run E2E tests (may take longer)
go test ./tests/e2e/... -v

# Skip in short mode
go test ./tests/e2e/... -v -short  # E2E tests are skipped

# Rename back to example
mv e2e/pr_review_test.go e2e/pr_review_test.go.example
```

## Writing New Tests

### Unit Tests

Place unit tests next to the code they test:

```
features/analyzer/
├── analyzer.go
└── analyzer_test.go  # Unit tests for analyzer.go
```

### Integration Tests

Add to `tests/integration/`:

```go
package integration

import (
    "testing"
    "github.com/yourorg/github-code-agent/tests/mocks"
)

func TestMyIntegration(t *testing.T) {
    // Use mocks
    githubClient := mocks.NewMockGitHubClient()
    aiClient := mocks.NewMockAIClient()

    // Test workflow
    // ...
}
```

### E2E Tests

Add to `tests/e2e/`:

```go
package e2e

import (
    "testing"
    "time"
)

func TestMyE2E(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()

    // Test complete workflow
    // ...
}
```

## Test Coverage

Current test coverage:

```
Package                  Coverage    Tests
features/analyzer          20.0%       2
features/fixer            11.2%       5
features/gitops            7.4%       6
features/reviewer         30.6%       8
features/standards        50.6%       6
features/webhook          17.3%       3
pkg/config                57.8%       3
pkg/github                65.2%       9
pkg/performance           49.6%      11
--------------------------------------------------
Total                      ~30%      50+
```

**Improving Coverage:**

```bash
# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Open in browser
open coverage.html
```

## Best Practices

### 1. Use Table-Driven Tests

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:  "valid input",
            input: "hello",
            want:  "HELLO",
            wantErr: false,
        },
        {
            name:  "empty input",
            input: "",
            want:  "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### 2. Use Mocks for External Dependencies

Never make real API calls in tests:

```go
// Good
githubClient := mocks.NewMockGitHubClient()

// Bad
githubClient := github.NewClient(nil)
```

### 3. Test Error Cases

```go
func TestErrorHandling(t *testing.T) {
    // Test success case
    result, err := MyFunction("valid")
    if err != nil {
        t.Fatal(err)
    }

    // Test error case
    _, err = MyFunction("invalid")
    if err == nil {
        t.Error("expected error but got nil")
    }
}
```

### 4. Use Context with Timeouts

```go
func TestWithTimeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    result, err := MySlowFunction(ctx)
    // ...
}
```

### 5. Clean Up Resources

```go
func TestWithCleanup(t *testing.T) {
    tmpFile, err := os.CreateTemp("", "test-*.txt")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tmpFile.Name())  // Clean up

    // Use tmpFile in test
    // ...
}
```

## Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package
go test ./features/analyzer/...

# Run specific test
go test -run TestAnalyzePR ./features/analyzer/...

# Run with coverage
go test -cover ./...

# Run in short mode (skip slow tests)
go test -short ./...

# Run with race detection
go test -race ./...

# Parallel execution
go test -parallel 4 ./...
```

## Continuous Integration

Tests run automatically on:
- Every commit
- Every pull request
- Before deployment

**CI Configuration** (example for GitHub Actions):

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: go test -v -race -coverprofile=coverage.out ./...
      - uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
```

## Troubleshooting

### Tests Fail with "Too Many Open Files"

Increase file descriptor limit:

```bash
ulimit -n 4096
```

### Tests Timeout

Increase timeout:

```bash
go test -timeout 5m ./...
```

### Race Condition Detected

Fix race conditions or skip race detection:

```bash
# Run without race detection (not recommended)
go test ./...
```

### Import Cycle

Restructure code to break circular dependencies.

## Resources

- [Go Testing Documentation](https://pkg.go.dev/testing)
- [Table-Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Go Test Coverage](https://go.dev/blog/cover)
- [testify Package](https://github.com/stretchr/testify) (optional, for assertions)

---

For questions or issues with tests, please open an issue on GitHub.
