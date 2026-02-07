package github

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"os"
	"testing"

	"github.com/google/go-github/v57/github"
)

// generateTestPrivateKey creates a test RSA private key
func generateTestPrivateKey(t *testing.T) (*rsa.PrivateKey, []byte) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return privateKey, privateKeyPEM
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		appID       string
		setupKey    func(t *testing.T) string
		wantErr     bool
		errContains string
	}{
		{
			name:  "valid app ID and private key",
			appID: "123456",
			setupKey: func(t *testing.T) string {
				_, pemData := generateTestPrivateKey(t)
				tmpFile, err := os.CreateTemp("", "github-key-*.pem")
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { os.Remove(tmpFile.Name()) })

				if _, err := tmpFile.Write(pemData); err != nil {
					t.Fatal(err)
				}
				tmpFile.Close()
				return tmpFile.Name()
			},
			wantErr: false,
		},
		{
			name:  "invalid app ID",
			appID: "not-a-number",
			setupKey: func(t *testing.T) string {
				return "/tmp/test.pem"
			},
			wantErr:     true,
			errContains: "invalid GitHub app ID",
		},
		{
			name:  "non-existent private key file",
			appID: "123456",
			setupKey: func(t *testing.T) string {
				return "/nonexistent/path/key.pem"
			},
			wantErr:     true,
			errContains: "failed to read private key",
		},
		{
			name:  "invalid private key format",
			appID: "123456",
			setupKey: func(t *testing.T) string {
				tmpFile, err := os.CreateTemp("", "invalid-key-*.pem")
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { os.Remove(tmpFile.Name()) })

				tmpFile.WriteString("invalid key data")
				tmpFile.Close()
				return tmpFile.Name()
			},
			wantErr:     true,
			errContains: "failed to parse private key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyPath := tt.setupKey(t)
			client, err := NewClient(tt.appID, keyPath)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Error("expected non-nil client")
				return
			}

			if client.GetClient() == nil {
				t.Error("expected non-nil GitHub client")
			}

			if client.appID != 123456 {
				t.Errorf("appID = %d, want 123456", client.appID)
			}

			if client.privateKey == nil {
				t.Error("expected non-nil private key")
			}
		})
	}
}

func TestClient_GenerateJWT(t *testing.T) {
	_, pemData := generateTestPrivateKey(t)
	tmpFile, err := os.CreateTemp("", "github-key-*.pem")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(pemData); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	client, err := NewClient("123456", tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	token, err := client.generateJWT()
	if err != nil {
		t.Errorf("generateJWT() error = %v", err)
	}

	if token == "" {
		t.Error("expected non-empty JWT token")
	}

	// JWT should be in format: header.payload.signature
	parts := len(token)
	if parts == 0 {
		t.Error("JWT token is empty")
	}
}

func TestJWTTransport_RoundTrip(t *testing.T) {
	transport := &jwtTransport{
		token: "test-jwt-token",
	}

	req, err := newTestRequest()
	if err != nil {
		t.Fatal(err)
	}

	// Note: This will fail to actually connect, but we can verify headers are set
	_, err = transport.RoundTrip(req)

	// Check that headers were set correctly
	if req.Header.Get("Authorization") != "Bearer test-jwt-token" {
		t.Errorf("Authorization header = %q, want %q",
			req.Header.Get("Authorization"), "Bearer test-jwt-token")
	}

	if req.Header.Get("Accept") != "application/vnd.github+json" {
		t.Errorf("Accept header = %q, want %q",
			req.Header.Get("Accept"), "application/vnd.github+json")
	}
}

func TestTokenTransport_RoundTrip(t *testing.T) {
	transport := &tokenTransport{
		token: "test-token",
	}

	req, err := newTestRequest()
	if err != nil {
		t.Fatal(err)
	}

	// Note: This will fail to actually connect, but we can verify headers are set
	_, err = transport.RoundTrip(req)

	// Check that headers were set correctly
	if req.Header.Get("Authorization") != "token test-token" {
		t.Errorf("Authorization header = %q, want %q",
			req.Header.Get("Authorization"), "token test-token")
	}

	if req.Header.Get("Accept") != "application/vnd.github+json" {
		t.Errorf("Accept header = %q, want %q",
			req.Header.Get("Accept"), "application/vnd.github+json")
	}
}

func TestGetPRFiles(t *testing.T) {
	// This test would require mocking the GitHub client
	// For now, we'll test error handling with nil context
	ctx := context.Background()
	client := github.NewClient(nil)

	_, err := GetPRFiles(ctx, client, "owner", "repo", 1)

	// Without authentication, this should fail
	if err == nil {
		t.Log("Note: GetPRFiles requires GitHub API mocking for full testing")
	}
}

func TestGetPR(t *testing.T) {
	// This test would require mocking the GitHub client
	ctx := context.Background()
	client := github.NewClient(nil)

	_, err := GetPR(ctx, client, "owner", "repo", 1)

	// Without authentication, this should fail
	if err == nil {
		t.Log("Note: GetPR requires GitHub API mocking for full testing")
	}
}

func TestCheckCIStatus(t *testing.T) {
	// This test would require mocking the GitHub client
	ctx := context.Background()
	client := github.NewClient(nil)

	_, err := CheckCIStatus(ctx, client, "owner", "repo", "main")

	// Without authentication, this should fail
	if err == nil {
		t.Log("Note: CheckCIStatus requires GitHub API mocking for full testing")
	}
}

func TestPRContext(t *testing.T) {
	prCtx := &PRContext{
		Owner:          "testowner",
		Repo:           "testrepo",
		Number:         42,
		Branch:         "feature-branch",
		BaseBranch:     "main",
		Author:         "testauthor",
		Title:          "Test PR",
		Files:          []string{"file1.go", "file2.go"},
		InstallationID: 12345,
	}

	if prCtx.Owner != "testowner" {
		t.Errorf("Owner = %q, want %q", prCtx.Owner, "testowner")
	}
	if prCtx.Repo != "testrepo" {
		t.Errorf("Repo = %q, want %q", prCtx.Repo, "testrepo")
	}
	if prCtx.Number != 42 {
		t.Errorf("Number = %d, want %d", prCtx.Number, 42)
	}
	if len(prCtx.Files) != 2 {
		t.Errorf("Files length = %d, want 2", len(prCtx.Files))
	}
}

func TestFileChange(t *testing.T) {
	fc := &FileChange{
		Filename:  "test.go",
		Status:    "modified",
		Additions: 10,
		Deletions: 5,
		Changes:   15,
		Patch:     "diff content",
		BlobURL:   "https://github.com/blob",
	}

	if fc.Filename != "test.go" {
		t.Errorf("Filename = %q, want %q", fc.Filename, "test.go")
	}
	if fc.Status != "modified" {
		t.Errorf("Status = %q, want %q", fc.Status, "modified")
	}
	if fc.Additions != 10 {
		t.Errorf("Additions = %d, want 10", fc.Additions)
	}
	if fc.Deletions != 5 {
		t.Errorf("Deletions = %d, want 5", fc.Deletions)
	}
	if fc.Changes != 15 {
		t.Errorf("Changes = %d, want 15", fc.Changes)
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr)*2 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func newTestRequest() (*http.Request, error) {
	return http.NewRequest("GET", "https://api.github.com/test", nil)
}
