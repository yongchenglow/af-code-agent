package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/yourorg/github-code-agent/pkg/config"
	ghpkg "github.com/yourorg/github-code-agent/pkg/github"
)

// Service handles webhook business logic
type Service struct {
	agent         *agent.Agent
	ghClient      *ghpkg.Client
	webhookSecret string
	debouncer     *Debouncer
	config        *config.Config
}

// NewService creates a new webhook service
func NewService(a *agent.Agent, webhookSecret string, ghClient *ghpkg.Client, cfg *config.Config) *Service {
	return &Service{
		agent:         a,
		ghClient:      ghClient,
		webhookSecret: webhookSecret,
		debouncer:     NewDebouncer(),
		config:        cfg,
	}
}

// ProcessWebhook processes a webhook event
func (s *Service) ProcessWebhook(ctx context.Context, eventType string, payload []byte, signature string) error {
	// Validate signature
	if err := ValidateSignature(payload, signature, s.webhookSecret); err != nil {
		return fmt.Errorf("signature validation failed: %w", err)
	}

	// Parse payload
	var payloadMap map[string]any
	if err := json.Unmarshal(payload, &payloadMap); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	// Extract repository information for GitHub App authentication
	owner, repo, err := extractRepoInfo(payloadMap)
	if err != nil {
		log.Printf("Warning: Could not extract repo info: %v", err)
	} else {
		// Check if repository is allowed
		if !s.isRepositoryAllowed(owner, repo) {
			log.Printf("Repository %s/%s is not in allowed list, skipping", owner, repo)
			return nil
		}

		// Authenticate for GitHub App
		if s.ghClient.IsAppAuth() {
			if err := s.ghClient.AuthenticateForRepo(ctx, owner, repo); err != nil {
				return fmt.Errorf("failed to authenticate for repo %s/%s: %w", owner, repo, err)
			}
			log.Printf("Authenticated GitHub client for %s/%s", owner, repo)
		}
	}

	// Check debounce for synchronize events
	if s.shouldDebounce(eventType, payloadMap) {
		log.Printf("Event %s debounced", eventType)
		return nil
	}

	// Prepare input for reasoner
	input := map[string]any{
		"event_type":     eventType,
		"payload":        payloadMap,
		"payload_raw":    payload,
		"signature":      signature,
		"webhook_secret": s.webhookSecret,
	}

	// Call reasoner
	_, err = s.agent.CallLocal(ctx, "handle_webhook", input)
	if err != nil {
		return fmt.Errorf("reasoner failed: %w", err)
	}

	log.Printf("Webhook processed successfully: %s", eventType)
	return nil
}

// extractRepoInfo extracts owner and repo name from webhook payload
func extractRepoInfo(payload map[string]any) (owner, repo string, err error) {
	// Try to get repository info from the payload
	repoData, ok := payload["repository"].(map[string]any)
	if !ok {
		return "", "", fmt.Errorf("repository field not found in payload")
	}

	// Extract owner
	ownerData, ok := repoData["owner"].(map[string]any)
	if !ok {
		return "", "", fmt.Errorf("owner field not found in repository")
	}

	owner, ok = ownerData["login"].(string)
	if !ok {
		return "", "", fmt.Errorf("owner login not found")
	}

	// Extract repo name
	repo, ok = repoData["name"].(string)
	if !ok {
		return "", "", fmt.Errorf("repository name not found")
	}

	return owner, repo, nil
}

// isRepositoryAllowed checks if a repository is allowed based on configuration
func (s *Service) isRepositoryAllowed(owner, repo string) bool {
	// If no filtering configured, allow all
	if s.config == nil {
		return true
	}

	allowedRepos := s.config.Agent.AllowedRepositories
	allowedPatterns := s.config.Agent.AllowedPatterns

	// If both are empty, allow all repositories
	if len(allowedRepos) == 0 && len(allowedPatterns) == 0 {
		return true
	}

	fullName := fmt.Sprintf("%s/%s", owner, repo)

	// Check exact matches
	for _, allowed := range allowedRepos {
		if allowed == fullName {
			return true
		}
	}

	// Check pattern matches
	for _, pattern := range allowedPatterns {
		if matchPattern(pattern, fullName) {
			return true
		}
	}

	return false
}

// matchPattern checks if a repository name matches a pattern
// Supports simple glob-style patterns like "myorg/*" or "*/backend-*"
func matchPattern(pattern, name string) bool {
	matched, err := filepath.Match(pattern, name)
	if err != nil {
		log.Printf("Warning: Invalid pattern %q: %v", pattern, err)
		return false
	}
	return matched
}

// shouldDebounce checks if the event should be debounced
func (s *Service) shouldDebounce(eventType string, payload map[string]any) bool {
	// Only debounce pull_request synchronize events
	if eventType != "pull_request" {
		return false
	}

	action, ok := payload["action"].(string)
	if !ok || action != "synchronize" {
		return false
	}

	// Extract PR number
	prData, ok := payload["pull_request"].(map[string]any)
	if !ok {
		return false
	}

	prNumber, ok := prData["number"].(float64)
	if !ok {
		return false
	}

	key := fmt.Sprintf("pr-%d", int(prNumber))
	return s.debouncer.ShouldDebounce(key, 30*time.Second)
}

// Debouncer handles event debouncing
type Debouncer struct {
	timestamps map[string]time.Time
}

// NewDebouncer creates a new debouncer
func NewDebouncer() *Debouncer {
	d := &Debouncer{
		timestamps: make(map[string]time.Time),
	}
	// Start cleanup goroutine
	go d.cleanup()
	return d
}

// ShouldDebounce checks if an event with the given key should be debounced
func (d *Debouncer) ShouldDebounce(key string, timeout time.Duration) bool {
	now := time.Now()

	// Check if we recently processed this key
	if lastTime, exists := d.timestamps[key]; exists {
		if now.Sub(lastTime) < timeout {
			// Update timestamp to extend debounce window
			d.timestamps[key] = now
			return true
		}
	}

	// Record this event time
	d.timestamps[key] = now
	return false
}

// cleanup removes old debounce entries
func (d *Debouncer) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().Add(-5 * time.Minute)
		for key, timestamp := range d.timestamps {
			if timestamp.Before(cutoff) {
				delete(d.timestamps, key)
			}
		}
	}
}
