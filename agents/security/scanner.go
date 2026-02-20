package security

import (
	"regexp"
	"strings"
)

// SecretFinding represents a detected secret in code
type SecretFinding struct {
	// Type is the type of secret detected (e.g., "AWS Access Key ID")
	Type string `json:"type"`
	// FilePath is the path to the file containing the secret
	FilePath string `json:"file_path"`
	// Line is the line number where the secret was found
	Line int `json:"line"`
	// Column is the column where the secret starts
	Column int `json:"column"`
	// Value is the detected secret value (may be truncated for safety)
	Value string `json:"value"`
	// Severity is the severity level (always Critical for secrets)
	Severity string `json:"severity"`
	// Description provides a description of the finding
	Description string `json:"description"`
	// Remediation provides steps to fix the issue
	Remediation string `json:"remediation"`
}

// Scanner detects secrets and sensitive information in code
type Scanner struct {
	patterns map[string]*regexp.Regexp
}

// NewScanner creates a new secret scanner
func NewScanner() *Scanner {
	return &Scanner{
		patterns: secretPatterns,
	}
}

// secretPatterns defines regex patterns for detecting common secrets
var secretPatterns = map[string]*regexp.Regexp{
	// AWS Credentials
	"AWS Access Key ID":     regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	"AWS Secret Access Key": regexp.MustCompile(`(?i)(?:aws[_\-]?secret[_\-]?access[_\-]?key|AWS_SECRET_ACCESS_KEY)\s*[=:]\s*['"]?([A-Za-z0-9/+=]{40})['"]?`),
	"AWS Account ID":        regexp.MustCompile(`(?i)(?:aws[_\-]?account[_\-]?id|AWS_ACCOUNT_ID)\s*[=:]\s*['"]?([0-9]{12})['"]?`),

	// GitHub Tokens
	"GitHub Personal Access Token":  regexp.MustCompile(`ghp_[A-Za-z0-9_]{36,}`),
	"GitHub OAuth Access Token":     regexp.MustCompile(`gho_[A-Za-z0-9_]{36,}`),
	"GitHub User-to-Server Token":   regexp.MustCompile(`ghu_[A-Za-z0-9_]{36,}`),
	"GitHub Server-to-Server Token": regexp.MustCompile(`ghs_[A-Za-z0-9_]{36,}`),
	"GitHub Refresh Token":          regexp.MustCompile(`ghr_[A-Za-z0-9_]{36,}`),
	"GitHub Fine-Grained PAT":       regexp.MustCompile(`github_pat_[A-Za-z0-9_]{22,}_[A-Za-z0-9_]{59,}`),
	"GitHub App Installation Token": regexp.MustCompile(`ghu_[A-Za-z0-9_]{36,}`),
	"GitHub Action Token":           regexp.MustCompile(`ghs_[A-Za-z0-9_]{36,}`),

	// GitLab Tokens
	"GitLab Personal Access Token": regexp.MustCompile(`glpat-[A-Za-z0-9\-]{20,}`),
	"GitLab Runner Token":          regexp.MustCompile(`glrt-[A-Za-z0-9\-]{20,}`),
	"GitLab Pipeline Token":        regexp.MustCompile(`glptt-[A-Za-z0-9\-]{40,}`),

	// Google Cloud
	"Google API Key":         regexp.MustCompile(`AIza[0-9A-Za-z\-_]{35}`),
	"Google OAuth Token":     regexp.MustCompile(`ya29\.[0-9A-Za-z\-_]+`),
	"Google Service Account": regexp.MustCompile(`"type":\s*"service_account"`),

	// Azure
	"Azure Storage Account Key": regexp.MustCompile(`(?i)(?:AZURE[_\-]?STORAGE[_\-]?ACCOUNT[_\-]?KEY|AccountKey)\s*[=:]\s*['"]?([A-Za-z0-9+/=]{88})['"]?`),
	"Azure Connection String":   regexp.MustCompile(`(?i)DefaultEndpointsProtocol=https;AccountName=[^;]+;AccountKey=[A-Za-z0-9+/=]+`),
	"Azure SAS Token":           regexp.MustCompile(`(?i)SharedAccessSignature\s+[?]?sig=[A-Za-z0-9%]+`),

	// Private Keys
	"RSA Private Key":     regexp.MustCompile(`-----BEGIN RSA PRIVATE KEY-----`),
	"EC Private Key":      regexp.MustCompile(`-----BEGIN EC PRIVATE KEY-----`),
	"OpenSSH Private Key": regexp.MustCompile(`-----BEGIN OPENSSH PRIVATE KEY-----`),
	"PGP Private Key":     regexp.MustCompile(`-----BEGIN PGP PRIVATE KEY BLOCK-----`),
	"Generic Private Key": regexp.MustCompile(`-----BEGIN PRIVATE KEY-----`),

	// JWT
	"JWT Token": regexp.MustCompile(`eyJ[A-Za-z0-9\-_]+\.eyJ[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+`),

	// Slack
	"Slack Bot Token":       regexp.MustCompile(`xoxb-[0-9]{10,13}-[0-9]{10,13}[a-zA-Z0-9-]*`),
	"Slack User Token":      regexp.MustCompile(`xoxp-[0-9]{10,13}-[0-9]{10,13}[a-zA-Z0-9-]*`),
	"Slack Workspace Token": regexp.MustCompile(`xoxa-[0-9]{10,13}-[0-9]{10,13}[a-zA-Z0-9-]*`),

	// Stripe
	"Stripe API Key":            regexp.MustCompile(`sk_live_[0-9a-zA-Z]{24,}`),
	"Stripe Restricted API Key": regexp.MustCompile(`rk_live_[0-9a-zA-Z]{24,}`),
	"Stripe Publishable Key":    regexp.MustCompile(`pk_live_[0-9a-zA-Z]{24,}`),

	// Twilio
	"Twilio API Key":     regexp.MustCompile(`SK[0-9a-fA-F]{32}`),
	"Twilio Account SID": regexp.MustCompile(`AC[0-9a-fA-F]{32}`),
	"Twilio Auth Token":  regexp.MustCompile(`(?i)twilio[_\-]?auth[_\-]?token\s*[=:]\s*['"]?[0-9a-f]{32}['"]?`),

	// Database Connection Strings
	"PostgreSQL Connection String": regexp.MustCompile(`(?i)postgres(?:ql)?://[^:]+:[^@]+@[^/]+/\w+`),
	"MySQL Connection String":      regexp.MustCompile(`(?i)mysql://[^:]+:[^@]+@[^/]+/\w+`),
	"MongoDB Connection String":    regexp.MustCompile(`(?i)mongodb(?:\+srv)?://[^:]+:[^@]+@[^/]+`),
	"Redis Connection String":      regexp.MustCompile(`(?i)redis://[^:]*:[^@]+@[^/]+`),

	// Generic Secrets
	"Generic Secret":  regexp.MustCompile(`(?i)(?:api[_\-]?key|secret|password|passwd|pwd|auth[_\-]?token|access[_\-]?token)\s*[=:]\s*['"][^'"]{8,}['"]`),
	"Generic API Key": regexp.MustCompile(`(?i)(?:api[_\-]?key|apikey)\s*[=:]\s*['"][A-Za-z0-9\-_]{16,}['"]`),
	"Bearer Token":    regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+`),

	// NPM Tokens
	"NPM Token": regexp.MustCompile(`//registry\.npmjs\.org/:_authToken=[A-Za-z0-9\-_]+`),

	// SendGrid
	"SendGrid API Key": regexp.MustCompile(`SG\.[A-Za-z0-9\-_]{22}\.[A-Za-z0-9\-_]{43}`),

	// Mailgun
	"Mailgun API Key": regexp.MustCompile(`key-[0-9a-zA-Z]{32}`),

	// Datadog
	"Datadog API Key": regexp.MustCompile(`(?i)datadog[_\-]?api[_\-]?key\s*[=:]\s*['"]?[a-f0-9]{32}['"]?`),

	// Square
	"Square Access Token": regexp.MustCompile(`sq0atp-[0-9A-Za-z\-_]{22}`),
	"Square OAuth Secret": regexp.MustCompile(`sq0csp-[0-9A-Za-z\-_]{43}`),

	// Telegram
	"Telegram Bot Token": regexp.MustCompile(`[0-9]+:[A-Za-z0-9\-_]{35}`),

	// Password in URL
	"Password in URL": regexp.MustCompile(`[a-zA-Z0-9]+://[^:]+:[^@]+@`),
}

