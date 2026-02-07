# GitHub App Setup Guide

This guide will walk you through creating and configuring a GitHub App for the Code Review Agent.

## Step 1: Create a GitHub App

1. Navigate to your GitHub organization or personal account settings
2. Go to **Developer settings** → **GitHub Apps** → **New GitHub App**

### Basic Information

- **GitHub App name**: `Code Review Agent` (or your preferred name)
- **Homepage URL**: Your deployment URL or repository URL
- **Webhook URL**: `https://your-domain.com/webhook`
  - During development, you can use tools like [ngrok](https://ngrok.com/) or [smee.io](https://smee.io/) for local testing
- **Webhook secret**: Generate a strong random secret (save this for later)

  ```bash
  # Generate a webhook secret
  openssl rand -hex 32
  ```

### Permissions

Configure the following repository permissions:

- **Contents**: Read & Write
  - Required to: Clone repositories, create branches, push commits

- **Pull requests**: Read & Write
  - Required to: Read PR content, post review comments, create PRs

- **Metadata**: Read-only
  - Required to: Access basic repository information

- **Checks**: Read-only (optional)
  - Required if: Using `check_suite.completed` webhook events

- **Workflows**: Read-only (optional)
  - Required if: Using `workflow_run.completed` webhook events

### Subscribe to Events

Select the following webhook events:

- ☑️ **Pull request**
  - Triggers: PR opened, synchronized, reopened

- ☑️ **Check suite** (optional but recommended)
  - Triggers: When CI/CD checks complete

- ☑️ **Workflow run** (optional)
  - Triggers: When GitHub Actions workflows complete

### Where can this GitHub App be installed?

- Choose **Only on this account** (or **Any account** if you plan to distribute it)

## Step 2: Generate Private Key

1. After creating the app, scroll to **Private keys**
2. Click **Generate a private key**
3. Download the `.pem` file
4. Save it securely (e.g., `github-app.pem` in your project root)
5. **Important**: Add `*.pem` to your `.gitignore` to prevent committing the key

## Step 3: Note Your App ID

1. On the GitHub App settings page, find your **App ID** at the top
2. Save this number for configuration

## Step 4: Install the App

1. Go to **Install App** in the left sidebar
2. Click **Install** next to your organization/account
3. Choose:
   - **All repositories** (if you trust the agent), or
   - **Only select repositories** (recommended for testing)
4. Complete the installation

## Step 5: Configure Environment Variables

Create a `.env` file in your project root:

```bash
# GitHub Configuration
GITHUB_APP_ID=123456                              # Your App ID from Step 3
GITHUB_PRIVATE_KEY_PATH=./github-app.pem          # Path to your private key
GITHUB_WEBHOOK_SECRET=your-generated-secret-here   # Secret from Step 1

# DeepSeek AI Configuration
OPENAI_API_KEY=your-deepseek-api-key
AI_BASE_URL=https://api.deepseek.com
AI_MODEL=deepseek-chat
AI_TEMPERATURE=0.2
AI_MAX_TOKENS=4000

# Application Settings
LOG_LEVEL=info
PORT=8080
```

## Step 6: Test Webhook Delivery

### Local Testing with ngrok

1. Install ngrok:

   ```bash
   # macOS
   brew install ngrok

   # Or download from https://ngrok.com/
   ```

2. Start ngrok:

   ```bash
   ngrok http 8080
   ```

3. Copy the HTTPS forwarding URL (e.g., `https://abc123.ngrok.io`)

4. Update your GitHub App's webhook URL:
   - Go to GitHub App settings
   - Update **Webhook URL** to `https://abc123.ngrok.io/webhook`

5. Start your agent:

   ```bash
   go run cmd/agent/main.go
   ```

6. Create a test PR in one of your repositories

7. Check ngrok's web interface at `http://localhost:4040` to see webhook deliveries

### Alternative: Using smee.io

1. Go to [smee.io](https://smee.io/) and click **Start a new channel**

2. Copy the webhook proxy URL

3. Update your GitHub App's webhook URL to the smee.io URL

4. Install and run the smee client:

   ```bash
   npm install -g smee-client
   smee -u https://smee.io/your-channel-id -t http://localhost:8080/webhook
   ```

5. Start your agent in another terminal

## Step 7: Verify Setup

1. Check the agent's health endpoint:

   ```bash
   curl http://localhost:8080/health
   ```

   Expected response:

   ```json
   {
     "status": "healthy",
     "mode": "safe"
   }
   ```

2. Create a test PR and verify:
   - Webhook is received (check logs)
   - Signature validation passes
   - Event is processed correctly

## Troubleshooting

### Webhook Signature Validation Fails

- Ensure `GITHUB_WEBHOOK_SECRET` matches the secret in GitHub App settings
- Check that you're using `X-Hub-Signature-256` header (not `X-Hub-Signature`)

### Private Key Parse Error

- Verify the `.pem` file is in valid PEM format
- Check file permissions: `chmod 600 github-app.pem`
- Ensure the path in `GITHUB_PRIVATE_KEY_PATH` is correct

### Installation Token Creation Fails

- Verify the App ID is correct
- Check that the private key belongs to the correct GitHub App
- Ensure the app is installed on the repository

### Webhooks Not Received

- Check webhook URL is publicly accessible
- Verify the URL includes `https://` (GitHub requires HTTPS in production)
- Check Recent Deliveries in GitHub App settings
- Look for error responses in the delivery details

## Security Best Practices

1. **Private Key Storage**
   - Never commit private keys to version control
   - Use environment variables or secret management services
   - Rotate keys periodically

2. **Webhook Secret**
   - Use a strong, randomly generated secret
   - Keep it separate from the codebase
   - Rotate if compromised

3. **Permissions**
   - Request only the minimum required permissions
   - Review permissions periodically
   - Use installation-specific tokens (1-hour expiry)

4. **Network Security**
   - Use HTTPS for all webhook endpoints
   - Implement rate limiting
   - Monitor for suspicious activity

## Production Deployment

For production deployment:

1. Deploy to a cloud provider (AWS, GCP, Azure, etc.)
2. Use a proper domain with HTTPS certificate
3. Update GitHub App webhook URL to production domain
4. Use secret management service for credentials (AWS Secrets Manager, HashiCorp Vault, etc.)
5. Set up monitoring and logging
6. Configure auto-scaling for high volume

## Resources

- [GitHub Apps Documentation](https://docs.github.com/en/apps)
- [Authenticating with GitHub Apps](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app)
- [Webhook Events and Payloads](https://docs.github.com/en/webhooks/webhook-events-and-payloads)
- [Best Practices for GitHub Apps](https://docs.github.com/en/apps/creating-github-apps/setting-up-a-github-app/best-practices-for-creating-a-github-app)
