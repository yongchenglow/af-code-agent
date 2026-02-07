# AgentField Helm Chart

This Helm chart deploys the AgentField-based GitHub Code Agent system, which consists of two main components:

## Architecture

```
┌─────────────────────────────────────────────┐
│          Control Plane (1-2 replicas)       │
│  - Receives webhooks from GitHub            │
│  - Manages agent coordination via AgentField│
│  - Exposes HTTP API on port 8080            │
│  - NodePort service for external access     │
└─────────────────────────────────────────────┘
                    ↓
         AgentField Communication
                    ↓
┌─────────────────────────────────────────────┐
│       Agent Workers (3-10 replicas)         │
│  - Process code review tasks                │
│  - Execute AI-powered analysis              │
│  - Apply fixes to pull requests             │
│  - Auto-scales based on CPU/memory          │
└─────────────────────────────────────────────┘
```

## Components

### Control Plane

- **Purpose**: Entry point for GitHub webhooks and task orchestration
- **Replicas**: 1-2 (can autoscale)
- **Resources**: 500m CPU / 512Mi memory (requests), 1 CPU / 1Gi memory (limits)
- **Service**: NodePort on port 30007 (production)
- **Health checks**: Liveness and readiness probes on `/health`

### Agent Workers

- **Purpose**: Execute code review, analysis, and fix generation tasks
- **Replicas**: 3-10 (autoscales based on load)
- **Resources**: 1 CPU / 1Gi memory (requests), 2 CPU / 2Gi memory (limits)
- **No external service**: Communicates with control plane via AgentField

## Installation

### Prerequisites

1. Kubernetes cluster (v1.19+)
2. Helm 3.13.0+
3. GHCR credentials for pulling images
4. GitHub App credentials (stored as Kubernetes secrets)

### Create Image Pull Secret

```bash
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=YOUR_GITHUB_USERNAME \
  --docker-password=YOUR_GITHUB_PAT \
  --docker-email=YOUR_EMAIL \
  -n agentfield
```

### Create Application Secrets

**IMPORTANT**: Create the `agentfield-secrets` secret BEFORE deploying the Helm chart.

#### Option 1: Using the secret template (Recommended)

```bash
# Copy the example
cp secret-example.yaml secret.yaml

# Edit secret.yaml with your actual credentials
vim secret.yaml

# Create the secret
kubectl apply -f secret.yaml

# Clean up (DO NOT commit secret.yaml!)
rm secret.yaml
```

#### Option 2: From .env file

```bash
# If you have a .env file with all required variables
kubectl create secret generic agentfield-secrets \
  --from-env-file=.env \
  -n agentfield
```

#### Option 3: From command line literals

```bash
kubectl create secret generic agentfield-secrets \
  --from-literal=AGENTFIELD_URL="http://agentfield-control-plane:8080" \
  --from-literal=GITHUB_TOKEN="ghp_your_token" \
  --from-literal=GITHUB_WEBHOOK_SECRET="your_webhook_secret" \
  --from-literal=OPENAI_API_KEY="your_api_key" \
  --from-literal=AI_BASE_URL="https://api.deepseek.com" \
  --from-literal=AI_MODEL="deepseek-chat" \
  --from-literal=LOG_LEVEL="info" \
  --from-literal=PORT="8080" \
  -n agentfield
```

#### Required Secret Keys

The `agentfield-secrets` secret must contain:

| Key | Description | Example |
|-----|-------------|---------|
| `AGENTFIELD_URL` | AgentField control plane URL | `http://agentfield-control-plane:8080` |
| `GITHUB_TOKEN` | GitHub Personal Access Token | `ghp_...` |
| `GITHUB_WEBHOOK_SECRET` | GitHub webhook secret | `your-secret` |
| `OPENAI_API_KEY` | AI provider API key | DeepSeek or OpenRouter key |
| `AI_BASE_URL` | AI API base URL | `https://api.deepseek.com` |
| `AI_MODEL` | AI model name | `deepseek-chat` |
| `LOG_LEVEL` | Application log level | `info` |
| `PORT` | Application port | `8080` |

See `secret-example.yaml` for the complete secret template.

### Install the Chart

#### Default Installation

```bash
helm install agentfield ./helm/agentfield \
  --namespace agentfield \
  --create-namespace
```

#### Production Installation

```bash
helm install agentfield ./helm/agentfield \
  -n agentfield \
  --create-namespace \
  -f ./helm/agentfield/values-production.yaml \
  --set controlPlane.image.tag=main-abc123 \
  --set agent.image.tag=main-abc123
```

#### Install with Custom Values

```bash
helm install agentfield ./helm/agentfield \
  -n agentfield \
  --create-namespace \
  --set agent.replicaCount=5 \
  --set agent.resources.limits.cpu=3000m \
  --set controlPlane.ingress.host=myapp.example.com
```

## Configuration

### Key Values

