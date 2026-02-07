package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/yourorg/github-code-agent/pkg/config"
)

// Handler handles GitHub webhook events
type Handler struct {
	config        *config.Config
	webhookSecret string
	debounceMap   map[string]time.Time
}

// NewHandler creates a new webhook handler
func NewHandler(cfg *config.Config, webhookSecret string) *Handler {
	return &Handler{
		config:        cfg,
		webhookSecret: webhookSecret,
		debounceMap:   make(map[string]time.Time),
	}
}

// HandleWebhook processes incoming webhook events
func (h *Handler) HandleWebhook(ctx context.Context, eventType string, payload []byte, signature string) (*WorkflowResult, error) {
	// Validate webhook signature
	if err := ValidateSignature(payload, signature, h.webhookSecret); err != nil {
		return nil, fmt.Errorf("webhook validation failed: %w", err)
	}

	// Parse event
	event, err := h.parseEvent(eventType, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event: %w", err)
	}

	// Check if event should be processed
	if !ShouldProcessEvent(eventType, event.Action, h.config.Webhooks.Triggers) {
		return &WorkflowResult{
			Success: true,
			Message: fmt.Sprintf("Event %s.%s not configured for processing", eventType, event.Action),
		}, nil
	}

	// Check debounce for synchronize events
	if eventType == "pull_request" && event.Action == "synchronize" {
		if h.shouldDebounce(event) {
			return &WorkflowResult{
				Success: true,
				Message: "Event debounced, waiting for more commits",
			}, nil
		}
	}

	// Route to appropriate handler
	return h.routeEvent(ctx, event)
}

// parseEvent parses webhook payload into structured event
func (h *Handler) parseEvent(eventType string, payload []byte) (*WebhookEvent, error) {
	event := &WebhookEvent{
		Type: eventType,
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}

	// Extract action if present
	if action, ok := raw["action"].(string); ok {
		event.Action = action
	}

	// Extract installation
	if inst, ok := raw["installation"].(map[string]interface{}); ok {
		if id, ok := inst["id"].(float64); ok {
			event.Installation = &InstallationPayload{ID: int64(id)}
		}
	}

	// Parse based on event type
	switch eventType {
	case "pull_request":
		if pr, ok := raw["pull_request"].(map[string]interface{}); ok {
			prData, _ := json.Marshal(pr)
			event.PullRequest = &PullRequestPayload{}
			json.Unmarshal(prData, event.PullRequest)
		}

	case "check_suite":
		if cs, ok := raw["check_suite"].(map[string]interface{}); ok {
			csData, _ := json.Marshal(cs)
			event.CheckSuite = &CheckSuitePayload{}
			json.Unmarshal(csData, event.CheckSuite)
		}

	case "workflow_run":
		if wr, ok := raw["workflow_run"].(map[string]interface{}); ok {
			wrData, _ := json.Marshal(wr)
			event.WorkflowRun = &WorkflowRunPayload{}
			json.Unmarshal(wrData, event.WorkflowRun)
		}
	}

	return event, nil
}

// shouldDebounce checks if event should be debounced
func (h *Handler) shouldDebounce(event *WebhookEvent) bool {
	if event.PullRequest == nil {
		return false
	}

	debounceTimeout := h.config.Webhooks.GetDebounceTimeout()
	if debounceTimeout == 0 {
		return false
	}

	key := fmt.Sprintf("pr-%d", event.PullRequest.Number)

	// Check if we recently processed this PR
	if lastTime, exists := h.debounceMap[key]; exists {
		if time.Since(lastTime) < debounceTimeout {
			// Update the timestamp to extend debounce window
			h.debounceMap[key] = time.Now()
			return true
		}
	}

	// Record this event time
	h.debounceMap[key] = time.Now()

	// Clean up old entries (older than 5 minutes)
	h.cleanupDebounceMap()

	return false
}