// remediationGuidance provides remediation steps for each secret type
var remediationGuidance = map[string]string{
	"AWS Access Key ID":             "1. Immediately rotate the compromised access key in AWS IAM console\n2. Review CloudTrail logs for unauthorized access\n3. Use IAM roles instead of access keys where possible\n4. Store credentials in AWS Secrets Manager or environment variables",
	"AWS Secret Access Key":         "1. Immediately rotate the compromised secret key in AWS IAM console\n2. Review CloudTrail logs for unauthorized access\n3. Never commit AWS credentials to version control\n4. Use IAM roles or AWS Secrets Manager",
	"AWS Account ID":                "1. Review AWS account access policies\n2. Enable MFA on all IAM users\n3. Monitor CloudTrail for suspicious activity",
	"GitHub Personal Access Token":  "1. Immediately revoke the token in GitHub Settings > Developer settings > Personal access tokens\n2. Generate a new token with minimal required scopes\n3. Store tokens in environment variables or secret managers\n4. Consider using GitHub Apps for server-to-server authentication",
	"GitHub OAuth Access Token":     "1. Revoke the OAuth token immediately\n2. Review OAuth application permissions\n3. Rotate client secrets\n4. Implement proper token storage",
	"GitHub User-to-Server Token":   "1. Revoke the token in GitHub App settings\n2. Review app permissions\n3. Rotate client secret\n4. Ensure tokens are stored securely",
	"GitHub Server-to-Server Token": "1. Revoke the token immediately\n2. Generate new token with minimal permissions\n3. Store in environment variables or secret manager\n4. Review app access logs",
	"GitHub Fine-Grained PAT":       "1. Revoke the fine-grained PAT in GitHub settings\n2. Create new PAT with repository-specific access\n3. Set appropriate expiration date\n4. Store securely",
	"GitLab Personal Access Token":  "1. Revoke token in GitLab Settings > Access Tokens\n2. Create new token with minimal scopes\n3. Use environment variables for storage\n4. Enable token expiration",
	"GitLab Runner Token":           "1. Regenerate runner token in GitLab CI/CD settings\n2. Review runner permissions\n3. Secure runner configuration file",
	"Google API Key":                "1. Regenerate API key in Google Cloud Console\n2. Restrict key to specific APIs and referrers\n3. Use environment variables or Secret Manager\n4. Monitor usage in Cloud Logging",
	"Google OAuth Token":            "1. Revoke OAuth token in Google Cloud Console\n2. Review OAuth consent screen settings\n3. Rotate client credentials\n4. Implement proper token refresh",
	"Private Key":                   "1. Generate new key pair immediately\n2. Revoke old key from all services\n3. Store private keys in secure key management system\n4. Never commit private keys to version control\n5. Use hardware security modules for production",
	"JWT Token":                     "1. Invalidate the JWT by rotating the signing key\n2. Review token generation and validation logic\n3. Implement short token expiration times\n4. Use refresh tokens for long-lived sessions",
	"Slack Bot Token":               "1. Revoke token in Slack App settings\n2. Regenerate with minimal scopes\n3. Store in environment variables\n4. Review bot permissions",
	"Slack User Token":              "1. Revoke token immediately\n2. Review app installation\n3. Rotate workspace tokens\n4. Audit app activity",
	"Stripe API Key":                "1. Roll API key in Stripe Dashboard\n2. Review API logs for unauthorized access\n3. Implement key rotation policy\n4. Use restricted keys where possible",
	"Twilio API Key":                "1. Regenerate API key in Twilio Console\n2. Review API usage logs\n3. Implement IP whitelisting\n4. Use environment variables",
	"Database Connection String":    "1. Change database password immediately\n2. Update connection string in all environments\n3. Use environment variables or secret managers\n4. Implement network-level access controls\n5. Review database access logs",
	"Generic Secret":                "1. Rotate the secret/password immediately\n2. Use environment variables or secret managers\n3. Implement secret rotation policy\n4. Review access logs for unauthorized use",
	"Generic API Key":               "1. Regenerate API key\n2. Restrict API key permissions\n3. Use environment variables\n4. Implement rate limiting",
	"NPM Token":                     "1. Regenerate NPM token\n2. Update .npmrc with new token\n3. Use CI/CD secrets for token storage\n4. Review published packages",
	"SendGrid API Key":              "1. Regenerate API key in SendGrid\n2. Review email sending logs\n3. Implement IP access management\n4. Use minimal required permissions",
	"Mailgun API Key":               "1. Regenerate API key in Mailgun\n2. Review email logs\n3. Implement domain verification\n4. Use environment variables",
	"Password in URL":               "1. Remove password from URL immediately\n2. Use environment variables or config files\n3. Rotate the exposed password\n4. Review git history for exposure",
	"Bearer Token":                  "1. Invalidate the token\n2. Review authentication flow\n3. Implement token refresh mechanism\n4. Use secure token storage",
	"Azure Storage Account Key":     "1. Regenerate storage account key\n2. Update all applications using the key\n3. Use Azure Key Vault\n4. Implement managed identities",
	"Azure Connection String":       "1. Regenerate access keys\n2. Update connection strings\n3. Use Azure Key Vault references\n4. Implement managed identities",
	"Square Access Token":           "1. Revoke token in Square Developer Dashboard\n2. Generate new token\n3. Review application permissions\n4. Implement secure storage",
	"Telegram Bot Token":            "1. Revoke bot token via @BotFather\n2. Generate new token\n3. Store in environment variables\n4. Review bot activity logs",
}

