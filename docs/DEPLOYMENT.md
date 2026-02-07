# Deployment Guide

This guide explains how to deploy the GitHub Code Agent using GitHub Actions and Helm to Kubernetes.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [GitHub Secrets Configuration](#github-secrets-configuration)
- [Workflows](#workflows)
- [Helm Charts](#helm-charts)
- [Deployment Process](#deployment-process)
- [Troubleshooting](#troubleshooting)

## Overview

The deployment infrastructure consists of:

- **Namespace**: `agentfield`
- **Control Plane**: Deployed as `agentfield-control-plane` (1-2 replicas)
- **Agent Workers**: Deployed as `agentfield-agent` (3-10 replicas, autoscaling)
- **Production URL**: <https://agentfield.yongchenglow.com>

### Architecture

The system uses a distributed architecture powered by AgentField:

```
┌────────────────────────────────────┐
│    Control Plane (1-2 replicas)    │
│  - Webhook handling                 │
│  - Task orchestration               │
│  - HTTP API (port 8080)             │
└────────────────────────────────────┘
              ↓ AgentField
┌────────────────────────────────────┐
│   Agent Workers (3-10 replicas)    │
│  - Code review execution            │
│  - AI-powered analysis              │
│  - Fix generation                   │
│  - Auto-scales on CPU/memory        │
└────────────────────────────────────┘
```

## Prerequisites

1. Kubernetes cluster with kubectl access
2. Helm 3.13.0 or later
3. GitHub Container Registry (GHCR) access
4. Cloudflare account with:
   - Tunnel configured
   - DNS zone for yongchenglow.com
   - API token with DNS and Tunnel edit permissions

## GitHub Secrets Configuration

Configure the following secrets in your GitHub repository settings:

### Required Secrets

#### Production Deployment

- `KUBECONFIG_PROD`: Base64-encoded kubeconfig file for production cluster

  ```bash
  cat ~/.kube/config | base64 | pbcopy
  ```

- `CLOUDFLARE_API_TOKEN`: Cloudflare API token with permissions:
  - Zone:DNS:Edit
  - Account:Cloudflare Tunnel:Edit

- `CLOUDFLARE_ACCOUNT_ID`: Your Cloudflare account ID

- `CLOUDFLARE_TUNNEL_ID`: Your Cloudflare tunnel ID

### Container Registry

The workflows use `GITHUB_TOKEN` which is automatically provided by GitHub Actions.

### Creating GHCR Secret in Kubernetes

Before deploying, create a secret for pulling images from GitHub Container Registry:

```bash
# Create the secret in the default namespace first
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=YOUR_GITHUB_USERNAME \
  --docker-password=YOUR_GITHUB_PAT \
  --docker-email=YOUR_EMAIL \
  -n default

# The workflow will copy it to the agentfield namespace
```

## Workflows

### CI Workflow (`.github/workflows/ci.yml`)

Triggered on:

- Push to `main` or `develop` branches
- Pull requests to `main` or `develop` branches

Workflow steps:

1. **Build & Test**: Runs Go tests and builds the application
2. **Docker Build & Push**: Creates multi-arch Docker image and pushes to GHCR
3. **Deploy to Production**: Deploys to production when pushing to `main` branch

### Reusable Workflows

#### `reusable-build.yml`

- Sets up Go environment
- Caches dependencies
- Runs tests and linting
- Builds the application

#### `reusable-docker.yml`

- Builds multi-architecture Docker images (amd64, arm64)
- Pushes to GitHub Container Registry
- Uses Docker layer caching for faster builds

#### `production-deploy.yml`

- Deploys using Helm
- Configures Cloudflare Tunnel routes
- Creates/updates DNS records
- Verifies deployment health

## Helm Charts

### Chart Structure

```
helm/agentfield/
├── Chart.yaml                    # Chart metadata
├── values.yaml                   # Default values
├── values-production.yaml        # Production overrides
├── README.md                     # Chart documentation
└── templates/
    ├── _helpers.tpl             # Template helpers
    ├── deployment.yaml          # Control plane deployment
    ├── agent-deployment.yaml    # Agent worker deployment
    ├── service.yaml             # Control plane service
    ├── serviceaccount.yaml      # ServiceAccount manifest
    ├── configmap.yaml           # ConfigMap manifest
    ├── hpa.yaml                 # Control plane HPA
    ├── agent-hpa.yaml           # Agent worker HPA
    └── NOTES.txt               # Post-install notes
```

### Values Configuration

#### Production (`values-production.yaml`)

**Control Plane:**

- 1 replica (with autoscaling up to 2)
- 500m CPU / 512Mi memory (requests)
- 1 CPU / 1Gi memory (limits)
- NodePort: 30007
- Host: agentfield.yongchenglow.com

**Agent Workers:**

- 3 replicas (with autoscaling up to 10)
- 1 CPU / 1Gi memory (requests)
- 2 CPU / 2Gi memory (limits)
- Auto-scales at 70% CPU / 75% memory

### Environment Variables

Configure application settings via the `env` section in values files:

```yaml
controlPlane:
  env:
    - name: AGENTFIELD_URL
      value: "http://agentfield-control-plane:8080"
    - name: PORT
      value: "8080"
    - name: ENVIRONMENT
      value: "production"
```

### Secrets Management

**CRITICAL**: Create the `agentfield-secrets` secret BEFORE deploying the Helm chart.

#### Create from .env file (Recommended)

```bash
# Navigate to the helm chart directory
cd helm/agentfield

# Create secret from your .env file
kubectl create secret generic agentfield-secrets \
  --from-env-file=../../.env \
  -n agentfield
```

#### Create from secret template

```bash
# Copy and edit the template
cp helm/agentfield/secret-example.yaml helm/agentfield/secret.yaml
vim helm/agentfield/secret.yaml

# Apply the secret
kubectl apply -f helm/agentfield/secret.yaml

# Clean up (important!)
rm helm/agentfield/secret.yaml
```

#### Required Secret Keys

The secret must contain these environment variables from `.env.example`:
- `AGENTFIELD_URL`
- `GITHUB_TOKEN`
- `GITHUB_WEBHOOK_SECRET`
- `AI_API_KEY`
- `AI_BASE_URL`
- `AI_MODEL`
- `LOG_LEVEL`
- `PORT`

See `helm/agentfield/secret-example.yaml` for the complete template.

## Deployment Process

### Automatic Deployment

**Push to main branch** → triggers production deployment

```bash
git checkout main
git add .
git commit -m "Your changes"
git push origin main
```

### Manual Deployment

#### Using Helm directly

**Production:**

```bash
export IMAGE_TAG="main-abc123def456"

helm upgrade --install agentfield-control-plane ./helm/agentfield \
  -n agentfield \
  --create-namespace \
  -f ./helm/agentfield/values-production.yaml \
  --set controlPlane.image.tag=$IMAGE_TAG \
  --wait
```

### Verify Deployment

```bash
# Check all pods
kubectl get pods -n agentfield

# Check control plane pods
kubectl get pods -l app.kubernetes.io/component=control-plane -n agentfield

# Check agent worker pods
kubectl get pods -l app.kubernetes.io/component=agent -n agentfield

# Check services
kubectl get svc -n agentfield

# View control plane logs
kubectl logs -f deployment/agentfield-control-plane -n agentfield

# View agent worker logs
kubectl logs -f deployment/agentfield-agent -n agentfield

# Check deployment status
kubectl rollout status deployment/agentfield-control-plane -n agentfield
kubectl rollout status deployment/agentfield-agent -n agentfield

# Check autoscaling status
kubectl get hpa -n agentfield
```

## Deployment Architecture

```
┌─────────────────────────────────────────────────────┐
│                  GitHub Actions                      │
│  ┌──────────┐  ┌──────────┐  ┌────────────────┐   │
│  │  Build   │→ │  Docker  │→ │   Deploy       │   │
│  │  & Test  │  │  Build   │  │  (Production)  │   │
│  └──────────┘  └──────────┘  └────────────────┘   │
└─────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────┐
│           GitHub Container Registry (GHCR)          │
│              ghcr.io/yourorg/af-code-agent          │
└─────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────┐
│              Kubernetes Cluster                      │
│  Namespace: agentfield                              │
│  ┌───────────────────────────────────────────┐     │
│  │  Control Plane Deployment                 │     │
│  │  - Name: agentfield-control-plane         │     │
│  │  - Replicas: 1-2 (autoscaling)            │     │
│  │  - Port: 8080                             │     │
│  │  - Resources: 500m CPU / 512Mi RAM        │     │
│  └───────────────────────────────────────────┘     │
│  ┌───────────────────────────────────────────┐     │
│  │  Agent Workers Deployment                 │     │
│  │  - Name: agentfield-agent                 │     │
│  │  - Replicas: 3-10 (autoscaling)           │     │
│  │  - Resources: 1 CPU / 1Gi RAM             │     │
│  │  - Connects to Control Plane via AgentField     │
│  └───────────────────────────────────────────┘     │
│  ┌───────────────────────────────────────────┐     │
│  │  Service: NodePort 30007                  │     │
│  │  - Exposes Control Plane only             │     │
│  └───────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────┐
│              Cloudflare Tunnel                       │
│  agentfield.yongchenglow.com → localhost:30007      │
└─────────────────────────────────────────────────────┘
```

## Troubleshooting

### Deployment Fails

1. **Check workflow logs** in GitHub Actions tab
2. **Verify secrets** are correctly configured
3. **Check Kubernetes cluster** access:

   ```bash
   kubectl get nodes
   kubectl get ns
   ```

### Image Pull Errors

If pods fail to pull images:

```bash
# Verify secret exists
kubectl get secret ghcr-secret -n agentfield

# If not, create it
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=YOUR_GITHUB_USERNAME \
  --docker-password=YOUR_GITHUB_PAT \
  -n agentfield
```

### Pod Crashes or CrashLoopBackOff

```bash
# Check pod logs
kubectl logs -f deployment/agentfield-control-plane -n agentfield

# Describe pod for events
kubectl describe pod -l app.kubernetes.io/name=agentfield -n agentfield

# Check resource limits
kubectl top pods -n agentfield
```

### DNS/Cloudflare Issues

1. Verify Cloudflare API token has correct permissions
2. Check tunnel is running:

   ```bash
   cloudflared tunnel info YOUR_TUNNEL_ID
   ```

3. Verify DNS records in Cloudflare dashboard

### Health Check Failures

The deployment includes health checks on `/health` endpoint:

```bash
# Test locally (if port-forwarded)
kubectl port-forward svc/agentfield-control-plane 8080:8080 -n agentfield
curl http://localhost:8080/health
```

### Rollback Deployment

If a deployment fails, rollback to previous version:

```bash
# View rollout history
kubectl rollout history deployment/agentfield-control-plane -n agentfield

# Rollback to previous version
kubectl rollout undo deployment/agentfield-control-plane -n agentfield

# Rollback to specific revision
kubectl rollout undo deployment/agentfield-control-plane --to-revision=2 -n agentfield
```

## Monitoring

### View Application Logs

```bash
# Follow logs
kubectl logs -f deployment/agentfield-control-plane -n agentfield

# Last 100 lines
kubectl logs --tail=100 deployment/agentfield-control-plane -n agentfield

# Logs from previous pod (if crashed)
kubectl logs --previous deployment/agentfield-control-plane -n agentfield
```

### Resource Usage

```bash
# Check pod resource usage
kubectl top pods -n agentfield

# Check node resource usage
kubectl top nodes
```

### Horizontal Pod Autoscaling (Production)

```bash
# Check HPA status
kubectl get hpa -n agentfield

# Describe HPA
kubectl describe hpa agentfield-control-plane -n agentfield
```

## Security Best Practices

1. **Never commit secrets** to the repository
2. **Use Kubernetes secrets** for sensitive data
3. **Enable RBAC** in your cluster
4. **Use least privilege** service accounts
5. **Regularly update** dependencies and base images
6. **Enable network policies** for pod-to-pod communication
7. **Use sealed secrets** or external secret managers for production

## Support

For issues or questions:

1. Check the [troubleshooting section](#troubleshooting)
2. Review workflow logs in GitHub Actions
3. Check Kubernetes pod logs and events
4. Consult the main [README.md](../README.md)
