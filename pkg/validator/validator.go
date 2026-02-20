package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/yourorg/github-code-agent/pkg/logger"
)

var (
	// ErrInvalidContentType is returned when the content type is not application/json
	ErrInvalidContentType = errors.New("invalid content type: must be application/json")
	// ErrPayloadTooLarge is returned when the payload exceeds the maximum allowed size
	ErrPayloadTooLarge = errors.New("payload too large: exceeds 10MB limit")
	// ErrInvalidEventType is returned when the GitHub event type is not recognized
	ErrInvalidEventType = errors.New("invalid or unsupported GitHub event type")
	// ErrInvalidJSON is returned when the payload is not valid JSON
	ErrInvalidJSON = errors.New("invalid JSON format")
	// ErrInvalidPRNumber is returned when the pull request number is invalid
	ErrInvalidPRNumber = errors.New("invalid pull request number")
	// ErrPathTraversal is returned when path traversal is detected
	ErrPathTraversal = errors.New("path traversal detected")
	// ErrInjectionDetected is returned when injection patterns are detected
	ErrInjectionDetected = errors.New("potential injection attack detected")
)

// Validator validates webhook requests
type Validator struct {
	maxPayloadSize int64
	allowedEvents  map[string]bool
	log            *logger.Logger
}

// ValidatorConfig configures the validator
type ValidatorConfig struct {
	// MaxPayloadSize is the maximum allowed payload size in bytes (default: 10MB)
	MaxPayloadSize int64
	// AllowedEvents is a list of allowed GitHub event types
	AllowedEvents []string
}

// NewValidator creates a new webhook validator
func NewValidator(config ValidatorConfig) *Validator {
	maxSize := config.MaxPayloadSize
	if maxSize == 0 {
		maxSize = 10 * 1024 * 1024 // 10MB default
	}

	allowedEvents := make(map[string]bool)
	for _, event := range config.AllowedEvents {
		allowedEvents[event] = true
	}

	// If no events specified, allow common events
	if len(allowedEvents) == 0 {
		allowedEvents["pull_request"] = true
		allowedEvents["push"] = true
		allowedEvents["issue_comment"] = true
		allowedEvents["issues"] = true
	}

	return &Validator{
		maxPayloadSize: maxSize,
		allowedEvents:  allowedEvents,
		log:            logger.Default(),
	}
}

// ValidateRequest performs comprehensive validation of a webhook request
func (v *Validator) ValidateRequest(r *http.Request, body []byte, eventType string) error {
	// Validate content type
	if err := v.validateContentType(r); err != nil {
		return err
	}

	// Validate payload size
	if err := v.validatePayloadSize(body); err != nil {
		return err
	}

	// Validate event type
	if err := v.validateEventType(eventType); err != nil {
		return err
	}

	// Validate JSON structure
	if err := v.validateJSON(body); err != nil {
		return err
	}

	// Validate for injection attacks
	if err := v.validateInjection(body); err != nil {
		return err
	}

	return nil
}

// validateContentType checks if the content type is application/json
func (v *Validator) validateContentType(r *http.Request) error {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return fmt.Errorf("%w: missing Content-Type header", ErrInvalidContentType)
	}

	// Allow application/json with optional charset
	if !strings.HasPrefix(contentType, "application/json") {
		return fmt.Errorf("%w: got %s", ErrInvalidContentType, contentType)
	}

	return nil
}

// validatePayloadSize checks if the payload size is within limits
func (v *Validator) validatePayloadSize(body []byte) error {
	if int64(len(body)) > v.maxPayloadSize {
		return fmt.Errorf("%w: size %d bytes", ErrPayloadTooLarge, len(body))
	}
	return nil
}

// validateEventType checks if the event type is allowed
func (v *Validator) validateEventType(eventType string) error {
	if eventType == "" {
		return fmt.Errorf("%w: missing event type", ErrInvalidEventType)
	}

	if !v.allowedEvents[eventType] {
		return fmt.Errorf("%w: %s", ErrInvalidEventType, eventType)
	}

	return nil
}

// validateJSON checks if the body is valid JSON
func (v *Validator) validateJSON(body []byte) error {
	if !json.Valid(body) {
		return ErrInvalidJSON
	}
	return nil
}

// validateInjection checks for potential injection attacks
func (v *Validator) validateInjection(body []byte) error {
	bodyStr := string(body)

	// Check for path traversal
	if strings.Contains(bodyStr, "../") || strings.Contains(bodyStr, "..\\") {
		// Allow in comments/strings but flag suspicious patterns
		if matched, _ := regexp.MatchString(`["'][^"']*\.\./[^"']*["']`, bodyStr); !matched {
			// Only flag if not in a string context
			v.log.Debug("Potential path traversal detected", "payload_preview", truncateString(bodyStr, 100))
		}
	}

	// Check for SQL injection patterns
	sqlInjectionPatterns := []string{
		`(?i)'\s*OR\s+'1'\s*=\s*'1`,
		`(?i)'\s*OR\s+1\s*=\s*1`,
		`(?i)--\s*$`,
		`(?i);\s*DROP\s+TABLE`,
		`(?i);\s*DELETE\s+FROM`,
	}

	for _, pattern := range sqlInjectionPatterns {
		if matched, _ := regexp.MatchString(pattern, bodyStr); matched {
			v.log.Warn("Potential SQL injection pattern detected")
			return ErrInjectionDetected
		}
	}

	// Check for NoSQL injection patterns
	nosqlPatterns := []string{
		`\$\s*where`,
		`\$\s*ne`,
		`\$\s*gt`,
		`\$\s*lt`,
	}

	for _, pattern := range nosqlPatterns {
		if matched, _ := regexp.MatchString(pattern, bodyStr); matched {
			v.log.Warn("Potential NoSQL injection pattern detected")
			return ErrInjectionDetected
		}
	}

	return nil
}

