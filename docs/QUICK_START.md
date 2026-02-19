# Quick Start Guide

Get the GitHub Code Review Agent running in minutes.

## Prerequisites

- [ ] Go 1.24 or higher
- [ ] GitHub account with admin access
- [ ] DeepSeek API key (or OpenAI-compatible API)
- [ ] Kubernetes cluster (for production deployment)

## Deployment Options Overview

```mermaid
flowchart LR
    subgraph Options["Deployment Options"]
        direction TB
        A[Local Development<br/>5 minutes<br/>Best for: Testing & Dev]
        B[Docker<br/>10 minutes<br/>Best for: Staging]
        C[Kubernetes<br/>20 minutes<br/>Best for: Production]
    end
    
    subgraph Features["Common Features"]
        F1[GitHub Webhooks]
        F2[AI Code Review]
        F3[Auto-Fixes]
        F4[Standards Validation]
    end
    
    A --> Features
    B --> Features
    C --> Features
    
    style A fill:#d4edda
    style B fill:#fff3cd
    style C fill:#f8d7da
    style Features fill:#e1f5ff
```

---

## Option 1: Local Development (5 minutes)

### Step 1: Clone and Build

```bash
git clone https://github.com/yongchenglow/af-code-agent.git
cd af-code-agent
go build -o github-code-agent ./cmd/agent
```

### Step 2: Configure Environment

```bash
cp .env.example .env
```

Edit `.env` with your credentials:

```bash
# GitHub App Configuration
GITHUB_APP_ID=your-app-id
GITHUB_PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----"
GITHUB_WEBHOOK_SECRET=your-webhook-secret

# AI Configuration
OPENAI_API_KEY=your-deepseek-api-key
AI_BASE_URL=https://api.deepseek.com
AI_MODEL=deepseek-chat

# Application Settings
PORT=8001
LOG_LEVEL=info
```

### Step 3: Set Up GitHub App

Follow the [GitHub App Setup Guide](GITHUB_APP_SETUP.md) to:

1. Create a GitHub App
2. Generate private key
3. Configure webhook URL (use ngrok for local testing)
4. Install the app on your repository

### Step 4: Run the Agent

```bash
./github-code-agent
```

Or with hot reload:

```bash
go install github.com/cosmtrek/air@latest
air
```

### Step 5: Test with ngrok

```bash
# In a new terminal
ngrok http 8001

# Copy the HTTPS URL and update your GitHub App webhook URL
```

### Step 6: Create a Test PR

1. Make code changes in a branch
2. Push: `git push origin feature-branch`
3. Open a PR on GitHub
4. Watch the agent review your code!

---

## Option 2: Docker (10 minutes)

### Step 1: Build Docker Image

```bash
docker build -t github-code-agent .
```

### Step 2: Run Container

```bash
docker run -d -p 8001:8001 --env-file .env --name code-agent github-code-agent
```

### Step 3: Expose with ngrok

```bash
ngrok http 8001
```

Update GitHub App webhook URL with the ngrok HTTPS URL.

---

## Option 3: Kubernetes Production (20 minutes)

### Step 1: Configure GitHub Secrets

In your GitHub repository **Settings → Secrets and variables → Actions**, add:

```bash
# Kubeconfig (base64 encoded)
cat ~/.kube/config | base64 | pbcopy
# Add as: KUBECONFIG_PROD

# Cloudflare credentials
CLOUDFLARE_API_TOKEN=your-token
CLOUDFLARE_ACCOUNT_ID=your-account-id
CLOUDFLARE_TUNNEL_ID=your-tunnel-id
```

### Step 2: Create Kubernetes Secrets

```bash
# Create namespace
kubectl create namespace agentfield

# Create secrets
kubectl create secret generic agentfield-secrets \
  --from-literal=GITHUB_APP_ID=123456 \
  --from-literal=GITHUB_PRIVATE_KEY="-----BEGIN..." \
  --from-literal=GITHUB_WEBHOOK_SECRET=your-secret \
  --from-literal=OPENAI_API_KEY=sk-your-key \
  -n agentfield

# Create GHCR pull secret
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=YOUR_GITHUB_USERNAME \
  --docker-password=YOUR_GITHUB_PAT \
  --docker-email=YOUR_EMAIL \
  -n agentfield
```

### Step 3: Deploy with GitHub Actions

Push to `main` branch:

```bash
git add .
git commit -m "Deploy code review agent"
git push origin main
```

GitHub Actions will automatically:

1. Build and test
2. Create Docker image
3. Push to GHCR
4. Deploy to Kubernetes
5. Configure Cloudflare

### Step 4: Verify Deployment

```bash
# Check pods
kubectl get pods -n agentfield

# View logs
kubectl logs -f deployment/agentfield-control-plane -n agentfield

# Test endpoint
curl https://agentfield.yongchenglow.com/health
```

---

## Next Steps

### Configure Your Repository

Create `.github/code-agent.yml` in your repository:

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
```

See [Configuration Reference](CONFIGURATION_REFERENCE.md) for all options.

### Learn More

- **[User Guide](USER_GUIDE.md)** - How to use the agent effectively
- **[GitHub App Setup](GITHUB_APP_SETUP.md)** - Detailed GitHub App configuration
- **[Configuration Reference](CONFIGURATION_REFERENCE.md)** - All configuration options
- **[Deployment Guide](DEPLOYMENT.md)** - Production deployment details
- **[API Reference](API_REFERENCE.md)** - Technical API documentation

### Troubleshooting

**Agent not responding?**

- Check webhook delivery: GitHub App → Advanced → Recent Deliveries
- Verify webhook URL is publicly accessible
- Check agent logs for errors

**Permission errors?**

- Verify GitHub App has correct permissions:
  - Contents: Read & Write
  - Pull requests: Read & Write
  - Checks: Read

**High API costs?**

- Enable caching (enabled by default)
- Use `wait_for_ci: true` to avoid reviewing failing code
- Exclude test files: `ignore_paths: ["*.test.js"]`

For more help, see [Troubleshooting](USER_GUIDE.md#troubleshooting) in the User Guide.

---

## Quick Reference

### Ports

- **Local/Docker**: 8001
- **Kubernetes NodePort**: 30007

### Key Files

- `.env` - Environment configuration
- `.github/code-agent.yml` - Repository configuration
- `helm/agentfield/` - Kubernetes Helm chart

### Common Commands

```bash
# Run tests
go test ./...

# Build
go build -o github-code-agent ./cmd/agent

# Run locally
go run ./cmd/agent

# Docker
docker build -t github-code-agent .
docker run -p 8001:8001 --env-file .env github-code-agent

# Kubernetes
kubectl get pods -n agentfield
kubectl logs -f deployment/agentfield-control-plane -n agentfield
helm upgrade --install agentfield-control-plane ./helm/agentfield -n agentfield
```

### URLs

- **Local**: http://localhost:8001
- **Production**: https://agentfield.yongchenglow.com
