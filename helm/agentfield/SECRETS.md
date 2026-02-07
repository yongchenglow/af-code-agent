# Secrets Management Guide

This guide explains how to manage secrets for the AgentField Helm chart deployment.

## Overview

The Helm chart expects an **external Kubernetes secret** named `agentfield-secrets` to exist in the namespace before deployment. This secret contains all sensitive configuration values.

## Secret Name and Namespace

- **Name**: `agentfield-secrets` (required, do not change)
- **Namespace**: `agentfield` (or your target namespace)
- **Type**: `Opaque`

## Required Environment Variables

The secret must contain all variables from `.env.example`:

| Variable | Description | Example Value |
|----------|-------------|---------------|
| `AGENTFIELD_URL` | Control plane service URL | `http://agentfield-control-plane:8080` |
| `GITHUB_TOKEN` | GitHub Personal Access Token | `ghp_xxxxxxxxxxxxx` |
| `GITHUB_WEBHOOK_SECRET` | GitHub webhook validation secret | `your-webhook-secret` |
| `AI_API_KEY` | AI provider API key | DeepSeek or OpenRouter key |
| `AI_BASE_URL` | AI API endpoint | `https://api.deepseek.com` |
| `AI_MODEL` | AI model identifier | `deepseek-chat` |
| `LOG_LEVEL` | Application log level | `info` |
| `PORT` | Application port | `8080` |

## Creation Methods

### Method 1: From .env File (Recommended)

If you have a `.env` file with all required variables:

```bash
# Create namespace first
kubectl create namespace agentfield

# Create secret from .env file
kubectl create secret generic agentfield-secrets \
  --from-env-file=.env \
  -n agentfield

# Verify secret was created
kubectl get secret agentfield-secrets -n agentfield
kubectl describe secret agentfield-secrets -n agentfield
```

### Method 2: From YAML Template

```bash
# Copy the example
cp secret-example.yaml secret.yaml

# Edit with your actual values
vim secret.yaml
# or
nano secret.yaml

# Apply the secret
kubectl apply -f secret.yaml

# IMPORTANT: Delete the file (never commit secrets!)
rm secret.yaml

# Verify
kubectl get secret agentfield-secrets -n agentfield
```

### Method 3: From Command Line

```bash
kubectl create secret generic agentfield-secrets \
  --from-literal=AGENTFIELD_URL="http://agentfield-control-plane:8080" \
  --from-literal=GITHUB_TOKEN="ghp_your_token_here" \
  --from-literal=GITHUB_WEBHOOK_SECRET="your_webhook_secret" \
  --from-literal=AI_API_KEY="your_api_key_here" \
  --from-literal=AI_BASE_URL="https://api.deepseek.com" \
  --from-literal=AI_MODEL="deepseek-chat" \
  --from-literal=LOG_LEVEL="info" \
  --from-literal=PORT="8080" \
  -n agentfield
```

## Verification

After creating the secret, verify it exists and contains all required keys:

```bash
# Check if secret exists
kubectl get secret agentfield-secrets -n agentfield

# View secret keys (values are hidden)
kubectl describe secret agentfield-secrets -n agentfield

# Decode and view secret (use carefully!)
kubectl get secret agentfield-secrets -n agentfield -o jsonpath='{.data}' | jq 'map_values(@base64d)'
```

Expected output should show all 8 required keys.

## Updating Secrets

### Update Individual Values

```bash
# Patch specific key
kubectl patch secret agentfield-secrets -n agentfield \
  -p '{"stringData":{"AI_API_KEY":"new_api_key_value"}}'

# Restart deployments to pick up changes
kubectl rollout restart deployment/agentfield-control-plane -n agentfield
kubectl rollout restart deployment/agentfield-agent -n agentfield
```

### Replace Entire Secret

```bash
# Delete old secret
kubectl delete secret agentfield-secrets -n agentfield

# Create new secret
kubectl create secret generic agentfield-secrets \
  --from-env-file=.env \
  -n agentfield

# Restart deployments
kubectl rollout restart deployment/agentfield-control-plane -n agentfield
kubectl rollout restart deployment/agentfield-agent -n agentfield
```

