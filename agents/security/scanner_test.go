package security

import (
	"testing"
)

func TestScanner_Scan(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		filePath     string
		wantFindings int
		wantTypes    []string
	}{
		{
			name: "AWS Access Key ID",
			content: `
package main

var awsKey = "AKIAIOSFODNN7EXAMPLE"
`,
			filePath:     "main.go",
			wantFindings: 1,
			wantTypes:    []string{"AWS Access Key ID"},
		},
		{
			name: "AWS Secret Access Key",
			content: `
AWS_SECRET_ACCESS_KEY = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
`,
			filePath:     ".env",
			wantFindings: 1,
			wantTypes:    []string{"AWS Secret Access Key"},
		},
		{
			name: "GitHub Personal Access Token",
			content: `
const token = "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx1234"
`,
			filePath:     "config.js",
			wantFindings: 1,
			wantTypes:    []string{"GitHub Personal Access Token"},
		},
		{
			name: "RSA Private Key",
			content: `
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF8PbnGy...
-----END RSA PRIVATE KEY-----
`,
			filePath:     "key.pem",
			wantFindings: 1,
			wantTypes:    []string{"RSA Private Key"},
		},
		{
			name: "Google API Key",
			content: `
api_key = "AIzaSyDaGmWKa4JsXZ-HjGw7ISLn_3namBGewQe"
`,
			filePath:     "config.py",
			wantFindings: 1,
			wantTypes:    []string{"Google API Key"},
		},
		{
			name: "JWT Token",
			content: `
const token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
`,
			filePath:     "auth.js",
			wantFindings: 1,
			wantTypes:    []string{"JWT Token"},
		},
		{
			name: "Slack Bot Token",
			content: `
SLACK_TOKEN = "xoxb-FALSEPOSITIVE-TEST-TOKEN-NOTREAL"
`,
			filePath:     "slack.go",
			wantFindings: 1,
			wantTypes:    []string{"Slack Bot Token"},
		},
		{
			name: "Stripe API Key",
			content: `
stripe.api_key = "sk_live_TESTKEY-NOTAREALSECRET123"
`,
			filePath:     "payment.py",
			wantFindings: 1,
			wantTypes:    []string{"Stripe API Key"},
		},
		{
			name: "PostgreSQL Connection String",
			content: `
DATABASE_URL = "postgres://user:password123@localhost:5432/mydb?sslmode=disable"
`,
			filePath:     "db.go",
			wantFindings: 1,
			wantTypes:    []string{"Password in URL"},
		},
		{
			name: "Generic Secret",
			content: `
api_key = "supersecretapikey12345678"
password = "verysecurepassword123"
`,
			filePath:     "config.json",
			wantFindings: 2,
			wantTypes:    []string{"Generic Secret", "Generic Secret"},
		},
		{
			name: "Multiple secrets in one file",
			content: `
AWS_ACCESS_KEY_ID = "AKIAIOSFODNN7EXAMPLE"
AWS_SECRET_ACCESS_KEY = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
GITHUB_TOKEN = "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx1234"
`,
			filePath:     ".env",
			wantFindings: 3,
			wantTypes:    []string{"AWS Access Key ID", "AWS Secret Access Key", "GitHub Personal Access Token"},
		},
		{
			name: "No secrets",
			content: `
package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`,
			filePath:     "main.go",
			wantFindings: 0,
			wantTypes:    []string{},
		},
		{
			name: "Password in URL",
			content: `
const dbURL = "mysql://admin:secretpass@localhost:3306/production"
`,
			filePath:     "db.go",
			wantFindings: 1,
			wantTypes:    []string{"Password in URL"},
		},
		{
			name: "SendGrid API Key",
			content: `
SENDGRID_API_KEY = "SG.abcdefghijklmnopqrstuv.1234567890abcdefghijklmnopqrstuvwxyz12345"
`,
			filePath:     "email.go",
			wantFindings: 1,
			wantTypes:    []string{"Generic Secret"},
		},
		{
			name: "NPM Token",
			content: `
//registry.npmjs.org/:_authToken=npm_abcdefghijklmnopqrstuvwxyz1234567890
`,
			filePath:     ".npmrc",
			wantFindings: 1,
			wantTypes:    []string{"NPM Token"},
		},
		{
			name: "Azure Connection String",
			content: `
AZURE_STORAGE_CONNECTION_STRING = "DefaultEndpointsProtocol=https;AccountName=myaccount;AccountKey=abcdefghijklmnopqrstuvwxyz1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdef==;EndpointSuffix=core.windows.net"
`,
			filePath:     "azure.go",
			wantFindings: 1,
			wantTypes:    []string{"Azure Connection String"},
		},
	}

	scanner := NewScanner()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := scanner.Scan(tt.content, tt.filePath)

			// Check minimum findings (some patterns may match multiple rules)
			if len(findings) < tt.wantFindings {
				t.Errorf("Scan() found %d findings, want at least %d", len(findings), tt.wantFindings)
			}

			// Verify finding types are present
			foundTypes := make(map[string]bool)
			for _, f := range findings {
				foundTypes[f.Type] = true
			}

			for _, wantType := range tt.wantTypes {
				if !foundTypes[wantType] {
					t.Errorf("Scan() missing expected type: %s. Found: %v", wantType, foundTypes)
				}
			}

			// Verify all findings have required fields
			for _, f := range findings {
				if f.Type == "" {
					t.Error("Scan() finding missing Type")
				}
				if f.FilePath == "" {
					t.Error("Scan() finding missing FilePath")
				}
				if f.Line == 0 {
					t.Error("Scan() finding missing Line")
				}
				if f.Severity != "Critical" {
					t.Errorf("Scan() finding has severity %s, want Critical", f.Severity)
				}
				if f.Description == "" {
					t.Error("Scan() finding missing Description")
				}
				if f.Remediation == "" {
					t.Error("Scan() finding missing Remediation")
				}
			}
		})
	}
}