// Scan scans content for secrets and returns findings
func (s *Scanner) Scan(content, filePath string) []*SecretFinding {
	var findings []*SecretFinding

	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		for secretType, pattern := range s.patterns {
			matches := pattern.FindAllStringSubmatchIndex(line, -1)
			for _, match := range matches {
				if len(match) >= 2 {
					start, end := match[0], match[1]
					// Handle capture groups
					if len(match) > 2 && match[2] != -1 {
						start, end = match[2], match[3]
					}

					value := line[start:end]
					// Truncate value for safety in logs
					truncatedValue := truncateSecret(value)

					finding := &SecretFinding{
						Type:        secretType,
						FilePath:    filePath,
						Line:        lineNum + 1, // 1-indexed
						Column:      start + 1,   // 1-indexed
						Value:       truncatedValue,
						Severity:    "Critical",
						Description: getDescription(secretType),
						Remediation: getRemediation(secretType),
					}
					findings = append(findings, finding)
				}
			}
		}
	}

	return findings
}

// ScanFiles scans multiple files for secrets
func (s *Scanner) ScanFiles(files map[string]string) []*SecretFinding {
	var allFindings []*SecretFinding

	for filePath, content := range files {
		findings := s.Scan(content, filePath)
		allFindings = append(allFindings, findings...)
	}

	return allFindings
}

