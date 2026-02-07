package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// Client wraps the GitHub API client
type Client struct {
	client *github.Client
	token  string
}

// NewClient creates a new GitHub client with token authentication
func NewClient(token string) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	// Create OAuth2 token source
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	return &Client{
		client: github.NewClient(tc),
		token:  token,
	}, nil
}

// GetClient returns the underlying GitHub client
func (c *Client) GetClient() *github.Client {
	return c.client
}

// PRContext contains pull request metadata
type PRContext struct {
	Owner      string
	Repo       string
	Number     int
	Branch     string
	BaseBranch string
	Author     string
	Title      string
	Files      []string
}

// FileChange represents a changed file in a PR
type FileChange struct {
	Filename  string
	Status    string // added, modified, deleted
	Additions int
	Deletions int
	Changes   int
	Patch     string
	BlobURL   string
}

// GetPRFiles fetches all changed files in a PR
func GetPRFiles(ctx context.Context, client *github.Client, owner, repo string, prNumber int) ([]*FileChange, error) {
	files, _, err := client.PullRequests.ListFiles(ctx, owner, repo, prNumber, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list PR files: %w", err)
	}

	changes := make([]*FileChange, 0, len(files))
	for _, file := range files {
		changes = append(changes, &FileChange{
			Filename:  file.GetFilename(),
			Status:    file.GetStatus(),
			Additions: file.GetAdditions(),
			Deletions: file.GetDeletions(),
			Changes:   file.GetChanges(),
			Patch:     file.GetPatch(),
			BlobURL:   file.GetBlobURL(),
		})
	}

	return changes, nil
}

// GetPR fetches pull request details
func GetPR(ctx context.Context, client *github.Client, owner, repo string, prNumber int) (*github.PullRequest, error) {
	pr, _, err := client.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR: %w", err)
	}
	return pr, nil
}

// CheckCIStatus checks the CI/CD status for a PR
func CheckCIStatus(ctx context.Context, client *github.Client, owner, repo, ref string) (string, error) {
	status, _, err := client.Repositories.GetCombinedStatus(ctx, owner, repo, ref, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get CI status: %w", err)
	}
	return status.GetState(), nil
}
