# GitHub Code Review Agent

An autonomous GitHub code review agent built with Go and AgentField that automatically reviews pull requests, identifies issues, and applies fixes using AI.

[![Status](https://img.shields.io/badge/status-production%20ready-success)](https://github.com)
[![Go](https://img.shields.io/badge/go-1.24-blue)](https://go.dev/)
[![Tests](https://img.shields.io/badge/tests-passing-success)](https://github.com)
[![License](https://img.shields.io/badge/license-CC%20BY--NC%204.0-blue)](LICENSE)

## Quick Links

- **📖 [Quick Start Guide](docs/QUICK_START.md)** - Get started in 5 minutes
- **📖 [User Guide](docs/USER_GUIDE.md)** - How to use the agent
- **📖 [GitHub App Setup](docs/GITHUB_APP_SETUP.md)** - Configure GitHub App
- **📖 [Configuration Reference](docs/CONFIGURATION_REFERENCE.md)** - All options
- **📖 [Deployment Guide](docs/DEPLOYMENT.md)** - Production deployment
- **📖 [API Reference](docs/API_REFERENCE.md)** - Technical documentation

## Overview

The GitHub Code Review Agent autonomously:

1. 🔍 **Reviews code** - AI-powered analysis using DeepSeek
2. 🛡️ **Detects vulnerabilities** - OWASP Top 10 security scanning
3. 📏 **Validates standards** - Enforces coding conventions
4. 🔧 **Generates fixes** - Auto-fixes with validation loop
5. 💬 **Posts reviews** - Detailed PR comments with severity levels
6. ✅ **Applies fixes** - Direct push (YOLO) or PR (Safe mode)

## Features

### AI-Powered Code Review

- Comprehensive analysis: bugs, security, performance, maintainability
- DeepSeek integration (~$0.14/1M tokens input, ~$0.28/1M output)
- Structured output with severity categorization
- Smart filtering: skips binary files, vendor, node_modules

### Security Scanning

- OWASP Top 10 vulnerability detection
- Secret detection (API keys, passwords, credentials)
- CWE classification
- Actionable remediation suggestions

### Standards Validation

- Configurable rules: line length, function length, naming
- Multi-language support: Go, Python, JavaScript/TypeScript
- Documentation checks: docstrings, type hints, comments
- Auto-fixable violations

### Automated Fix Generation

- Validation loop with max 3 attempts per fix
- Comprehensive validation:
  - Syntax checking (language-specific)
  - Linting (golangci-lint, flake8, eslint)
  - Formatting (gofmt, black, prettier)
  - Security scanning
  - Issue verification
- Smart retry with error feedback
- Bounded execution prevents infinite loops

### Operating Modes

| Mode  | Description | Best For |
| ----- | ----------- | -------- |
| **Safe** (default) | Creates PR with fixes for human review | Production repos, teams new to agent |
| **YOLO** | Pushes fixes directly to PR branch | Personal repos, trusted codebases |

### Reliability Features

- **Circuit Breaker**: Protects against AI and GitHub API failures
  - Opens after 5 consecutive failures
  - Auto-recovers after 1 minute timeout
  - Graceful error messages when unavailable

- **Retry Logic**: Handles transient failures gracefully
  - Exponential backoff (1s initial, 30s max delay)
  - Context-aware cancellation
  - Different policies for AI vs GitHub API calls

- **Health Checks**: Comprehensive health monitoring
  - `/health/live` - Liveness probe (process alive)
  - `/health/ready` - Readiness probe (dependencies healthy)
  - `/health/started` - Startup probe (initialization complete)

### Security Features

- **Secret Scanning**: Detects exposed credentials
  - AWS access keys and secret keys
  - GitHub tokens (PAT, OAuth, App tokens)
  - Private keys (RSA, EC, OPENSSH)
  - Generic secrets and API keys

- **Webhook Validation**: Comprehensive input validation
  - Signature verification
  - Payload size limits (max 10MB)
  - Content type validation
  - Rate limiting per repository (10 req/min)

- **Security Headers**: Hardened HTTP responses
  - X-Content-Type-Options: nosniff
  - X-Frame-Options: DENY
  - X-XSS-Protection: 1; mode=block
  - Content-Security-Policy: default-src 'none'

## Architecture

```mermaid
flowchart TB
    A[GitHub Webhook] --> B[Webhook Handler<br/>Validates signature, routes events]
    B --> C[Code Analyzer<br/>Analyzes PR files, metrics]
    C --> D[Reviewer]
    C --> E[Standards<br/>AI + deterministic validation]
    D --> F[Fixer<br/>Generates & validates fixes]
    E --> F
    F --> G[GitOps<br/>Branch, commit, push, PR]
    
    style A fill:#e1f5ff
    style B fill:#fff3cd
    style C fill:#e1f5ff
    style D fill:#d4edda
    style E fill:#d4edda
    style F fill:#f8d7da
    style G fill:#d1ecf1
```

### Technology Stack

| Component       | Technology                |
| --------------- | ------------------------- |
| Language        | Go 1.24                   |
| Framework       | AgentField SDK            |
| AI Model        | DeepSeek Chat             |
| GitHub API      | google/go-github          |
| Configuration   | YAML + Environment Vars   |
| Deployment      | Docker + Kubernetes/Helm  |

## Project Structure

```
af-code-agent/
├── cmd/
│   └── agent/
│       └── main.go                    # Entry point
├── agents/                            # Feature-based agents
│   ├── webhook/                       # GitHub webhook handling
│   ├── analyzer/                      # PR analysis
│   ├── reviewer/                      # AI code review
│   ├── standards/                     # Standards validation
│   ├── fixer/                         # Fix generation
│   └── gitops/                        # Git operations
├── pkg/
│   ├── app/                           # Application bootstrap
│   ├── config/                        # Configuration
│   ├── github/                        # GitHub API client
│   ├── performance/                   # Caching, parallel exec
│   ├── context/                       # Context management
│   ├── constants/                     # Constants
│   └── utils/                         # Utilities
├── helm/agentfield/                   # Kubernetes Helm chart
├── docs/                              # Documentation
├── .github/
│   ├── code-agent.yml                 # Agent configuration
│   └── workflows/                     # CI/CD pipelines
├── Dockerfile                         # Multi-stage build
├── Makefile                           # Build automation
└── README.md                          # This file
```

## Getting Started

### Prerequisites

- Go 1.24 or higher
- GitHub App credentials (App ID, Private Key)
- DeepSeek API key (or OpenAI-compatible API)
- Git installed
- Docker (optional)
- Kubernetes (optional, for production)

### Quick Start (Local Development)

1. **Clone and build:**

   ```bash
   git clone https://github.com/yongchenglow/af-code-agent.git
   cd af-code-agent
   go build -o github-code-agent ./cmd/agent
   ```

2. **Configure environment:**

   ```bash
   cp .env.example .env
   # Edit .env with your credentials
   ```

3. **Set up GitHub App:**

   See [GitHub App Setup Guide](docs/GITHUB_APP_SETUP.md)

4. **Run the agent:**

   ```bash
   ./github-code-agent
   ```

5. **Test with ngrok:**

   ```bash
   ngrok http 8001
   # Update GitHub App webhook URL with ngrok HTTPS URL
   ```

For detailed instructions, see [Quick Start Guide](docs/QUICK_START.md).

## Configuration

### Environment Variables (.env)

```bash
# GitHub App Configuration
GITHUB_APP_ID=123456
GITHUB_PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----"
GITHUB_WEBHOOK_SECRET=your-secret

# AI Configuration
OPENAI_API_KEY=your-deepseek-api-key
AI_BASE_URL=https://api.deepseek.com
AI_MODEL=deepseek-chat

# Application Settings
LOG_LEVEL=info
PORT=8001
```

### Repository Configuration (.github/code-agent.yml)

```yaml
agent:
  enabled: true
  mode: safe # or "yolo"

review:
  auto_review: true
  auto_fix: true
  severity_threshold: medium

webhooks:
  wait_for_ci: true
  debounce_seconds: 30

validation:
  enabled: true
  max_fix_attempts: 3
  checks:
    - syntax
    - linting
    - formatting
    - security
```

See [Configuration Reference](docs/CONFIGURATION_REFERENCE.md) for all options.

## Usage

### Example Review Comment

The agent posts comments with severity indicators:

```
🔴 Critical: Potential nil pointer dereference

Details:
The function `main` doesn't check if `user` is nil before accessing
the `Name` field, which will cause a panic.

Suggested fix:
Add nil check: if user != nil { ... }

Automated review by GitHub Code Agent
```

### Severity Levels

| Emoji | Severity | Auto-Fix           | Examples                                     |
| ----- | -------- | ------------------ | -------------------------------------------- |
| 🔴    | Critical | ✓ (Safe mode only) | Security vulnerabilities, breaking changes   |
| 🟠    | High     | ✓                  | Bugs, performance issues, major code smells  |
| 🟡    | Medium   | ✓                  | Maintainability concerns, minor improvements |
| 🔵    | Low      | ✓                  | Style issues, documentation improvements     |

### Control Per-PR with Labels

- `agent:safe` - Force safe mode
- `agent:yolo` - Force YOLO mode
- `agent:skip` - Skip automated review
- `agent:review-only` - Review but don't auto-fix

## Development

### Run Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./agents/...

# Run specific agent
go test ./agents/reviewer/...

# Verbose output
go test -v ./agents/...

# Check coverage threshold (60%)
make test-coverage-check

# View coverage by function
make test-coverage-func

# View coverage by package
make test-coverage-pkg

# Run integration tests
make test-integration

# Run with race detector
make test-race

# Run benchmarks
make test-bench
```

### Test Coverage

This project maintains a minimum of 60% test coverage:

```bash
# Generate coverage report
make test-coverage

# View HTML report
open coverage.html
```

### Run Locally

```bash
# Direct run
go run cmd/agent/main.go

# With hot reload
go install github.com/cosmtrek/air@latest
air
```

### Build

```bash
# Build binary
go build -o github-code-agent cmd/agent/main.go

# Using Makefile
make build
```

### Docker

```bash
# Build
docker build -t github-code-agent .

# Run
docker run -p 8001:8001 --env-file .env github-code-agent
```

### Kubernetes

```bash
# Deploy with Helm
helm install agentfield-control-plane ./helm/agentfield \
  -n agentfield \
  --create-namespace

# Check status
kubectl get pods -n agentfield
```

## Performance

### Optimizations

| Optimization        | Improvement     | Description                    |
| ------------------- | --------------- | ------------------------------ |
| Parallel Execution  | ~10x faster     | Concurrent file analysis       |
| Smart Caching       | 35% cost reduction | AI response caching (15-min TTL) |
| Rate Limit Handling | Automatic       | Backoff and adaptive limiting  |

### Cost Estimation

**Assumptions:** 50 PRs/day, 200 LOC/PR, ~4000 tokens/review

**DeepSeek Pricing (via OpenRouter):**

- Input: $0.14 per 1M tokens
- Output: $0.28 per 1M tokens

**Monthly Costs:**

- AI API (without caching): ~$38/month
- AI API (with 50% cache hits): **~$20-25/month**
- Infrastructure: $50-100/month (2-4 pods)
- **Total: ~$80-135/month**

## Deployment

### Docker

```bash
docker build -t github-code-agent .
docker run -d -p 8001:8001 --env-file .env --name code-agent github-code-agent
```

### Kubernetes (Production)

Production-ready Helm chart with:

- Control plane: 1-2 replicas
- Agent workers: 3-10 replicas with HPA
- Health and readiness probes
- ConfigMap for configuration
- External secrets management

```bash
# Create secrets
kubectl create secret generic agentfield-secrets \
  --from-literal=GITHUB_APP_ID=123456 \
  --from-literal=GITHUB_PRIVATE_KEY="-----BEGIN..." \
  --from-literal=OPENAI_API_KEY=sk-your-key \
  -n agentfield

# Deploy
helm install agentfield-control-plane ./helm/agentfield \
  --namespace agentfield \
  -f ./helm/agentfield/values-production.yaml
```

See [Deployment Guide](docs/DEPLOYMENT.md) for complete instructions.

## Documentation

| Document | Description |
| -------- | ----------- |
| [Quick Start](docs/QUICK_START.md) | Get started in 5-20 minutes |
| [User Guide](docs/USER_GUIDE.md) | How to use the agent effectively |
| [GitHub App Setup](docs/GITHUB_APP_SETUP.md) | Configure GitHub App step-by-step |
| [Configuration Reference](docs/CONFIGURATION_REFERENCE.md) | All configuration options |
| [Deployment Guide](docs/DEPLOYMENT.md) | Production deployment with Kubernetes |
| [API Reference](docs/API_REFERENCE.md) | Technical API documentation |
| [Architecture Decisions](docs/adr/) | ADRs for major design decisions |
| [Reliability Features](docs/reliability-features.md) | Circuit breaker, retry, health checks |

## Troubleshooting

### Agent Not Responding

1. Check webhook delivery: GitHub App → Advanced → Recent Deliveries
2. Verify webhook URL is publicly accessible
3. Check agent logs: `docker logs github-code-agent` or `kubectl logs -f deployment/agentfield-control-plane -n agentfield`

### Permission Errors

Verify GitHub App permissions:
- Contents: Read & Write
- Pull requests: Read & Write
- Checks: Read

### High API Costs

- Enable caching (default: 15-min TTL)
- Use `wait_for_ci: true` to avoid reviewing failing code
- Exclude files: `ignore_paths: ["*.test.js", "vendor/**"]`

See [User Guide - Troubleshooting](docs/USER_GUIDE.md#troubleshooting) for more.

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `go test ./...`
5. Submit a pull request

## License

This project is licensed under the **Creative Commons Attribution-NonCommercial 4.0 International License (CC BY-NC 4.0)**.

### Commercial Licensing

**Self-Deployment: FREE**

Use freely for any purpose including commercial:
- ✅ Internal company tools
- ✅ Your own code reviews
- ✅ Your infrastructure

**Hosted/SaaS: 10% Revenue Sharing**

If you offer this as a service:
- 10% of gross revenue from the hosted service
- Includes support and updates

See [LICENSE](LICENSE) for full details.

### Attribution

When using or modifying this work:

```text
Based on af-code-agent by Yong Cheng Low
(https://github.com/yongchenglow/af-code-agent), licensed under CC BY-NC 4.0
```

## Support

- **Issues**: https://github.com/yongchenglow/af-code-agent/issues
- **Discussions**: https://github.com/yongchenglow/af-code-agent/discussions
- **AgentField**: https://www.agentfield.ai

## Sponsorship

If you find this project useful, consider sponsoring:

- [GitHub Sponsors](https://github.com/sponsors/yongchenglow)
- [DBS PayLah!](https://www.dbs.com.sg/personal/mobile/paylink/index.html?tranRef=zp6RIxPiw5)

---

**Last Updated:** 2026-02-19
**Version:** 1.0.0
**Go Version:** 1.24
**Status:** ✅ Production Ready
