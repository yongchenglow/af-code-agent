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
	service *Service
}

// NewServer creates a new webhook server
func NewServer(app *agent.Agent, webhookSecret string, ghClient *ghpkg.Client) *Server {
	return &Server{
		service: NewService(app, webhookSecret, ghClient),
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

	// Get event type from header
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		log.Printf("Missing X-GitHub-Event header")
		http.Error(w, "Missing event type", http.StatusBadRequest)
		return
	}

	// Process webhook asynchronously (fire-and-forget)
	// Respond immediately to GitHub to avoid timeout
	go func() {
		bgCtx := context.Background()
		if err := s.service.ProcessWebhook(bgCtx, eventType, body, signature); err != nil {
			log.Printf("Webhook processing failed: %v", err)
		}
	}()

	// Return success response immediately
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted) // 202 Accepted for async processing
	_ = json.NewEncoder(w).Encode(map[string]string{
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
