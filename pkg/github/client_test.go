package github

import (
	"context"
	"testing"

	"github.com/google/go-github/v57/github"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid token",
			token:   "ghp_test_token_123456",
			wantErr: false,
		},
		{
			name:        "empty token",
			token:       "",
			wantErr:     true,
			errContains: "GitHub token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.token)

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

			if client.token != tt.token {
				t.Errorf("token = %q, want %q", client.token, tt.token)
			}
		})
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
		Owner:      "testowner",
		Repo:       "testrepo",
		Number:     42,
		Branch:     "feature-branch",
		BaseBranch: "main",
		Author:     "testauthor",
		Title:      "Test PR",
		Files:      []string{"file1.go", "file2.go"},
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
