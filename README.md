# GitHub Code Review Agent

An autonomous GitHub code review agent built with Go and [AgentField](https://www.agentfield.ai) that automatically reviews pull requests, identifies issues, and applies fixes.

## Overview

This agent operates in two modes:

- **YOLO mode**: Directly pushes fixes to the source branch
- **Safe mode**: Creates a new PR with fixes for human review

## Implementation Status

### ✅ Phase 1: Foundation (COMPLETE)

- Go project structure with feature-based organization
- Configuration system (YAML + environment variables)
- AgentField SDK integration for AI (supports DeepSeek and any OpenAI-compatible API)
- GitHub API client with App authentication
- Webhook handler with signature validation
- Event debouncing for rapid commits
- CI/CD status checking support
- Code analyzer for PR file analysis
- Language detection
- Main entry point with HTTP server

### ✅ Phase 2: Review Engine (COMPLETE)

- **Code Reviewer** - AI-powered code review using DeepSeek
  - Comprehensive code analysis
  - Security vulnerability detection (OWASP Top 10)
  - Issue categorization (Bug, Security, Performance, Maintainability, Style)
  - Severity classification (Critical, High, Medium, Low)
  - Issue prioritization and deduplication
- **Standards Validator** - Configurable coding standards
  - Line length enforcement
  - Function length checks
  - Naming conventions
  - Documentation requirements
  - Hardcoded secrets detection
- **GitHub Integration**
  - Review comment posting with inline suggestions
  - Emoji-based severity indicators (🔴🟠🟡🔵)
  - Summary comments with issue breakdown
  - Approval reviews when no issues found

### 🚧 Phase 3 (Next): Fix Generation

- Fix generation with AI
- Validation-retry feedback loop (max 3 attempts)
- Git operations (clone, branch, commit, push)
- Safe mode: Create fix PR
- YOLO mode: Direct push
- Comment linking (update reviews with fix links)

## Project Structure

```
github-code-agent/
├── cmd/
│   └── agent/
│       └── main.go                    # Entry point
├── features/
│   ├── webhook/                       # ✅ Webhook handling
│   │   ├── webhook.go
│   │   ├── types.go
│   │   ├── validator.go
│   │   └── reasoners.go
│   ├── analyzer/                      # ✅ Code analysis
│   │   ├── analyzer.go
│   │   ├── types.go
│   │   └── reasoners.go
│   ├── reviewer/                      # ✅ AI code review (Phase 2)
│   │   ├── reviewer.go
│   │   ├── types.go
│   │   ├── reasoners.go
│   │   ├── prioritizer.go
│   │   └── comments.go
│   ├── standards/                     # ✅ Standards validation (Phase 2)
│   │   ├── standards.go
│   │   ├── types.go
│   │   └── reasoners.go
│   ├── fixer/                         # 🚧 Fix generation (Phase 3)
│   └── gitops/                        # 🚧 Git operations (Phase 3)
├── pkg/
│   ├── github/                        # GitHub API client
│   │   └── client.go
│   └── config/                        # Configuration
│       ├── config.go
│       └── types.go
├── .github/
│   └── code-agent.yml                 # Agent configuration
├── .env.example                       # Environment variables template
├── go.mod
├── go.sum
└── README.md
```

## Setup

### Prerequisites

- Go 1.21 or higher
- GitHub App credentials
- AI API key (DeepSeek, OpenRouter, or any OpenAI-compatible API)

### Installation

1. Clone the repository:

```bash
git clone <repository-url>
cd github-code-agent
```

1. Install dependencies:

```bash
go mod download
```

1. Create `.env` file from template:

```bash
cp .env.example .env
```

1. Configure environment variables in `.env`:

```bash
# GitHub Configuration
GITHUB_APP_ID=your-app-id
GITHUB_PRIVATE_KEY_PATH=./github-app.pem
GITHUB_WEBHOOK_SECRET=your-webhook-secret

# DeepSeek AI Configuration
AI_API_KEY=your-deepseek-api-key
AI_BASE_URL=https://api.deepseek.com
AI_MODEL=deepseek-chat

# Application Settings
PORT=8080
LOG_LEVEL=info
```

1. Configure agent behavior in `.github/code-agent.yml`

### GitHub App Setup

1. Create a GitHub App with the following permissions:
   - **Pull requests**: Read & Write
   - **Contents**: Write
   - **Metadata**: Read

2. Subscribe to webhook events:
   - `pull_request`
   - `check_suite`
   - `workflow_run`

3. Set webhook URL to: `https://your-domain.com/webhook`

4. Generate and download the private key

5. Note your App ID and webhook secret

## Running the Agent

### Development

```bash
go run cmd/agent/main.go
```

### Production

```bash
# Build
go build -o github-code-agent cmd/agent/main.go

# Run
./github-code-agent
```

### Docker

```bash
# Build
docker build -t github-code-agent .

# Run
docker run -p 8080:8080 --env-file .env github-code-agent
```

## Configuration

### Agent Modes

- **Safe mode** (default): Creates a new PR with fixes

  ```yaml
  agent:
    mode: safe
  ```

- **YOLO mode**: Pushes fixes directly to PR branch

  ```yaml
  agent:
    mode: yolo
  ```

### Webhook Triggers

Configure which events trigger the agent:

```yaml
webhooks:
  triggers:
    - pull_request.opened
    - pull_request.synchronize
    - check_suite.completed
  wait_for_ci: true
  debounce_seconds: 30
```

### Review Settings

```yaml
review:
  auto_review: true
  auto_fix: true
  severity_threshold: medium
  ignore_paths:
    - "*.md"
    - "docs/**"
```

### Validation Loop

Configure the fix validation feedback loop:

```yaml
validation:
  enabled: true
  max_fix_attempts: 3
  checks:
    - syntax
    - linting
    - formatting
    - security
  timeout_seconds: 30
  auto_format: true
```

## API Endpoints

- `GET /health` - Health check endpoint
- `POST /webhook` - GitHub webhook endpoint

## Development

### Run Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./features/...

# Run specific feature
go test ./features/reviewer/...

# Verbose output
go test -v ./features/...
```

**Current Test Status:**
```
✅ features/analyzer    - All tests passing
✅ features/reviewer    - All tests passing
✅ features/standards   - All tests passing
✅ features/webhook     - All tests passing
```

### Run with Hot Reload

```bash
# Install air
go install github.com/cosmtrek/air@latest

# Run
air
```

## Architecture

The agent uses a feature-based architecture where each feature is self-contained:

- **Webhook Handler**: Receives and validates GitHub webhooks
- **Code Analyzer**: Fetches and analyzes PR files
- **Code Reviewer**: AI-powered code review (Phase 2)
- **Standards Validator**: Checks coding standards (Phase 2)
- **Fix Generator**: Generates and validates fixes (Phase 2)
- **Git Operations**: Manages commits and PRs (Phase 2)

## Roadmap

### Phase 1 ✅ Foundation (Week 1-2) - COMPLETE

- ✅ Project setup and configuration
- ✅ Webhook handling with debouncing
- ✅ GitHub API integration
- ✅ Code analyzer with language detection
- ✅ CI/CD status checking

### Phase 2 ✅ Review Engine (Week 3-4) - COMPLETE

- ✅ AI-powered code review using DeepSeek
- ✅ Security vulnerability detection
- ✅ Standards validation with 5+ built-in rules
- ✅ Issue prioritization and deduplication
- ✅ GitHub review comment posting
- ✅ 19 unit tests all passing

### Phase 3 🚧 Fix Generation (Week 5-6) - NEXT

- 🚧 AI-powered fix generation
- 🚧 Validation-retry feedback loop (max 3 attempts)
- 🚧 Syntax, linting, formatting validation
- 🚧 Git operations (clone, branch, commit, push)
- 🚧 Safe mode: Create fix PR
- 🚧 YOLO mode: Direct push
- 🚧 Comment linking with fix references

### Phase 4 📋 Polish & Testing (Week 7-8) - PLANNED

- 📋 Integration tests
- 📋 Performance optimization
- 📋 Documentation
- 📋 Monitoring and observability

### Phase 5 🎯 Advanced Features (Week 9-10) - PLANNED

- 🎯 Multi-repository support
- 🎯 Learning from feedback
- 🎯 Custom rule marketplace

## Contributing

Contributions are welcome! Please read our contributing guidelines.

## License

[Your License Here]

## Documentation

- 📋 [Plan.md](Plan.md) - Complete implementation plan with architecture details
- 📊 [PHASE2_SUMMARY.md](PHASE2_SUMMARY.md) - Phase 2 implementation summary
- 📂 [features/README.md](features/README.md) - Feature-specific documentation
- 📈 [STATUS.md](STATUS.md) - Current implementation status

## Key Features

### AI-Powered Review
- Uses DeepSeek Chat for intelligent code analysis
- Structured prompting for consistent, actionable feedback
- Temperature: 0.2 for balanced creativity and consistency
- Supports any OpenAI-compatible API

### Security Analysis
- OWASP Top 10 vulnerability detection
- Hardcoded secrets scanning (API keys, passwords, tokens)
- CWE and OWASP classification
- Remediation suggestions

### Issue Prioritization
- Automatic severity classification
- Category-based grouping
- Deduplication of similar issues
- Threshold-based filtering

### GitHub Integration
- Review comments with inline suggestions
- Summary comments with issue breakdown
- Emoji severity indicators (🔴🟠🟡🔵)
- Approval reviews for clean code

## Support

For issues and questions:

- GitHub Issues: [repository-url]/issues
- Documentation: See docs above
- AgentField: https://www.agentfield.ai