| Parameter                       | Description                  | Default                         |
| ------------------------------- | ---------------------------- | ------------------------------- |
| `replicaCount`                  | Control plane replica count  | `1`                             |
| `controlPlane.image.repository` | Control plane image          | `ghcr.io/yourorg/af-code-agent` |
| `controlPlane.image.tag`        | Image tag                    | `latest`                        |
| `controlPlane.service.nodePort` | NodePort for external access | `30007`                         |
| `controlPlane.ingress.enabled`  | Enable ingress               | `true`                          |
| `controlPlane.ingress.host`     | Ingress hostname             | `agentfield.yongchenglow.com`   |
| `agent.enabled`                 | Deploy agent workers         | `true`                          |
| `agent.replicaCount`            | Initial agent replica count  | `2`                             |
| `agent.autoscaling.enabled`     | Enable agent autoscaling     | `true`                          |
| `agent.autoscaling.minReplicas` | Min agent replicas           | `2`                             |
| `agent.autoscaling.maxReplicas` | Max agent replicas           | `10`                            |

### Environment Variables

Both control plane and agents require these environment variables:

```yaml
controlPlane:
  env:
    - name: AGENTFIELD_URL
      value: "http://agentfield-control-plane:8080"
    - name: PORT
      value: "8080"
    - name: ENVIRONMENT
      value: "production"

agent:
  env:
    - name: AGENTFIELD_URL
      value: "http://agentfield-control-plane:8080"
    - name: NODE_TYPE
      value: "agent"
    - name: ENVIRONMENT
      value: "production"
```

### Resource Limits

#### Control Plane

```yaml
resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi
```

#### Agents (Production)

```yaml
resources:
  limits:
    cpu: 2000m
    memory: 2Gi
  requests:
    cpu: 1000m
    memory: 1Gi
```

## Upgrading

```bash
# Upgrade with new image tag
helm upgrade agentfield ./helm/agentfield \
  -n agentfield \
  -f ./helm/agentfield/values-production.yaml \
  --set controlPlane.image.tag=main-xyz789 \
  --set agent.image.tag=main-xyz789

# Check rollout status
kubectl rollout status deployment/agentfield-control-plane -n agentfield
kubectl rollout status deployment/agentfield-agent -n agentfield
```

## Uninstalling

```bash
helm uninstall agentfield -n agentfield
kubectl delete namespace agentfield
```

## Monitoring

### Check Pod Status

```bash
# All pods
kubectl get pods -n agentfield

# Control plane only
kubectl get pods -l app.kubernetes.io/component=control-plane -n agentfield

# Agents only
kubectl get pods -l app.kubernetes.io/component=agent -n agentfield
```

### View Logs

```bash
# Control plane logs
kubectl logs -f deployment/agentfield-control-plane -n agentfield

# Agent logs
kubectl logs -f deployment/agentfield-agent -n agentfield

# Specific pod logs
kubectl logs -f <pod-name> -n agentfield
```

### Check Autoscaling

```bash
# View HPA status
kubectl get hpa -n agentfield

# Detailed HPA info
kubectl describe hpa agentfield-agent -n agentfield
```

### Resource Usage

```bash
# Pod resource usage
kubectl top pods -n agentfield

# Node resource usage
kubectl top nodes
```

## Troubleshooting

### Pods Not Starting

1. Check pod events:

```bash
kubectl describe pod <pod-name> -n agentfield
```

1. Check image pull issues:

```bash
kubectl get events -n agentfield | grep -i "pull"
```

1. Verify secrets exist:

```bash
kubectl get secrets -n agentfield
```

### Agents Not Processing Tasks

1. Check agent logs:

```bash
kubectl logs -f deployment/agentfield-agent -n agentfield
```

1. Verify control plane connectivity:

```bash
kubectl get svc agentfield-control-plane -n agentfield
```

1. Check environment variables:

```bash
kubectl exec -it deployment/agentfield-agent -n agentfield -- env | grep AGENTFIELD
```

### High Memory/CPU Usage

1. Check resource usage:

```bash
kubectl top pods -n agentfield
```

1. Adjust resource limits in values file:

```yaml
agent:
  resources:
    limits:
      cpu: 3000m
      memory: 3Gi
```

1. Upgrade with new limits:

```bash
helm upgrade agentfield ./helm/agentfield -n agentfield -f values-custom.yaml
```

## Advanced Configuration

### Disable Agent Deployment

To deploy only the control plane:

```yaml
agent:
  enabled: false
```

### Custom Node Selection

```yaml
agent:
  nodeSelector:
    workload-type: ai-processing
  tolerations:
    - key: "ai-workload"
      operator: "Equal"
      value: "true"
      effect: "NoSchedule"
```

### Pod Affinity

Spread agents across nodes:

```yaml
agent:
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 100
          podAffinityTerm:
            labelSelector:
              matchExpressions:
                - key: app.kubernetes.io/component
                  operator: In
                  values:
                    - agent
            topologyKey: kubernetes.io/hostname
```

## Production Checklist

- [ ] Image pull secrets configured
- [ ] Application secrets created (GitHub App, API keys)
- [ ] Resource limits set appropriately
- [ ] Autoscaling configured for agents
- [ ] Ingress/DNS configured for control plane
- [ ] Health checks verified
- [ ] Monitoring/logging configured
- [ ] Backup strategy for configuration

## Support

For issues and questions:

- **Documentation**: See main [README.md](../../README.md)
- **Deployment Guide**: See [docs/DEPLOYMENT.md](../../docs/DEPLOYMENT.md)
- **GitHub Issues**: Report issues at repository URL

## Version

- **Chart Version**: 1.0.0
- **App Version**: 1.0.0
- **Last Updated**: 2026-02-07
