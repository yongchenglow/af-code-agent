package github

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/yourorg/github-code-agent/pkg/circuitbreaker"
	"github.com/yourorg/github-code-agent/pkg/retry"
	"golang.org/x/oauth2"
)

// Client wraps the GitHub API client
type Client struct {
	client         *github.Client
	token          string
	appAuth        *AppAuth
	circuitBreaker *circuitbreaker.CircuitBreaker
	retryer        *retry.Retryer
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	Threshold           int
	Timeout             time.Duration
	HalfOpenMaxRequests int
}

// RetryerConfig holds retry configuration
type RetryerConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// DefaultCircuitBreakerConfig returns the default circuit breaker configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Threshold:           10,
		Timeout:             2 * time.Minute,
		HalfOpenMaxRequests: 5,
	}
}

// DefaultRetryerConfig returns the default retry configuration for GitHub API
func DefaultRetryerConfig() RetryerConfig {
	return RetryerConfig{
		MaxAttempts:  5,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     60 * time.Second,
		Multiplier:   2.0,
	}
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

	cbConfig := DefaultCircuitBreakerConfig()
	retryConfig := DefaultRetryerConfig()

	return &Client{
		client: github.NewClient(tc),
		token:  token,
		circuitBreaker: circuitbreaker.New(circuitbreaker.Config{
			Threshold:           cbConfig.Threshold,
			Timeout:             cbConfig.Timeout,
			HalfOpenMaxRequests: cbConfig.HalfOpenMaxRequests,
			Name:                "github-api",
		}),
		retryer: retry.New(retry.Config{
			MaxAttempts:  retryConfig.MaxAttempts,
			InitialDelay: retryConfig.InitialDelay,
			MaxDelay:     retryConfig.MaxDelay,
			Multiplier:   retryConfig.Multiplier,
			Jitter:       0.1,
			Retryable:    retry.TransientRetryable,
		}),
	}, nil
}

// NewClientWithAppCredentials creates a GitHub client configured for App authentication
// The actual token will be fetched when needed based on the repository context
func NewClientWithAppCredentials(appID, privateKey string) (*Client, error) {
	if appID == "" {
		return nil, fmt.Errorf("GitHub App ID is required")
	}
	if privateKey == "" {
		return nil, fmt.Errorf("GitHub private key is required")
	}

	auth, err := NewAppAuth(appID, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create app auth: %w", err)
	}

	cbConfig := DefaultCircuitBreakerConfig()
	retryConfig := DefaultRetryerConfig()

	// Create a basic client without authentication for now
	// The client will be recreated with proper auth when GetClientForRepo is called
	return &Client{
		client:  github.NewClient(nil),
		appAuth: auth,
		circuitBreaker: circuitbreaker.New(circuitbreaker.Config{
			Threshold:           cbConfig.Threshold,
			Timeout:             cbConfig.Timeout,
			HalfOpenMaxRequests: cbConfig.HalfOpenMaxRequests,
			Name:                "github-api",
		}),
		retryer: retry.New(retry.Config{
			MaxAttempts:  retryConfig.MaxAttempts,
			InitialDelay: retryConfig.InitialDelay,
			MaxDelay:     retryConfig.MaxDelay,
			Multiplier:   retryConfig.Multiplier,
			Jitter:       0.1,
			Retryable:    retry.TransientRetryable,
		}),
	}, nil
}

// GetClientForRepo returns a GitHub client authenticated for a specific repository
// This is used when using GitHub App authentication
func (c *Client) GetClientForRepo(ctx context.Context, owner, repo string) (*github.Client, error) {
	if c.appAuth == nil {
		// Not using app auth, return the existing client
		return c.client, nil
	}

	// Get installation token for this specific repo
	token, err := c.appAuth.GetInstallationToken(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get installation token: %w", err)
	}

	// Create a new client with the installation token
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc), nil
}

// GetClient returns the underlying GitHub client
// Note: When using App authentication, this returns a client without repo-specific auth.
// Use GetClientForRepo for authenticated operations with App authentication.
func (c *Client) GetClient() *github.Client {
	return c.client
}

// IsAppAuth returns true if this client uses GitHub App authentication
func (c *Client) IsAppAuth() bool {
	return c.appAuth != nil
}

// AuthenticateForRepo updates the client's internal GitHub client with repo-specific authentication
// This is a convenience method for GitHub App authentication
func (c *Client) AuthenticateForRepo(ctx context.Context, owner, repo string) error {
	if c.appAuth == nil {
		// Not using app auth, no need to update
		return nil
	}

	// Get installation token for this specific repo
	token, err := c.appAuth.GetInstallationToken(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to get installation token: %w", err)
	}

	// Update the client with the installation token
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	c.client = github.NewClient(tc)
	c.token = token

	return nil
}

// executeWithRetryAndCircuitBreaker executes a GitHub API call with retry and circuit breaker protection
func (c *Client) executeWithRetryAndCircuitBreaker(ctx context.Context, operation func() error) error {
	if c.retryer == nil || c.circuitBreaker == nil {
		// No protection configured, execute directly
		return operation()
	}

	return c.retryer.Execute(ctx, func() error {
		return c.circuitBreaker.Execute(operation)
	})
}

