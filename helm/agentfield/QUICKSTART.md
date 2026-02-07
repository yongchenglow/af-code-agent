# Quick Start Guide

This is a quick reference for deploying AgentField agents. For detailed information, see [README.md](README.md).

## Prerequisites Checklist

- [ ] Kubernetes cluster running
- [ ] AgentField control-plane deployed and accessible
- [ ] `kubectl` configured and connected
- [ ] Helm 3.13.0+ installed
- [ ] GitHub personal access token ready
- [ ] AI API key ready (OpenAI, DeepSeek, or OpenRouter)

## 5-Minute Setup

### 1. Create Namespace

```bash
kubectl create namespace agentfield
```

### 2. Create Secrets

```bash
# GHCR Image Pull Secret
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=YOUR_GITHUB_USERNAME \
  --docker-password=YOUR_GITHUB_PAT \
  -n agentfield

# Application Secrets
kubectl create secret generic agentfield-secrets \
  --from-literal=AGENTFIELD_URL="http://agentfield-control-plane:8080" \
  --from-literal=GITHUB_TOKEN="ghp_YOUR_TOKEN" \
  --from-literal=GITHUB_WEBHOOK_SECRET="your-webhook-secret" \
  --from-literal=OPENROUTER_API_KEY="sk-or-v1-YOUR_KEY" \
  --from-literal=AI_BASE_URL="https://openrouter.ai/api/v1" \
  --from-literal=AI_MODEL="deepseek/deepseek-chat" \
  --from-literal=LOG_LEVEL="info" \
  --from-literal=PORT="8080" \
  -n agentfield
```

### 3. Install Chart

```bash
helm install agentfield-agent ./helm/agentfield -n agentfield
```

### 4. Verify

```bash
# Check pod status
kubectl get pods -n agentfield

# Should show:
# NAME                               READY   STATUS    RESTARTS   AGE
# agentfield-agent-xxxxx-yyyyy       1/1     Running   0          30s

# Check logs
kubectl logs -f -l app.kubernetes.io/component=agent -n agentfield
```

## Common Configurations

### Development (Single Agent)

```bash
helm install agentfield-agent ./helm/agentfield \
  -n agentfield \
  --set agent.replicaCount=1
```

### Production (Multiple Agents with Autoscaling)

```bash
helm install agentfield-agent ./helm/agentfield \
  -n agentfield \
  --set agent.replicaCount=3 \
  --set agent.autoscaling.enabled=true \
  --set agent.autoscaling.minReplicas=2 \
  --set agent.autoscaling.maxReplicas=10
```

### Specific Image Version

```bash
helm install agentfield-agent ./helm/agentfield \
  -n agentfield \
  --set agent.image.tag="main-abc123def456"
```

## Troubleshooting Quick Fixes

### Pods Not Starting

```bash
# Describe pod to see events
kubectl describe pod -l app.kubernetes.io/component=agent -n agentfield

# Common fixes:
# 1. Image pull issue - verify ghcr-secret
kubectl get secret ghcr-secret -n agentfield

# 2. Secret missing - verify agentfield-secrets
kubectl get secret agentfield-secrets -n agentfield
```

### Agents Can't Connect to Control Plane

```bash
# Check AGENTFIELD_URL
kubectl get secret agentfield-secrets -n agentfield \
  -o jsonpath='{.data.AGENTFIELD_URL}' | base64 -d

# Test connectivity
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -n agentfield -- \
  curl http://agentfield-control-plane:8080/health
```

### Logs Showing Errors

```bash
# View recent logs
kubectl logs -l app.kubernetes.io/component=agent -n agentfield --tail=50

# Follow logs
kubectl logs -f -l app.kubernetes.io/component=agent -n agentfield

# Get logs from all pods
kubectl logs -l app.kubernetes.io/component=agent -n agentfield --all-containers=true
```

## Useful Commands

```bash
# Scale agents
kubectl scale deployment/agentfield-agent --replicas=5 -n agentfield

# Restart agents (rolling restart)
kubectl rollout restart deployment/agentfield-agent -n agentfield

# Check resource usage
kubectl top pods -n agentfield

# Get agent deployment info
kubectl get deployment agentfield-agent -n agentfield -o wide

# Delete everything
helm uninstall agentfield-agent -n agentfield
kubectl delete namespace agentfield
```

## Next Steps

- Read the [README.md](README.md) for detailed configuration options
- Check [SECRETS.md](SECRETS.md) for secret management best practices
- Review [MIGRATION.md](MIGRATION.md) if upgrading from an older version
- Monitor your agents with `kubectl logs` and `kubectl top`
- Set up autoscaling based on your workload

## Support

For issues, check the [README.md](README.md) troubleshooting section or open an issue in the repository.
