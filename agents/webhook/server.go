package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/yourorg/github-code-agent/pkg/config"
	ghpkg "github.com/yourorg/github-code-agent/pkg/github"
	"github.com/yourorg/github-code-agent/pkg/logger"
	"github.com/yourorg/github-code-agent/pkg/validator"
)

// Server wraps the webhook HTTP handler
type Server struct {
	service     *Service
	validator   *validator.Validator
	rateLimiter *validator.RateLimiter
	log         *logger.Logger
}

// NewServer creates a new webhook server
func NewServer(app *agent.Agent, webhookSecret string, ghClient *ghpkg.Client, cfg *config.Config) *Server {
	// Create validator with allowed events
	val := validator.NewValidator(validator.ValidatorConfig{
		MaxPayloadSize: 10 * 1024 * 1024, // 10MB
		AllowedEvents: []string{
			"pull_request",
			"push",
			"issue_comment",
			"issues",
			"pull_request_review",
			"pull_request_review_comment",
		},
	})

	// Create rate limiter: 10 requests per minute per repository
	rl := validator.NewRateLimiter(validator.RateLimiterConfig{
		MaxRequests:     10,
		Window:          time.Minute,
		CleanupInterval: 5 * time.Minute,
	})

	return &Server{
		service:     NewService(app, webhookSecret, ghClient, cfg),
		validator:   val,
		rateLimiter: rl,
		log:         logger.Default(),
	}
}

// ServeHTTP handles incoming GitHub webhook requests
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the raw body (needed for signature validation)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer func() {
		_ = r.Body.Close()
	}()

	// Get event type from header
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		log.Printf("Missing X-GitHub-Event header")
		http.Error(w, "Missing event type", http.StatusBadRequest)
		return
	}

	// Get signature from headers
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		// Fallback to older signature method
		signature = r.Header.Get("X-Hub-Signature")
	}
	if signature == "" {
		log.Printf("Missing webhook signature")
		http.Error(w, "Missing signature", http.StatusUnauthorized)
		return
	}

	// Step 1: Validate the request (content type, payload size, event type, JSON, injection)
	if err := s.validator.ValidateRequest(r, body, eventType); err != nil {
		s.log.Warn("Webhook validation failed",
			"event_type", eventType,
			"error", err.Error())
		http.Error(w, fmt.Sprintf("Validation failed: %v", err), http.StatusBadRequest)
		return
	}

	// Step 2: Extract repository information for rate limiting
	repo := extractRepositoryFromBody(body)
	if repo != "" {
		// Check rate limit
		if !s.rateLimiter.Allow(repo) {
			remaining := s.rateLimiter.GetRemaining(repo)
			resetTime := s.rateLimiter.GetResetTime(repo)

			s.log.Warn("Rate limit exceeded",
				"repository", repo,
				"remaining", remaining,
				"reset_at", resetTime.Format(time.RFC3339))

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
			w.Header().Set("Retry-After", strconv.FormatInt(int64(time.Until(resetTime).Seconds()), 10))
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":       "rate_limit_exceeded",
				"message":     "Too many requests from this repository",
				"remaining":   remaining,
				"reset_at":    resetTime.Format(time.RFC3339),
				"retry_after": int64(time.Until(resetTime).Seconds()),
			})
			return
		}
	}

	// Step 3: Process webhook asynchronously (fire-and-forget)
	// Respond immediately to GitHub to avoid timeout
	go func() {
		bgCtx := s.service.PrepareContext(context.Background())
		if err := s.service.ProcessWebhook(bgCtx, eventType, body, signature); err != nil {
			s.log.Error("Webhook processing failed",
				"event_type", eventType,
				"repository", repo,
				"error", err.Error())
		} else {
			s.log.Debug("Webhook processed successfully",
				"event_type", eventType,
				"repository", repo)
		}
	}()

	// Return success response immediately with rate limit headers
	w.Header().Set("Content-Type", "application/json")
	if repo != "" {
		remaining := s.rateLimiter.GetRemaining(repo)
		resetTime := s.rateLimiter.GetResetTime(repo)
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
	}
	w.WriteHeader(http.StatusAccepted) // 202 Accepted for async processing
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "accepted",
		"message": fmt.Sprintf("Webhook %s received and processing started", eventType),
	})
}

// extractRepositoryFromBody extracts the repository name from the webhook payload
func extractRepositoryFromBody(body []byte) string {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}

	// Try to get repository from different possible locations
	if repo, ok := payload["repository"].(map[string]interface{}); ok {
		if fullName, ok := repo["full_name"].(string); ok {
			return fullName
		}
	}

	// For pull_request events, check in the pull_request object
	if pr, ok := payload["pull_request"].(map[string]interface{}); ok {
		if repo, ok := pr["base"].(map[string]interface{}); ok {
			if repoObj, ok := repo["repo"].(map[string]interface{}); ok {
				if fullName, ok := repoObj["full_name"].(string); ok {
					return fullName
				}
			}
		}
	}

	return ""
}

// RegisterWebhookHandler registers the webhook HTTP handler with a standard HTTP server
func RegisterWebhookHandler(mux *http.ServeMux, app *agent.Agent, webhookSecret string, ghClient *ghpkg.Client, cfg *config.Config) {
	server := NewServer(app, webhookSecret, ghClient, cfg)
	mux.Handle("/webhook", server)
	log.Println("Webhook handler registered at /webhook")
}