// ValidatePRNumber validates a pull request number
func (v *Validator) ValidatePRNumber(prNumber int) error {
	if prNumber <= 0 {
		return fmt.Errorf("%w: must be positive, got %d", ErrInvalidPRNumber, prNumber)
	}
	return nil
}

// SanitizeFilePath sanitizes a file path to prevent path traversal
func (v *Validator) SanitizeFilePath(path string) (string, error) {
	// Check for path traversal attempts
	if strings.Contains(path, "../") || strings.Contains(path, "..\\") {
		return "", fmt.Errorf("%w: %s", ErrPathTraversal, path)
	}

	// Check for absolute paths
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
		return "", fmt.Errorf("%w: absolute paths not allowed: %s", ErrPathTraversal, path)
	}

	// Normalize path separators
	path = strings.ReplaceAll(path, "\\", "/")

	// Remove any null bytes
	path = strings.ReplaceAll(path, "\x00", "")

	return path, nil
}

// ValidateRepository validates repository information
func (v *Validator) ValidateRepository(owner, repo string) error {
	if owner == "" || repo == "" {
		return errors.New("owner and repository must not be empty")
	}

	// Validate owner name
	if !isValidName(owner) {
		return fmt.Errorf("invalid owner name: %s", owner)
	}

	// Validate repo name
	if !isValidName(repo) {
		return fmt.Errorf("invalid repository name: %s", repo)
	}

	return nil
}

// isValidName checks if a name is valid (alphanumeric, hyphens, underscores)
func isValidName(name string) bool {
	if len(name) == 0 || len(name) > 100 {
		return false
	}
	for _, c := range name {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '-' && c != '_' && c != '.' {
			return false
		}
	}
	return true
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// RateLimiter implements per-repository rate limiting
type RateLimiter struct {
	requests   map[string][]time.Time
	maxReqs    int
	window     time.Duration
	mu         chan struct{}
	log        *logger.Logger
	cleanupDur time.Duration
}

// RateLimiterConfig configures the rate limiter
type RateLimiterConfig struct {
	// MaxRequests is the maximum number of requests allowed per window
	MaxRequests int
	// Window is the time window for rate limiting
	Window time.Duration
	// CleanupInterval is how often to clean up old entries
	CleanupInterval time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	maxReqs := config.MaxRequests
	if maxReqs == 0 {
		maxReqs = 10 // default: 10 requests per window
	}

	window := config.Window
	if window == 0 {
		window = time.Minute // default: 1 minute window
	}

	cleanupDur := config.CleanupInterval
	if cleanupDur == 0 {
		cleanupDur = 5 * time.Minute
	}

	rl := &RateLimiter{
		requests:   make(map[string][]time.Time),
		maxReqs:    maxReqs,
		window:     window,
		mu:         make(chan struct{}, 1),
		log:        logger.Default(),
		cleanupDur: cleanupDur,
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// Allow checks if a request from the given repository is allowed
func (rl *RateLimiter) Allow(repo string) bool {
	rl.mu <- struct{}{}        // Acquire lock
	defer func() { <-rl.mu }() // Release lock

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Get existing requests
	requests := rl.requests[repo]

	// Filter to only requests within the window
	validRequests := make([]time.Time, 0, len(requests))
	for _, t := range requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}

	// Check if rate limit exceeded
	if len(validRequests) >= rl.maxReqs {
		rl.requests[repo] = validRequests
		return false
	}

	// Add current request
	validRequests = append(validRequests, now)
	rl.requests[repo] = validRequests

	return true
}

// GetRemaining returns the number of remaining requests for a repository
func (rl *RateLimiter) GetRemaining(repo string) int {
	rl.mu <- struct{}{}
	defer func() { <-rl.mu }()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	requests := rl.requests[repo]
	validCount := 0
	for _, t := range requests {
		if t.After(windowStart) {
			validCount++
		}
	}

	remaining := rl.maxReqs - validCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetResetTime returns when the rate limit will reset
func (rl *RateLimiter) GetResetTime(repo string) time.Time {
	rl.mu <- struct{}{}
	defer func() { <-rl.mu }()

	requests := rl.requests[repo]
	if len(requests) == 0 {
		return time.Now()
	}

	// Find the oldest request in the window
	oldest := requests[0]
	for _, t := range requests {
		if t.Before(oldest) {
			oldest = t
		}
	}

	return oldest.Add(rl.window)
}

// cleanupLoop periodically cleans up old entries
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupDur)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu <- struct{}{}
		now := time.Now()
		windowStart := now.Add(-rl.window)

		for repo, requests := range rl.requests {
			// Filter to only recent requests
			validRequests := make([]time.Time, 0, len(requests))
			for _, t := range requests {
				if t.After(windowStart) {
					validRequests = append(validRequests, t)
				}
			}

			if len(validRequests) == 0 {
				delete(rl.requests, repo)
			} else {
				rl.requests[repo] = validRequests
			}
		}

		rl.log.Debug("Rate limiter cleanup completed", "active_repos", len(rl.requests))
		<-rl.mu
	}
}