// truncateSecret truncates a secret value for safe logging
func truncateSecret(secret string) string {
	if len(secret) <= 8 {
		return "***REDACTED***"
	}
	// Show first 4 and last 4 characters
	if len(secret) > 16 {
		return secret[:4] + "***REDACTED***" + secret[len(secret)-4:]
	}
	return secret[:2] + "***REDACTED***" + secret[len(secret)-2:]
}

// getDescription returns a description for the secret type
func getDescription(secretType string) string {
	descriptions := map[string]string{
		"AWS Access Key ID":            "AWS access key ID detected in code. This credential can be used to access AWS resources.",
		"AWS Secret Access Key":        "AWS secret access key detected. This is a highly sensitive credential that grants AWS access.",
		"AWS Account ID":               "AWS account ID detected. While not directly exploitable, it aids targeted attacks.",
		"GitHub Personal Access Token": "GitHub personal access token detected. This token can access GitHub repositories and resources.",
		"GitHub OAuth Access Token":    "GitHub OAuth access token detected. This token grants access to GitHub user data.",
		"Private Key":                  "Private cryptographic key detected. This key can decrypt sensitive communications or authenticate to services.",
		"Database Connection String":   "Database connection string with credentials detected. This can provide direct database access.",
		"Generic Secret":               "Potential secret or password detected in code. This may grant unauthorized access to services.",
		"JWT Token":                    "JSON Web Token detected. This token may provide authenticated access to services.",
		"Password in URL":              "Password embedded in URL detected. Credentials in URLs are easily exposed.",
	}

	if desc, ok := descriptions[secretType]; ok {
		return desc
	}
	return "Sensitive credential or secret detected in code. This may grant unauthorized access to services."
}

// getRemediation returns remediation guidance for the secret type
func getRemediation(secretType string) string {
	if guidance, ok := remediationGuidance[secretType]; ok {
		return guidance
	}
	return "1. Remove the secret from code immediately\n2. Rotate the compromised credential\n3. Use environment variables or secret managers\n4. Review access logs for unauthorized use"
}
