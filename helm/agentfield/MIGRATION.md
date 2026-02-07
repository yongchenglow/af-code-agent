# Helm Chart Migration Guide

## What Changed

The Helm chart has been refactored to deploy **agents only**. Previously, the chart incorrectly tried to deploy both a control-plane and agents using the same agent image, which caused both to crash.

### Before

- Chart tried to deploy both control-plane and agents
- Control-plane used the agent image (wrong)
- Both pods crashed because the agent image tried to connect to itself
- Circular dependency issue

### After

- Chart only deploys agents
- AgentField control-plane must be deployed separately
- Agents connect to an external control-plane via `AGENTFIELD_URL`
- Clean, simple deployment model

## Breaking Changes

1. **Control-plane removed**: The chart no longer deploys a control-plane component
2. **Service removed**: No service is created (agents don't need external access)
3. **Ingress removed**: No ingress configuration (control-plane handles webhooks)
4. **Values structure simplified**: Removed `controlPlane.*` values, kept only `agent.*`
5. **Chart renamed**: From `agentfield` to `agentfield-agent`

## Migration Steps

### 1. Uninstall Old Release

```bash
helm uninstall agentfield-control-plane -n agentfield
```

### 2. Deploy AgentField Control Plane

You need to deploy the AgentField control-plane separately. This is typically done by:

**Option A: Using the official AgentField chart** (recommended)
```bash
# Get the AgentField control-plane chart from the AgentField project
helm repo add agentfield https://agentfield.io/charts
helm install agentfield-cp agentfield/control-plane -n agentfield
```

**Option B: Deploy from AgentField source**
```bash
# Clone the AgentField repository
git clone https://github.com/Agent-Field/agentfield
cd agentfield

# Deploy control-plane
# (Follow their installation instructions)
```

### 3. Update Your Secret

Ensure your `agentfield-secrets` has the correct `AGENTFIELD_URL` pointing to the control-plane:

```bash
kubectl get secret agentfield-secrets -n agentfield -o yaml > secret-backup.yaml

# Edit and update AGENTFIELD_URL
kubectl edit secret agentfield-secrets -n agentfield

# Or recreate:
kubectl delete secret agentfield-secrets -n agentfield
kubectl create secret generic agentfield-secrets \
  --from-literal=AGENTFIELD_URL="http://agentfield-control-plane:8080" \
  --from-literal=GITHUB_TOKEN="ghp_xxxx" \
  --from-literal=AI_API_KEY="sk-xxxx" \
  # ... other values
  --namespace=agentfield
```

### 4. Install New Agent Chart

```bash
# Install with default values
helm install agentfield-agent ./helm/agentfield -n agentfield

# Or with custom values
helm install agentfield-agent ./helm/agentfield \
  -n agentfield \
  --set agent.replicaCount=3 \
  --set agent.image.tag="main-abc123"
```

## New Values Structure

### Old (v1.0.0)

```yaml
controlPlane:
  image:
    repository: ghcr.io/yongchenglow/af-code-agent
    tag: "latest"
  replicaCount: 1
  service:
    type: NodePort
    port: 8080
  # ... many more control-plane settings

agent:
  enabled: true
  replicaCount: 2
  image:
    repository: ghcr.io/yongchenglow/af-code-agent
    tag: "main-f3e6da2660865678d42f507c7ed9f4f35b479bb8"
  # ... agent settings
```

### New (v2.0.0+)

```yaml
serviceAccount:
  create: true
  name: "agentfield-agents"

agent:
  enabled: true
  replicaCount: 1
  image:
    repository: ghcr.io/yongchenglow/af-code-agent
    tag: ""  # Uses Chart.AppVersion
  externalSecret:
    enabled: true
    name: "agentfield-secrets"
  resources:
    requests:
      cpu: 250m
      memory: 512Mi
    limits:
      cpu: 1
      memory: 2Gi
  autoscaling:
    enabled: false
    minReplicas: 1
    maxReplicas: 10
```

## Common Issues After Migration

### Agents Can't Connect to Control Plane

**Symptom**: Logs show "context deadline exceeded" errors

**Solution**: Verify `AGENTFIELD_URL` in the secret:

```bash
kubectl get secret agentfield-secrets -n agentfield -o jsonpath='{.data.AGENTFIELD_URL}' | base64 -d
```

Should output something like: `http://agentfield-control-plane:8080`

Test connectivity:

```bash
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -n agentfield -- \
  curl http://agentfield-control-plane:8080/health
```

### No Control Plane Found

**Symptom**: Agents can't find the control-plane service

**Solution**: Deploy the AgentField control-plane first (see step 2 above)

### Image Pull Errors

**Symptom**: `ImagePullBackOff` errors

**Solution**: Verify GHCR secret exists:

```bash
kubectl get secret ghcr-secret -n agentfield
```

If missing, create it:

```bash
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=<github-username> \
  --docker-password=<github-pat> \
  --namespace=agentfield
```

## Questions?

- Check the [README.md](README.md) for complete installation instructions
- Check the [SECRETS.md](SECRETS.md) for secret configuration details
- Open an issue if you encounter problems
