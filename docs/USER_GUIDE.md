# GitHub Code Review Agent - User Guide

## Table of Contents

1. [Introduction](#introduction)
2. [Installation](#installation)
3. [GitHub App Setup](#github-app-setup)
4. [Configuration](#configuration)
5. [First PR Review](#first-pr-review)
6. [Operating Modes](#operating-modes)
7. [Language Support](#language-support)
8. [Troubleshooting](#troubleshooting)
9. [Best Practices](#best-practices)
10. [FAQ](#faq)

## Introduction

The GitHub Code Review Agent is an autonomous AI-powered code reviewer that automatically:

- Reviews pull requests when they're opened or updated
- Identifies bugs, security vulnerabilities, and code smells
- Validates code against configurable standards
- Generates and applies fixes automatically
- Posts detailed review comments with severity levels

**Key Features:**

- 🤖 **Automated Reviews**: AI-powered code analysis using DeepSeek
- 🔒 **Security Scanning**: OWASP Top 10 vulnerability detection
- 📝 **Standards Validation**: Enforces coding standards and best practices
- 🔧 **Auto-Fix**: Generates and applies fixes with validation loop
- 🎯 **Smart Prioritization**: Issues categorized by severity (Critical, High, Medium, Low)
- 🔄 **Two Modes**: YOLO (direct push) and Safe (PR with fixes)

## Installation

### Prerequisites

- Go 1.21 or higher
- GitHub organization or repository with admin access
- DeepSeek API key (or OpenAI-compatible API)
- AgentField deployed (optional, for distributed deployment)

### Quick Start

1. **Clone the repository:**

```bash
git clone https://github.com/yourorg/github-code-agent.git
cd github-code-agent
```

2. **Install dependencies:**

```bash
go mod download
```

3. **Build the agent:**

```bash
go build -o github-code-agent ./cmd/agent
```

4. **Create environment configuration:**

```bash
cp .env.example .env
# Edit .env with your credentials
```

## GitHub App Setup

### Step 1: Create a GitHub App

1. Go to GitHub Settings → Developer settings → GitHub Apps → New GitHub App
2. Fill in the required information:
   - **GitHub App name**: `Your Org Code Review Agent`
   - **Homepage URL**: Your organization URL
   - **Webhook URL**: `https://your-domain.com/webhook` (where agent is deployed)
   - **Webhook secret**: Generate a strong secret (save for later)

3. **Configure permissions:**

   Repository permissions:
   - **Contents**: Read & Write (to push fixes)
   - **Pull requests**: Read & Write (to comment and create PRs)
   - **Checks**: Read (to access CI/CD status)
   - **Metadata**: Read

4. **Subscribe to events:**
   - Pull request (opened, synchronize, reopened)
   - Check suite (completed)
   - Workflow run (completed)

5. **Create the app** and note the **App ID**

### Step 2: Generate Private Key

1. Scroll to "Private keys" section
2. Click "Generate a private key"
3. Save the downloaded `.pem` file securely

### Step 3: Install the App

1. Go to "Install App" tab
2. Install on your organization or specific repositories
3. Note the **Installation ID** from the URL (e.g., `https://github.com/settings/installations/12345`)

### Step 4: Configure Environment Variables

Update your `.env` file:

```bash
# GitHub Configuration
GITHUB_APP_ID=123456
GITHUB_PRIVATE_KEY_PATH=./path/to/private-key.pem
GITHUB_WEBHOOK_SECRET=your-webhook-secret

# AI Configuration (DeepSeek via OpenRouter)
OPENROUTER_API_KEY=sk-or-v1-your-api-key
AI_MODEL=deepseek/deepseek-chat
AI_TEMPERATURE=0.2
AI_MAX_TOKENS=4000

# AgentField (optional)
AGENTFIELD_URL=http://localhost:8080

# Application
LOG_LEVEL=info
PORT=8080
```

## Configuration

Create `.github/code-agent.yml` in your repository:

```yaml
# Agent configuration
agent:
  enabled: true
  mode: safe  # "safe" or "yolo"

# Webhook triggers
webhooks:
  triggers:
    - pull_request.opened
    - pull_request.synchronize
    - check_suite.completed
  wait_for_ci: true
  debounce_seconds: 30

# Review settings
review:
  auto_review: true
  auto_fix: true
  severity_threshold: medium  # Only auto-fix medium and below
  ignore_paths:
    - "*.md"
    - "docs/**"
    - "tests/fixtures/**"

# Validation settings
validation:
  enabled: true
  max_fix_attempts: 3
  checks:
    - syntax
    - linting
    - formatting
    - security
  timeout_seconds: 30

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

  go:
    linters:
      - golangci-lint
    min_test_coverage: 75
```

## First PR Review

### Step 1: Create a Test PR

Create a simple PR with a few code changes:

```go
// Example: main.go
package main

import "fmt"

func main() {
    user := getUser()
    fmt.Println(user.Name) // Potential nil pointer!
}

func getUser() *User {
    // Missing nil check
    return nil
}

type User struct {
    Name string
}
```

### Step 2: Open the PR

1. Push your branch: `git push origin feature-branch`
2. Open a PR on GitHub
3. The agent will automatically trigger

### Step 3: Watch the Agent Work

You'll see the agent:

1. **Post initial comment** acknowledging the review
2. **Analyze the PR** (file changes, code structure, metrics)
3. **Run validations** (standards, security, quality)
4. **Post review comments** with issues found:

   ```
   🔴 Critical: Potential nil pointer dereference

   Details:
   The function `main` doesn't check if `user` is nil before accessing
   the `Name` field, which will cause a panic.

   Suggested fix:
   Add nil check: if user != nil { ... }

   Automated review by GitHub Code Agent
   ```

5. **Generate fixes** (if auto-fix enabled)
6. **Create fix PR** (Safe mode) or push directly (YOLO mode)
7. **Update comments** with links to fixes

### Step 4: Review the Results

**Safe Mode:**
- A new PR will be created (e.g., `🤖 Automated fixes for PR #123`)
- Review the fixes and merge if acceptable
- Original PR comments will link to the fix PR

**YOLO Mode:**
- Fixes are pushed directly to your PR branch
- PR comments will link to the fix commit
- CI/CD runs automatically to verify fixes

## Operating Modes

### Safe Mode (Recommended)

**What it does:**
- Creates a new branch with fixes
- Opens a PR targeting your original PR branch
- Allows human review before merging
- Safer for production repositories

**When to use:**
- Production repositories
- Teams new to the agent
- Complex changes requiring review
- High-stakes code

**Configuration:**

```yaml
agent:
  mode: safe
```

**Workflow:**

```
PR opened
  ↓
Review code → Find issues
  ↓
Post review comments
  ↓
Generate fixes → Validate
  ↓
Create fix branch: agent-fixes/pr-123
  ↓
Create PR #124 → targets feature-branch
  ↓
Human reviews PR #124
  ↓
Merge → Fixes applied to original PR
```

### YOLO Mode

**What it does:**
- Pushes fixes directly to your PR branch
- No intermediate PR created
- Faster, but more aggressive

**When to use:**
- Personal repositories
- Trusted, well-tested code
- Simple, low-risk fixes
- Development/staging environments

**Configuration:**

```yaml
agent:
  mode: yolo
  auto_fix:
    require_tests_passing: true  # Safety check
```

**Workflow:**

```
PR opened
  ↓
Review code → Find issues
  ↓
Post review comments
  ↓
Generate fixes → Validate
  ↓
Push directly to feature-branch
  ↓
CI/CD runs
  ↓
Update comments with commit link
```

### Controlling Modes Per-PR

Override the default mode using PR labels:

- `agent:safe` - Force safe mode
- `agent:yolo` - Force YOLO mode
- `agent:skip` - Skip automated review
- `agent:review-only` - Review but don't auto-fix

## Language Support

### Supported Languages

The agent supports the following languages with full analysis:

#### Go
- **Linters**: golangci-lint
- **Formatters**: gofmt
- **Metrics**: cyclomatic complexity, LOC
- **Security**: hardcoded secrets, SQL injection, unsafe operations

#### Python
- **Linters**: pylint, black, mypy, flake8
- **Formatters**: black, autopep8
- **Metrics**: complexity, test coverage
- **Security**: injection attacks, secrets, insecure dependencies

#### JavaScript/TypeScript
- **Linters**: eslint, tslint
- **Formatters**: prettier
- **Frameworks**: React, Vue, Angular detection
- **Security**: XSS, prototype pollution, dependency vulnerabilities

#### Other Languages (Basic Support)
- Java
- Ruby
- PHP
- C/C++
- Rust

### Language-Specific Configuration

```yaml
languages:
  go:
    linters:
      - golangci-lint
    min_test_coverage: 75
    complexity_threshold: 10

  python:
    linters:
      - pylint
      - black
      - mypy
    min_test_coverage: 80
    docstring_style: google

  javascript:
    linters:
      - eslint
      - prettier
    frameworks:
      - react
    min_test_coverage: 70
```

## Troubleshooting

### Agent Not Responding

**Problem**: Agent doesn't comment on PRs

**Solutions:**

1. **Check webhook delivery:**
   - Go to GitHub App → Advanced → Recent Deliveries
   - Verify webhooks are being sent successfully

2. **Check agent logs:**
   ```bash
   docker logs github-code-agent
   ```

3. **Verify configuration:**
   - Ensure `.github/code-agent.yml` exists
   - Check `agent.enabled: true`

4. **Check webhook URL:**
   - Verify it's publicly accessible
   - Test with `curl https://your-domain.com/webhook`

### Fixes Not Being Applied

**Problem**: Agent reviews but doesn't create fixes

**Solutions:**

1. **Check auto-fix setting:**
   ```yaml
   review:
     auto_fix: true  # Ensure this is enabled
   ```

2. **Check severity threshold:**
   ```yaml
   review:
     severity_threshold: medium  # Only fixes medium and below
   ```

3. **Check validation logs:**
   - Fixes may be failing validation
   - Check logs for syntax/lint errors

4. **Increase max attempts:**
   ```yaml
   validation:
     max_fix_attempts: 5  # Default is 3
   ```

### High API Costs

**Problem**: DeepSeek API costs are higher than expected

**Solutions:**

1. **Enable caching:**
   - Caching is enabled by default (15-min TTL)
   - Check cache hit rate in logs

2. **Reduce review frequency:**
   ```yaml
   webhooks:
     wait_for_ci: true  # Wait for CI before reviewing
     debounce_seconds: 60  # Wait longer for rapid commits
   ```

3. **Limit file scope:**
   ```yaml
   review:
     ignore_paths:
       - "vendor/**"
       - "node_modules/**"
       - "*.test.js"
   ```

4. **Lower temperature:**
   ```bash
   AI_TEMPERATURE=0.1  # More deterministic, uses less tokens
   ```

### Permission Errors

**Problem**: "403 Forbidden" or "Resource not accessible"

**Solutions:**

1. **Check GitHub App permissions:**
   - Contents: Read & Write
   - Pull requests: Read & Write
   - Checks: Read

2. **Reinstall the app:**
   - Go to GitHub Settings → Applications
   - Uninstall and reinstall the agent

3. **Verify private key:**
   - Ensure `.pem` file is correct
   - Check file permissions: `chmod 600 private-key.pem`

## Best Practices

### 1. Start with Safe Mode

Begin with Safe mode to understand how the agent works:

```yaml
agent:
  mode: safe
```

Once comfortable, consider YOLO mode for specific repositories.

### 2. Configure Severity Thresholds

Only auto-fix low-risk issues:

```yaml
review:
  severity_threshold: low  # Only auto-fix Low severity issues
```

Critical/High issues should be reviewed manually.

### 3. Wait for CI/CD

Let tests pass before reviewing:

```yaml
webhooks:
  wait_for_ci: true
  triggers:
    - check_suite.completed
```

This avoids reviewing code that fails tests.

### 4. Use Ignore Paths

Exclude generated files, dependencies:

```yaml
review:
  ignore_paths:
    - "vendor/**"
    - "node_modules/**"
    - "*.pb.go"  # Protocol buffer generated files
    - "docs/**"
```

### 5. Monitor and Adjust

Review agent performance regularly:

- Check fix acceptance rate
- Monitor API costs
- Adjust standards as needed
- Collect team feedback

### 6. Gradual Rollout

1. **Week 1**: Enable on 1-2 test repositories
2. **Week 2**: Review-only mode on more repos
3. **Week 3**: Safe mode with auto-fix
4. **Week 4+**: Consider YOLO mode for select repos

## FAQ

### How much does it cost to run?

**AI Costs (DeepSeek via OpenRouter):**
- ~$20-35/month for 50 PRs/day
- 85-90% cheaper than GPT-4 or Claude

**Infrastructure:**
- $50-100/month for cloud VMs (2-4 Go agent pods)

**Total: ~$80-135/month**

### Is my code sent to third parties?

- PR content is sent to DeepSeek API for analysis
- All data is transmitted over HTTPS
- DeepSeek doesn't store prompts (check their policy)
- Consider self-hosted AI for sensitive code

### Can I use a different AI model?

Yes! The agent supports any OpenAI-compatible API:

```bash
# OpenAI
OPENOPENAI_API_KEY=sk-proj-...
AI_MODEL=gpt-4o

# Anthropic Claude (via OpenRouter)
OPENROUTER_API_KEY=sk-or-v1-...
AI_MODEL=anthropic/claude-opus-4-5

# Self-hosted (e.g., Ollama)
AI_BASE_URL=http://localhost:11434/v1
AI_MODEL=codellama
```

### How do I disable the agent for a specific PR?

Add the `agent:skip` label to the PR.

### Can it review draft PRs?

Yes, configure webhook triggers:

```yaml
webhooks:
  triggers:
    - pull_request.opened
    - pull_request.ready_for_review  # Or wait until draft → ready
```

### What if the agent makes a mistake?

1. **Reject the fix PR** (Safe mode) or **revert the commit** (YOLO mode)
2. **Add feedback** as a PR comment (future enhancement: learning from feedback)
3. **Adjust configuration** to prevent similar issues

### Can it work with monorepos?

Yes! Configure per-directory:

```yaml
# .github/code-agent.yml
review:
  ignore_paths:
    - "service-a/**"  # Exclude specific services

# service-b/.code-agent.yml (override)
standards:
  coding:
    max_line_length: 120  # Different standard for this service
```

### How do I update the agent?

```bash
git pull origin main
go build -o github-code-agent ./cmd/agent
docker restart github-code-agent
```

For major updates, review the CHANGELOG for breaking changes.

---

## Getting Help

- **Issues**: https://github.com/yourorg/github-code-agent/issues
- **Discussions**: https://github.com/yourorg/github-code-agent/discussions
- **Documentation**: https://github.com/yourorg/github-code-agent/tree/main/docs
- **Email**: support@yourorg.com

## Contributing

We welcome contributions! See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](../LICENSE) for details.
