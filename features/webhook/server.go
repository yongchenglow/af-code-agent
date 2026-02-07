package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	ghpkg "github.com/yourorg/github-code-agent/pkg/github"
)

// Server wraps the webhook HTTP handler
type Server struct {
	agent         *agent.Agent
	webhookSecret string
	ghClient      *ghpkg.Client
}

// NewServer creates a new webhook server
func NewServer(app *agent.Agent, webhookSecret string, ghClient *ghpkg.Client) *Server {
	return &Server{
		agent:         app,
		webhookSecret: webhookSecret,
		ghClient:      ghClient,
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
	defer r.Body.Close()

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

	// Validate signature
	if err := ValidateSignature(body, signature, s.webhookSecret); err != nil {
		log.Printf("Webhook signature validation failed: %v", err)
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Get event type from header
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		log.Printf("Missing X-GitHub-Event header")
		http.Error(w, "Missing event type", http.StatusBadRequest)
		return
	}

	// Parse JSON payload
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("Failed to parse JSON payload: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Call the handle_webhook reasoner
	input := map[string]any{
		"event_type":     eventType,
		"payload":        payload,
		"payload_raw":    body,
		"signature":      signature,
		"webhook_secret": s.webhookSecret,
	}

	// Add agent and GitHub client to context so they can be retrieved by reasoners
	ctx := r.Context()
	ctx = withAgent(ctx, s.agent)
	ctx = context.WithValue(ctx, "github_client", s.ghClient)

	// Process webhook asynchronously (fire-and-forget)
	// Respond immediately to GitHub to avoid timeout
	go func() {
		// Use background context for async processing (not tied to HTTP request)
		bgCtx := context.Background()
		bgCtx = withAgent(bgCtx, s.agent)
		bgCtx = context.WithValue(bgCtx, "github_client", s.ghClient)

		_, err := s.agent.CallLocal(bgCtx, "handle_webhook", input)
		if err != nil {
			log.Printf("Webhook processing failed: %v", err)
		} else {
			log.Printf("Webhook processed successfully: %s", eventType)
		}
	}()

	// Return success response immediately
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted) // 202 Accepted for async processing
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "accepted",
		"message": fmt.Sprintf("Webhook %s received and processing started", eventType),
	})
}

// RegisterWebhookHandler registers the webhook HTTP handler with a standard HTTP server
func RegisterWebhookHandler(mux *http.ServeMux, app *agent.Agent, webhookSecret string, ghClient *ghpkg.Client) {
	server := NewServer(app, webhookSecret, ghClient)
	mux.Handle("/webhook", server)
	log.Println("Webhook handler registered at /webhook")
}

// withAgent adds the agent instance to the context
func withAgent(ctx context.Context, agent *agent.Agent) context.Context {
	return context.WithValue(ctx, "agent", agent)
}
