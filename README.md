# GitHub Code Review Agent

An autonomous GitHub code review agent built with Go and [AgentField](https://www.agentfield.ai) that automatically reviews pull requests, identifies issues, and applies fixes.

## Overview

This agent operates in two modes:

- **YOLO mode**: Directly pushes fixes to the source branch
- **Safe mode**: Creates a new PR with fixes for human review

## Phase 1 Implementation Status

✅ **Completed**:

- Go project structure with feature-based organization
- Configuration system (YAML + environment variables)
- DeepSeek AI client wrapper (placeholder for API integration)
- GitHub API client with App authentication
- Webhook handler with signature validation
- Event debouncing for rapid commits
- CI/CD status checking support
- Code analyzer for PR file analysis
- Language detection
- Main entry point with HTTP server

🚧 **Phase 2** (Next):

- AI-powered code review engine
- Standards validation
- Fix generation with validation loop
- Git operations
- Review comment posting

## Project Structure

```
github-code-agent/
├── cmd/
│   └── agent/
│       └── main.go                    # Entry point
├── features/
│   ├── webhook/                       # Webhook handling
│   │   ├── webhook.go
│   │   ├── types.go
│   │   └── validator.go
│   ├── analyzer/                      # Code analysis
│   │   ├── analyzer.go
│   │   └── types.go
│   ├── reviewer/                      # Code review (Phase 2)
│   ├── standards/                     # Standards validation (Phase 2)
│   ├── fixer/                         # Fix generation (Phase 2)
│   └── gitops/                        # Git operations (Phase 2)
├── pkg/
│   ├── github/                        # GitHub API client
│   │   └── client.go
│   ├── deepseek/                      # DeepSeek AI client
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
- DeepSeek API key

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
go test ./...
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

### Phase 1 ✅ (Complete)

- Project setup and configuration
- Webhook handling
- GitHub API integration
- Code analyzer

### Phase 2 (Next)

- AI-powered code review
- Standards validation
- Fix generation with validation loop
- Git operations

### Phase 3

- Polish and testing
- Performance optimization
- Documentation

### Phase 4

- Advanced features
- Multi-repository support
- Learning from feedback

## Contributing

Contributions are welcome! Please read our contributing guidelines.

## License

[Your License Here]

## Support

For issues and questions:

- GitHub Issues: [repository-url]/issues
- Documentation: [docs-url]
