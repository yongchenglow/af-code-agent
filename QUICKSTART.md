# AgentField Deployment Quick Start

This guide will get you up and running with AgentField in 5 minutes.

## Prerequisites

- Kubernetes cluster with kubectl configured
- Helm 3.13.0+
- GitHub Personal Access Token
- AI API key (DeepSeek or OpenRouter)

## Step 1: Create Namespace

```bash
kubectl create namespace agentfield
```

## Step 2: Create Image Pull Secret

```bash
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=YOUR_GITHUB_USERNAME \
  --docker-password=YOUR_GITHUB_PAT \
  --docker-email=YOUR_EMAIL \
  -n agentfield
```

## Step 3: Create Application Secrets

### Option A: From .env file (Recommended)

```bash
# Make sure you have a .env file with all required variables
kubectl create secret generic agentfield-secrets \
  --from-env-file=.env \
  -n agentfield
```

### Option B: Manually specify values

```bash
kubectl create secret generic agentfield-secrets \
  --from-literal=AGENTFIELD_URL="http://agentfield-control-plane:8080" \
  --from-literal=GITHUB_TOKEN="ghp_your_token" \
  --from-literal=GITHUB_WEBHOOK_SECRET="your_secret" \
  --from-literal=OPENAI_API_KEY="your_ai_key" \
  --from-literal=AI_BASE_URL="https://api.deepseek.com" \
  --from-literal=AI_MODEL="deepseek-chat" \
  --from-literal=LOG_LEVEL="info" \
  --from-literal=PORT="8080" \
  -n agentfield
```

## Step 4: Deploy with Helm

### Production Deployment

```bash
helm install agentfield ./helm/agentfield \
  -n agentfield \
  -f ./helm/agentfield/values-production.yaml \
  --set controlPlane.image.tag=latest \
  --set agent.image.tag=latest
```

### Development Deployment

```bash
helm install agentfield ./helm/agentfield \
  -n agentfield
```

## Step 5: Verify Deployment

```bash
# Check all pods are running
kubectl get pods -n agentfield

# Expected output:
# NAME                                        READY   STATUS    RESTARTS   AGE
# agentfield-control-plane-xxxxxxxxxx-xxxxx   1/1     Running   0          1m
# agentfield-agent-xxxxxxxxxx-xxxxx           1/1     Running   0          1m
# agentfield-agent-xxxxxxxxxx-yyyyy           1/1     Running   0          1m
# agentfield-agent-xxxxxxxxxx-zzzzz           1/1     Running   0          1m

# Check services
kubectl get svc -n agentfield

# View logs
kubectl logs -f deployment/agentfield-control-plane -n agentfield
```

## Step 6: Access the Application

### Via NodePort (Local/Testing)

```bash
# Get the NodePort
export NODE_PORT=$(kubectl get svc agentfield-control-plane -n agentfield -o jsonpath='{.spec.ports[0].nodePort}')
export NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[0].address}')

echo "Access at: http://$NODE_IP:$NODE_PORT"
```

### Via Port Forward (Local Development)

```bash
kubectl port-forward svc/agentfield-control-plane 8080:8080 -n agentfield

# Access at: http://localhost:8080
```

## Quick Commands

### View All Resources
```bash
kubectl get all -n agentfield
```

### Scale Agents
```bash
# Manual scaling
kubectl scale deployment agentfield-agent --replicas=5 -n agentfield

# Or update HPA
kubectl edit hpa agentfield-agent -n agentfield
```

### View Logs
```bash
# Control plane logs
kubectl logs -f deployment/agentfield-control-plane -n agentfield

# Agent logs
kubectl logs -f deployment/agentfield-agent -n agentfield

# All pods
kubectl logs -f -l app.kubernetes.io/name=agentfield -n agentfield
```

### Update Deployment
```bash
# Update image tag
helm upgrade agentfield ./helm/agentfield \
  -n agentfield \
  -f ./helm/agentfield/values-production.yaml \
  --set controlPlane.image.tag=v1.1.0 \
  --set agent.image.tag=v1.1.0
```

### Rollback
```bash
helm rollback agentfield -n agentfield
```

### Uninstall
```bash
helm uninstall agentfield -n agentfield
kubectl delete namespace agentfield
```

## Troubleshooting

### Pods Not Starting
```bash
kubectl describe pod <pod-name> -n agentfield
```

### Check Secret
```bash
kubectl get secret agentfield-secrets -n agentfield
kubectl describe secret agentfield-secrets -n agentfield
```

### Check Events
```bash
kubectl get events -n agentfield --sort-by='.lastTimestamp'
```

## What Gets Deployed

### Control Plane (1-2 replicas)
- Handles GitHub webhooks
- Orchestrates tasks via AgentField
- Exposes HTTP API on port 8080
- NodePort: 30007 (production)

### Agent Workers (3-10 replicas)
- Execute code review tasks
- Auto-scale based on CPU/memory
- Connect to control plane
- No external service

## Architecture

```
┌────────────────────────────┐
│  Control Plane (1-2 pods)  │
│  - Port: 8080              │
│  - NodePort: 30007         │
└─────────────┬──────────────┘
              │ AgentField
              ↓
┌────────────────────────────┐
│  Agent Workers (3-10 pods) │
│  - Auto-scaling            │
│  - Code reviews            │
│  - Fix generation          │
└────────────────────────────┘
```

## Next Steps

1. Configure GitHub webhook to point to your deployment
2. Set up ingress/DNS for production access
3. Configure monitoring and logging
4. Set up backup for secrets and configuration
5. Review security settings

## Documentation

- **Helm Chart**: [helm/agentfield/README.md](helm/agentfield/README.md)
- **Secrets Management**: [helm/agentfield/SECRETS.md](helm/agentfield/SECRETS.md)
- **Deployment Guide**: [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)
- **Main README**: [README.md](README.md)

## Support

For issues:
1. Check pod logs: `kubectl logs -f deployment/agentfield-control-plane -n agentfield`
2. Check events: `kubectl get events -n agentfield`
3. Verify secrets: `kubectl get secret agentfield-secrets -n agentfield`
4. Review documentation links above
