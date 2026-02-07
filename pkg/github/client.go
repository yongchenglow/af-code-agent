package github

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v57/github"
)

// Client wraps the GitHub API client
type Client struct {
	client *github.Client
	appID  int64
	privateKey *rsa.PrivateKey
}

// NewClient creates a new GitHub client
func NewClient(appID string, privateKeyPath string) (*Client, error) {
	id, err := strconv.ParseInt(appID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid GitHub app ID: %w", err)
	}

	// Read private key
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	// Parse private key
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return &Client{
		client:     github.NewClient(nil),
		appID:      id,
		privateKey: privateKey,
	}, nil
}

// GetInstallationClient creates a client authenticated as an installation
func (c *Client) GetInstallationClient(installationID int64) (*github.Client, error) {
	// Generate JWT for app authentication
	token, err := c.generateJWT()
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	// Create temporary client with JWT
	httpClient := &http.Client{
		Transport: &jwtTransport{
			token: token,
		},
	}
	tempClient := github.NewClient(httpClient)

	// Get installation token
	installToken, _, err := tempClient.Apps.CreateInstallationToken(
		context.Background(),
		installationID,
		&github.InstallationTokenOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create installation token: %w", err)
	}

	// Create client with installation token
	httpClient = &http.Client{
		Transport: &tokenTransport{
			token: installToken.GetToken(),
		},
	}

	return github.NewClient(httpClient), nil
}

// generateJWT generates a JWT for GitHub App authentication
func (c *Client) generateJWT() (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
		Issuer:    strconv.FormatInt(c.appID, 10),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(c.privateKey)
}

// jwtTransport adds JWT authentication to requests
type jwtTransport struct {
	token string
}

func (t *jwtTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	return http.DefaultTransport.RoundTrip(req)
}

// tokenTransport adds token authentication to requests
type tokenTransport struct {
	token string
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "token "+t.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	return http.DefaultTransport.RoundTrip(req)
}

// PRContext contains pull request metadata
type PRContext struct {
	Owner        string
	Repo         string
	Number       int
	Branch       string
	BaseBranch   string
	Author       string
	Title        string
	Files        []string
	InstallationID int64
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
