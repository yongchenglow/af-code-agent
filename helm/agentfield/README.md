# AgentField Agent Helm Chart

This Helm chart deploys AgentField agents for the GitHub Code Agent system.

## Architecture

```
┌─────────────────────────────────────────────┐
│      AgentField Control Plane (External)    │
│  - Must be deployed separately              │
│  - Manages agent coordination               │
│  - Provides task distribution               │
└─────────────────────────────────────────────┘
                    ↓
         AgentField Communication
                    ↓
┌─────────────────────────────────────────────┐
│       Agent Workers (1+ replicas)           │
│  - Process code review tasks                │
│  - Execute AI-powered analysis              │
│  - Apply fixes to pull requests             │
│  - Can auto-scale based on CPU/memory       │
└─────────────────────────────────────────────┘
```

## Components

### Agent Workers

- **Purpose**: Execute code review, analysis, and fix generation tasks
- **Replicas**: 1+ (can autoscale if enabled)
- **Resources**: 250m CPU / 512Mi memory (requests), 1 CPU / 2Gi memory (limits)
- **No external service**: Communicates with control plane via AgentField SDK

## Prerequisites

1. **AgentField Control Plane**: Must be deployed and accessible
2. **Kubernetes cluster**: v1.19+
3. **Helm**: 3.13.0+
4. **GHCR credentials**: For pulling images (if using private registry)
5. **Secrets**: Create `agentfield-secrets` with required configuration

## Installation

### 1. Create GHCR Image Pull Secret

```bash
kubectl create namespace agentfield

kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=<github-username> \
  --docker-password=<github-personal-access-token> \
  --namespace=agentfield
```

### 2. Create AgentField Secrets

Create a secret with the required environment variables:

```bash
kubectl create secret generic agentfield-secrets \
  --from-literal=AGENTFIELD_URL="http://agentfield-control-plane:8080" \
  --from-literal=GITHUB_TOKEN="ghp_xxxxxxxxxxxx" \
  --from-literal=GITHUB_WEBHOOK_SECRET="your-webhook-secret" \
  --from-literal=AI_API_KEY="your-openai-api-key" \
  --from-literal=AI_BASE_URL="https://api.deepseek.com" \
  --from-literal=AI_MODEL="deepseek-chat" \
  --from-literal=OPENROUTER_API_KEY="sk-or-v1-xxxx" \
  --from-literal=LOG_LEVEL="info" \
  --from-literal=PORT="8080" \
  --namespace=agentfield
```

**Required fields:**
- `AGENTFIELD_URL`: URL of the AgentField control plane (must be accessible from agents)
- `GITHUB_TOKEN`: GitHub personal access token with repo access
- `GITHUB_WEBHOOK_SECRET`: Secret for validating GitHub webhooks
- `AI_API_KEY` or `OPENROUTER_API_KEY`: API key for AI model provider
- `AI_BASE_URL`: Base URL for AI API
- `AI_MODEL`: AI model to use (e.g., "deepseek-chat")

See [SECRETS.md](SECRETS.md) for detailed secret configuration.

### 3. Install the Chart

```bash
# Development
helm install agentfield-agent ./helm/agentfield \
  --namespace agentfield \
  --create-namespace

# Production (with specific image tag)
helm install agentfield-agent ./helm/agentfield \
  --namespace agentfield \
  --create-namespace \
  --set agent.image.tag="main-abc123"
```

### 4. Verify Installation

```bash
# Check pods
kubectl get pods -n agentfield

# Check logs
kubectl logs -f -l app.kubernetes.io/component=agent -n agentfield

# Check secrets
kubectl get secret agentfield-secrets -n agentfield -o yaml
```

## Configuration

### Key Values

| Parameter | Description | Default |
|-----------|-------------|---------|
| `agent.enabled` | Enable agent deployment | `true` |
| `agent.replicaCount` | Number of agent replicas | `1` |
| `agent.image.repository` | Agent image repository | `ghcr.io/yongchenglow/af-code-agent` |
| `agent.image.tag` | Agent image tag | Chart.AppVersion |
| `agent.image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `agent.resources.requests.cpu` | CPU request | `250m` |
| `agent.resources.requests.memory` | Memory request | `512Mi` |
| `agent.resources.limits.cpu` | CPU limit | `1` |
| `agent.resources.limits.memory` | Memory limit | `2Gi` |
| `agent.autoscaling.enabled` | Enable HPA | `false` |
| `agent.autoscaling.minReplicas` | Minimum replicas | `1` |
| `agent.autoscaling.maxReplicas` | Maximum replicas | `10` |
| `agent.externalSecret.enabled` | Use external secret | `true` |
| `agent.externalSecret.name` | External secret name | `agentfield-secrets` |

### Example: Custom Values

Create `values-custom.yaml`:

```yaml
agent:
  replicaCount: 3
  image:
    tag: "main-abc123"
  resources:
    requests:
      cpu: 500m
      memory: 1Gi
    limits:
      cpu: 2
      memory: 4Gi
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70
```

Install with custom values:

```bash
helm install agentfield-agent ./helm/agentfield \
  --namespace agentfield \
  --values values-custom.yaml
```

## Upgrading

```bash
# Upgrade with new image tag
helm upgrade agentfield-agent ./helm/agentfield \
  --namespace agentfield \
  --set agent.image.tag="main-xyz789"

# Upgrade with custom values file
helm upgrade agentfield-agent ./helm/agentfield \
  --namespace agentfield \
  --values values-custom.yaml
```

## Uninstalling

```bash
helm uninstall agentfield-agent --namespace agentfield
```

## Troubleshooting

### Agents Not Connecting to Control Plane

Check the `AGENTFIELD_URL` in the secret:

```bash
kubectl get secret agentfield-secrets -n agentfield -o jsonpath='{.data.AGENTFIELD_URL}' | base64 -d
```

Verify control plane is accessible:

```bash
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://agentfield-control-plane:8080/health
```

### Pods Crashing

Check logs:

```bash
kubectl logs -l app.kubernetes.io/component=agent -n agentfield --tail=100
```

Common issues:
- Missing or invalid `AGENTFIELD_URL`
- Control plane not running
- Missing AI API keys
- Invalid GitHub token

### Image Pull Errors

Verify the GHCR secret:

```bash
kubectl get secret ghcr-secret -n agentfield
```

Test image pull:

```bash
kubectl run test-pull --image=ghcr.io/yongchenglow/af-code-agent:main --restart=Never -n agentfield
kubectl delete pod test-pull -n agentfield
```

## Support

For issues and questions, please create an issue in the repository.