// GetCircuitBreaker returns the circuit breaker for monitoring
func (c *Client) GetCircuitBreaker() *circuitbreaker.CircuitBreaker {
	return c.circuitBreaker
}

// GetRetryer returns the retryer for monitoring
func (c *Client) GetRetryer() *retry.Retryer {
	return c.retryer
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
func (c *Client) GetPRFiles(ctx context.Context, owner, repo string, prNumber int) ([]*FileChange, error) {
	var changes []*FileChange

	err := c.executeWithRetryAndCircuitBreaker(ctx, func() error {
		files, _, err := c.client.PullRequests.ListFiles(ctx, owner, repo, prNumber, nil)
		if err != nil {
			return fmt.Errorf("failed to list PR files: %w", err)
		}

		changes = make([]*FileChange, 0, len(files))
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
		return nil
	})

	return changes, err
}

// GetPR fetches pull request details
func (c *Client) GetPR(ctx context.Context, owner, repo string, prNumber int) (*github.PullRequest, error) {
	var pr *github.PullRequest

	err := c.executeWithRetryAndCircuitBreaker(ctx, func() error {
		var resp *github.PullRequest
		var httpResp *github.Response
		var err error
		resp, httpResp, err = c.client.PullRequests.Get(ctx, owner, repo, prNumber)
		if err != nil {
			return fmt.Errorf("failed to get PR: %w", err)
		}
		if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != 201 {
			return fmt.Errorf("unexpected status code: %d", httpResp.StatusCode)
		}
		pr = resp
		return nil
	})

	return pr, err
}

// CheckCIStatus checks the CI/CD status for a PR
func (c *Client) CheckCIStatus(ctx context.Context, owner, repo, ref string) (string, error) {
	var state string

	err := c.executeWithRetryAndCircuitBreaker(ctx, func() error {
		status, _, err := c.client.Repositories.GetCombinedStatus(ctx, owner, repo, ref, nil)
		if err != nil {
			return fmt.Errorf("failed to get CI status: %w", err)
		}
		state = status.GetState()
		return nil
	})

	return state, err
}

// GetFileContent fetches the content of a file from a specific commit/branch
func (c *Client) GetFileContent(ctx context.Context, owner, repo, path, ref string) (string, error) {
	var content string

	err := c.executeWithRetryAndCircuitBreaker(ctx, func() error {
		fileContent, _, _, err := c.client.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{
			Ref: ref,
		})
		if err != nil {
			return fmt.Errorf("failed to get file content for %s: %w", path, err)
		}

		if fileContent == nil {
			return fmt.Errorf("file not found: %s", path)
		}

		var decodeErr error
		content, decodeErr = fileContent.GetContent()
		if decodeErr != nil {
			return fmt.Errorf("failed to decode file content: %w", decodeErr)
		}
		return nil
	})

	return content, err
}

// GetPRFilesWithClient is a convenience function that fetches PR files using a standalone client.
//
// Deprecated: Use client.GetPRFiles() instead
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

// GetPRWithClient is a convenience function that fetches PR details using a standalone client.
//
// Deprecated: Use client.GetPR() instead
func GetPR(ctx context.Context, client *github.Client, owner, repo string, prNumber int) (*github.PullRequest, error) {
	pr, _, err := client.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR: %w", err)
	}
	return pr, nil
}

// CheckCIStatusWithClient is a convenience function that checks CI status using a standalone client.
//
// Deprecated: Use client.CheckCIStatus() instead
func CheckCIStatus(ctx context.Context, client *github.Client, owner, repo, ref string) (string, error) {
	status, _, err := client.Repositories.GetCombinedStatus(ctx, owner, repo, ref, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get CI status: %w", err)
	}
	return status.GetState(), nil
}

// GetFileContentWithClient is a convenience function that fetches file content using a standalone client.
//
// Deprecated: Use client.GetFileContent() instead
func GetFileContent(ctx context.Context, client *github.Client, owner, repo, path, ref string) (string, error) {
	fileContent, _, _, err := client.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: ref,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get file content for %s: %w", path, err)
	}

	if fileContent == nil {
		return "", fmt.Errorf("file not found: %s", path)
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return "", fmt.Errorf("failed to decode file content: %w", err)
	}

	return content, nil
}

// DetectLanguage detects the programming language from file extension
func DetectLanguage(filename string) string {
	// Simple extension-based detection
	ext := ""
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			ext = filename[i+1:]
			break
		}
	}

	switch ext {
	case "go":
		return "go"
	case "py":
		return "python"
	case "js":
		return "javascript"
	case "ts":
		return "typescript"
	case "java":
		return "java"
	case "rb":
		return "ruby"
	case "rs":
		return "rust"
	case "cpp", "cc", "cxx":
		return "cpp"
	case "c":
		return "c"
	case "cs":
		return "csharp"
	case "php":
		return "php"
	default:
		return "text"
	}
}
