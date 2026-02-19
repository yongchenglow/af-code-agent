package github

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// AppAuth handles GitHub App authentication
type AppAuth struct {
	appID          int64
	privateKey     *rsa.PrivateKey
	installationID int64

	// Token cache with automatic refresh
	mu          sync.RWMutex
	accessToken string
	tokenExpiry time.Time
}

// NewAppAuth creates a new GitHub App authenticator
func NewAppAuth(appID, privateKeyPEM string) (*AppAuth, error) {
	id, err := strconv.ParseInt(appID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID: %w", err)
	}

	privateKey, err := parsePrivateKey(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return &AppAuth{
		appID:      id,
		privateKey: privateKey,
	}, nil
}

// parsePrivateKey parses a PEM-encoded RSA private key
func parsePrivateKey(pemData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		rsaKey, ok := parsedKey.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key")
		}
		return rsaKey, nil
	}

	return key, nil
}

// generateJWT creates a JWT token for GitHub App authentication
func (a *AppAuth) generateJWT() (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iat": now.Unix(),
		"exp": now.Add(10 * time.Minute).Unix(), // GitHub requires max 10 minutes
		"iss": a.appID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(a.privateKey)
}

// GetInstallationToken fetches a new installation access token
func (a *AppAuth) GetInstallationToken(ctx context.Context, owner, repo string) (string, error) {
	// Check if we have a cached token that's still valid
	a.mu.RLock()
	if a.accessToken != "" && time.Now().Before(a.tokenExpiry.Add(-5*time.Minute)) {
		token := a.accessToken
		a.mu.RUnlock()
		return token, nil
	}
	a.mu.RUnlock()

	// Generate JWT for app-level authentication
	jwtToken, err := a.generateJWT()
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	// Create a client authenticated with JWT
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: jwtToken})
	tc := oauth2.NewClient(ctx, ts)
	appClient := github.NewClient(tc)

	// Get the installation ID for this repository
	installation, _, err := appClient.Apps.FindRepositoryInstallation(ctx, owner, repo)
	if err != nil {
		return "", fmt.Errorf("failed to find installation: %w", err)
	}

	// Create installation access token
	installationToken, _, err := appClient.Apps.CreateInstallationToken(
		ctx,
		installation.GetID(),
		&github.InstallationTokenOptions{},
	)
	if err != nil {
		return "", fmt.Errorf("failed to create installation token: %w", err)
	}

	// Cache the token
	a.mu.Lock()
	a.installationID = installation.GetID()
	a.accessToken = installationToken.GetToken()
	a.tokenExpiry = installationToken.GetExpiresAt().Time
	a.mu.Unlock()

	return installationToken.GetToken(), nil
}

// NewClientWithApp creates a new GitHub client using GitHub App authentication
func NewClientWithApp(ctx context.Context, appID, privateKey, owner, repo string) (*Client, error) {
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

	// Get installation token
	token, err := auth.GetInstallationToken(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get installation token: %w", err)
	}

	// Create OAuth2 token source with automatic refresh
	ts := &appTokenSource{
		auth:  auth,
		ctx:   ctx,
		owner: owner,
		repo:  repo,
	}

	tc := oauth2.NewClient(ctx, ts)
	ghClient := github.NewClient(tc)

	return &Client{
		client: ghClient,
		token:  token,
	}, nil
}

// appTokenSource implements oauth2.TokenSource for automatic token refresh
type appTokenSource struct {
	auth  *AppAuth
	ctx   context.Context
	owner string
	repo  string
}

// Token returns a valid token, refreshing if necessary
func (s *appTokenSource) Token() (*oauth2.Token, error) {
	token, err := s.auth.GetInstallationToken(s.ctx, s.owner, s.repo)
	if err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken: token,
		TokenType:   "Bearer",
		Expiry:      s.auth.tokenExpiry,
	}, nil
}