// cleanupDebounceMap removes old debounce entries
func (h *Handler) cleanupDebounceMap() {
	cutoff := time.Now().Add(-5 * time.Minute)
	for key, timestamp := range h.debounceMap {
		if timestamp.Before(cutoff) {
			delete(h.debounceMap, key)
		}
	}
}

// routeEvent routes events to appropriate handlers
func (h *Handler) routeEvent(ctx context.Context, event *WebhookEvent) (*WorkflowResult, error) {
	switch event.Type {
	case "pull_request":
		return h.handlePullRequest(ctx, event)
	case "check_suite":
		return h.handleCheckSuite(ctx, event)
	case "workflow_run":
		return h.handleWorkflowRun(ctx, event)
	default:
		return &WorkflowResult{
			Success: false,
			Message: fmt.Sprintf("Unknown event type: %s", event.Type),
		}, nil
	}
}

// handlePullRequest handles pull request events
func (h *Handler) handlePullRequest(ctx context.Context, event *WebhookEvent) (*WorkflowResult, error) {
	if event.PullRequest == nil {
		return nil, fmt.Errorf("missing pull request data")
	}

	pr := event.PullRequest

	// Extract PR context
	prContext := &PRContext{
		Number:         pr.Number,
		Branch:         pr.Head.Ref,
		BaseBranch:     pr.Base.Ref,
		Author:         pr.User.Login,
		Title:          pr.Title,
		HeadSHA:        pr.Head.SHA,
		InstallationID: event.Installation.ID,
	}

	log.Printf("Processing PR #%d: %s (action: %s)", pr.Number, pr.Title, event.Action)

	// Check if we should wait for CI/CD
	if h.config.Webhooks.WaitForCI && event.Action == "synchronize" {
		log.Printf("Configured to wait for CI/CD completion for PR #%d", pr.Number)
		return &WorkflowResult{
			Success: true,
			Message: "Waiting for CI/CD to complete",
		}, nil
	}

	// Start review workflow
	return h.startReviewWorkflow(ctx, prContext)
}

// handleCheckSuite handles check suite completion events
func (h *Handler) handleCheckSuite(ctx context.Context, event *WebhookEvent) (*WorkflowResult, error) {
	if event.CheckSuite == nil {
		return nil, fmt.Errorf("missing check suite data")
	}

	cs := event.CheckSuite
	log.Printf("Check suite completed with status: %s, conclusion: %s", cs.Status, cs.Conclusion)

	// TODO: Trigger review workflow for the associated PR
	// This requires looking up the PR by HeadSHA

	return &WorkflowResult{
		Success: true,
		Message: "Check suite processed",
	}, nil
}

// handleWorkflowRun handles workflow run completion events
func (h *Handler) handleWorkflowRun(ctx context.Context, event *WebhookEvent) (*WorkflowResult, error) {
	if event.WorkflowRun == nil {
		return nil, fmt.Errorf("missing workflow run data")
	}

	wr := event.WorkflowRun
	log.Printf("Workflow run completed with status: %s, conclusion: %s", wr.Status, wr.Conclusion)

	// TODO: Trigger review workflow for the associated PR
	// This requires looking up the PR by HeadSHA

	return &WorkflowResult{
		Success: true,
		Message: "Workflow run processed",
	}, nil
}

// startReviewWorkflow initiates the code review workflow
func (h *Handler) startReviewWorkflow(ctx context.Context, prContext *PRContext) (*WorkflowResult, error) {
	log.Printf("Starting review workflow for PR #%d", prContext.Number)

	// TODO: This will orchestrate the full review workflow:
	// 1. Invoke Code Analyzer
	// 2. Invoke Standards Validator
	// 3. Invoke Code Reviewer
	// 4. Generate fixes if needed
	// 5. Post review comments

	// For Phase 1, we just log and return success
	return &WorkflowResult{
		Success: true,
		Message: fmt.Sprintf("Review workflow initiated for PR #%d", prContext.Number),
	}, nil
}