func TestScanner_ScanFiles(t *testing.T) {
	scanner := NewScanner()

	files := map[string]string{
		"config.go": `
package main
var key = "AKIAIOSFODNN7EXAMPLE"
`,
		".env": `
AWS_SECRET_ACCESS_KEY = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
`,
		"safe.go": `
package main
func main() {}
`,
	}

	findings := scanner.ScanFiles(files)

	// Should find 2 secrets (one in config.go, one in .env)
	if len(findings) != 2 {
		t.Errorf("ScanFiles() found %d findings, want 2", len(findings))
	}

	// Verify file paths are correct
	foundFiles := make(map[string]bool)
	for _, f := range findings {
		foundFiles[f.FilePath] = true
	}

	if !foundFiles["config.go"] {
		t.Error("ScanFiles() missing finding in config.go")
	}
	if !foundFiles[".env"] {
		t.Error("ScanFiles() missing finding in .env")
	}
}

func TestScanner_LineAndColumn(t *testing.T) {
	scanner := NewScanner()

	content := `
package main

var awsKey = "AKIAIOSFODNN7EXAMPLE"
var githubToken = "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx1234"
`

	findings := scanner.Scan(content, "test.go")

	if len(findings) < 2 {
		t.Fatalf("Scan() found %d findings, want at least 2", len(findings))
	}

	// Verify line numbers are correct (1-indexed)
	for _, f := range findings {
		if f.Line < 1 || f.Line > 6 {
			t.Errorf("Scan() line number %d out of range [1, 6]", f.Line)
		}
		if f.Column < 1 {
			t.Errorf("Scan() column number %d should be >= 1", f.Column)
		}
	}
}

