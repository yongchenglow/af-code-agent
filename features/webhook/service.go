package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	ghpkg "github.com/yourorg/github-code-agent/pkg/github"
)

// Service handles webhook business logic
type Service struct {
	agent         *agent.Agent
	ghClient      *ghpkg.Client
	webhookSecret string
	debouncer     *Debouncer
}

// NewService creates a new webhook service
func NewService(a *agent.Agent, webhookSecret string, ghClient *ghpkg.Client) *Service {
	return &Service{
		agent:         a,
		ghClient:      ghClient,
		webhookSecret: webhookSecret,
		debouncer:     NewDebouncer(),
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
	_, err := s.agent.CallLocal(ctx, "handle_webhook", input)
	if err != nil {
		return fmt.Errorf("reasoner failed: %w", err)
	}

	log.Printf("Webhook processed successfully: %s", eventType)
	return nil
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
