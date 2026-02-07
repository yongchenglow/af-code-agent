package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
)

// Server wraps the webhook HTTP handler
type Server struct {
	agent         *agent.Agent
	webhookSecret string
}

// NewServer creates a new webhook server
func NewServer(app *agent.Agent, webhookSecret string) *Server {
	return &Server{
		agent:         app,
		webhookSecret: webhookSecret,
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

	// Add agent to context so it can be retrieved by reasoners
	ctx := r.Context()
	ctx = withAgent(ctx, s.agent)
	result, err := s.agent.CallLocal(ctx, "handle_webhook", input)
	if err != nil {
		log.Printf("Webhook processing failed: %v", err)
		http.Error(w, fmt.Sprintf("Webhook processing failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)

	log.Printf("Webhook processed successfully: %s", eventType)
}

// RegisterWebhookHandler registers the webhook HTTP handler with a standard HTTP server
func RegisterWebhookHandler(mux *http.ServeMux, app *agent.Agent, webhookSecret string) {
	server := NewServer(app, webhookSecret)
	mux.Handle("/webhook", server)
	log.Println("Webhook handler registered at /webhook")
}

// withAgent adds the agent instance to the context
func withAgent(ctx context.Context, agent *agent.Agent) context.Context {
	return context.WithValue(ctx, "agent", agent)
}