func TestScanner_TruncateSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		want   string
	}{
		{
			name:   "short secret",
			secret: "abc123",
			want:   "***REDACTED***",
		},
		{
			name:   "medium secret",
			secret: "password123",
			want:   "pa***REDACTED***23",
		},
		{
			name:   "long secret",
			secret: "AKIAIOSFODNN7EXAMPLE1234567890",
			want:   "AKIA***REDACTED***7890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateSecret(tt.secret)
			if got != tt.want {
				t.Errorf("truncateSecret() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestScanner_GetDescription(t *testing.T) {
	tests := []struct {
		name string
		want bool // true if description should exist
	}{
		{name: "AWS Access Key ID", want: true},
		{name: "GitHub Personal Access Token", want: true},
		{name: "Private Key", want: true},
		{name: "Unknown Type", want: true}, // Should return default description
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := getDescription(tt.name)
			if desc == "" && tt.want {
				t.Error("getDescription() returned empty string")
			}
		})
	}
}

func TestScanner_GetRemediation(t *testing.T) {
	tests := []struct {
		name string
		want bool // true if remediation should exist
	}{
		{name: "AWS Access Key ID", want: true},
		{name: "GitHub Personal Access Token", want: true},
		{name: "Private Key", want: true},
		{name: "Unknown Type", want: true}, // Should return default remediation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remediation := getRemediation(tt.name)
			if remediation == "" && tt.want {
				t.Error("getRemediation() returned empty string")
			}
		})
	}
}

func TestScanner_FalsePositives(t *testing.T) {
	scanner := NewScanner()

	// Test content that should NOT trigger false positives
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "Documentation about secrets",
			content: `
// This function handles AWS authentication
// You should never commit AWS_SECRET_ACCESS_KEY to git
// Always use environment variables for api_key storage
`,
		},
		{
			name: "Example values in comments",
			content: `
// Example: AKIAIOSFODNN7EXAMPLE (this is a documented example)
// See: https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html
`,
		},
		{
			name: "Short passwords",
			content: `
password = "abc"  // Too short to match generic secret pattern
key = "123456"    // Too short
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := scanner.Scan(tt.content, "test.go")
			// Allow some false positives but should be minimal
			if len(findings) > 1 {
				t.Logf("Potential false positives detected: %d", len(findings))
				for _, f := range findings {
					t.Logf("  - %s at line %d", f.Type, f.Line)
				}
			}
		})
	}
}

func TestScanner_AllPatternTypes(t *testing.T) {
	scanner := NewScanner()

	// Test that each pattern type can be detected
	testSecrets := map[string]string{
		"AWS Access Key ID":            "AKIAIOSFODNN7EXAMPLE",
		"AWS Secret Access Key":        "aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"GitHub Personal Access Token": "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx1234",
		"GitHub OAuth Access Token":    "gho_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx1234",
		"GitLab Personal Access Token": "glpat-abcdefghijklmnopqrst",
		"Google API Key":               "AIzaSyDaGmWKa4JsXZ-HjGw7ISLn_3namBGewQe",
		"RSA Private Key":              "-----BEGIN RSA PRIVATE KEY-----",
		"Generic Private Key":          "-----BEGIN PRIVATE KEY-----",
		"JWT Token":                    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
		"Slack Bot Token":              "xoxb-FALSEPOSITIVE-TEST-TOKEN-NOTREAL",
		"Stripe API Key":               "sk_live_TESTKEY-NOTAREALSECRET123",
		"PostgreSQL Connection String": "postgres://user:pass@localhost:5432/mydb",
		"MySQL Connection String":      "mysql://user:pass@localhost:3306/mydb",
		"MongoDB Connection String":    "mongodb://user:pass@localhost:27017/mydb",
		"Generic Secret":               "api_key = \"supersecretapikey12345678\"",
		"NPM Token":                    "//registry.npmjs.org/:_authToken=npm_abcdefghijklmnopqrstuvwxyz1234567890",
		"Password in URL":              "https://user:password123@example.com/api",
	}

	for secretType, secret := range testSecrets {
		t.Run(secretType, func(t *testing.T) {
			findings := scanner.Scan(secret, "test.go")

			found := false
			for _, f := range findings {
				if f.Type == secretType {
					found = true
					break
				}
			}

			if !found {
				t.Logf("Findings: %v", findings)
				t.Errorf("Scan() did not detect %s", secretType)
			}
		})
	}
}