## Deployment Workflow

The complete deployment workflow with secrets:

```bash
# 1. Create namespace
kubectl create namespace agentfield

# 2. Create image pull secret (for GHCR)
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=YOUR_GITHUB_USERNAME \
  --docker-password=YOUR_GITHUB_PAT \
  --docker-email=YOUR_EMAIL \
  -n agentfield

# 3. Create application secrets
kubectl create secret generic agentfield-secrets \
  --from-env-file=.env \
  -n agentfield

# 4. Verify secrets
kubectl get secrets -n agentfield

# 5. Deploy Helm chart
helm install agentfield ./helm/agentfield \
  -n agentfield \
  -f ./helm/agentfield/values-production.yaml \
  --set controlPlane.image.tag=v1.0.0 \
  --set agent.image.tag=v1.0.0

# 6. Verify deployment
kubectl get pods -n agentfield
```

## Security Best Practices

### DO:
✅ Create secrets before deploying the Helm chart
✅ Use `.env` file locally and keep it in `.gitignore`
✅ Use Kubernetes RBAC to restrict secret access
✅ Rotate secrets regularly
✅ Use different secrets for different environments
✅ Consider using sealed secrets or external secret managers for production

### DON'T:
❌ Never commit `secret.yaml` or `.env` files to version control
❌ Never hardcode secrets in Helm values files
❌ Never log or print secret values
❌ Never share secrets via insecure channels (email, Slack, etc.)
❌ Never use the same secrets across environments

## Alternative: External Secret Managers

For production environments, consider using external secret management:

### AWS Secrets Manager
```bash
# Install External Secrets Operator
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets external-secrets/external-secrets -n external-secrets-system --create-namespace

# Create SecretStore pointing to AWS Secrets Manager
# Then create ExternalSecret that syncs to agentfield-secrets
```

### HashiCorp Vault
```bash
# Use Vault Agent Injector
helm repo add hashicorp https://helm.releases.hashicorp.com
helm install vault hashicorp/vault --set "injector.enabled=true"

# Annotate deployments to inject secrets from Vault
```

### Sealed Secrets (Bitnami)
```bash
# Install Sealed Secrets controller
helm repo add sealed-secrets https://bitnami-labs.github.io/sealed-secrets
helm install sealed-secrets sealed-secrets/sealed-secrets -n kube-system

# Encrypt secrets and commit to git
kubeseal --format yaml < secret.yaml > sealed-secret.yaml
kubectl apply -f sealed-secret.yaml
```

## Troubleshooting

### Secret Not Found Error

```bash
Error: Secret "agentfield-secrets" not found
```

**Solution**: Create the secret before deploying:
```bash
kubectl create secret generic agentfield-secrets --from-env-file=.env -n agentfield
```

### Pods CrashLoopBackOff with Missing Env Vars

**Solution**: Check if secret has all required keys:
```bash
kubectl get secret agentfield-secrets -n agentfield -o json | jq '.data | keys'
```

### Permission Denied Errors

**Solution**: Ensure service account has access to secrets:
```bash
kubectl auth can-i get secrets --as=system:serviceaccount:agentfield:agentfield-control-plane -n agentfield
```

## Environment-Specific Secrets

### Development
```bash
kubectl create secret generic agentfield-secrets \
  --from-env-file=.env.dev \
  -n agentfield-dev
```

### Staging
```bash
kubectl create secret generic agentfield-secrets \
  --from-env-file=.env.staging \
  -n agentfield-staging
```

### Production
```bash
kubectl create secret generic agentfield-secrets \
  --from-env-file=.env.production \
  -n agentfield
```

## Support

For issues with secrets management:
1. Verify secret exists: `kubectl get secret agentfield-secrets -n agentfield`
2. Check secret keys: `kubectl describe secret agentfield-secrets -n agentfield`
3. View pod logs: `kubectl logs -f deployment/agentfield-control-plane -n agentfield`
4. Check environment variables in pod: `kubectl exec -it <pod-name> -n agentfield -- env | grep -E "(GITHUB|AI|AGENTFIELD)"`

See the main [README.md](README.md) for additional documentation.
