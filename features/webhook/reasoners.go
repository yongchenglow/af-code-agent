package webhook

import (
	"context"
	"fmt"
	"log"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
)

// RegisterReasoners registers all webhook-related reasoners
func RegisterReasoners(app *agent.Agent) {
	// Main webhook handler
	app.RegisterReasoner("handle_webhook", HandleWebhook,
		agent.WithDescription("Processes incoming GitHub webhook events"))

	// PR-specific handler
	app.RegisterReasoner("handle_pr_event", HandlePREvent,
		agent.WithDescription("Handles pull request events"))

	// Check suite handler
	app.RegisterReasoner("handle_check_suite", HandleCheckSuite,
		agent.WithDescription("Handles CI/CD check suite completion events"))
}

// HandleWebhook processes incoming webhook events
func HandleWebhook(ctx context.Context, input map[string]any) (any, error) {
	eventType, ok := input["event_type"].(string)
	if !ok {
		return nil, fmt.Errorf("event_type is required")
	}

	// Get raw payload bytes for signature validation
	payloadBytes, ok := input["payload_raw"].([]byte)
	if !ok {
		// Fallback: try to get as string and convert
		if payloadStr, ok := input["payload_raw"].(string); ok {
			payloadBytes = []byte(payloadStr)
		} else {
			return nil, fmt.Errorf("payload_raw is required for signature validation")
		}
	}

	payload, ok := input["payload"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("payload is required")
	}

	signature, ok := input["signature"].(string)
	if !ok {
		return nil, fmt.Errorf("signature is required (X-Hub-Signature-256 header)")
	}

	// Get webhook secret from environment
	webhookSecret, ok := input["webhook_secret"].(string)
	if !ok {
		return nil, fmt.Errorf("webhook_secret is required")
	}

	// Validate webhook signature using the correct implementation
	if err := ValidateSignature(payloadBytes, signature, webhookSecret); err != nil {
		return nil, fmt.Errorf("webhook validation failed: %w", err)
	}

	log.Printf("Processing webhook event: %s", eventType)

	// Route to appropriate handler based on event type
	switch eventType {
	case "pull_request":
		return handlePullRequestEvent(ctx, payload)
	case "check_suite":
		return handleCheckSuiteEvent(ctx, payload)
	case "workflow_run":
		return handleWorkflowRunEvent(ctx, payload)
	default:
		return map[string]any{
			"success": false,
			"message": fmt.Sprintf("Unsupported event type: %s", eventType),
		}, nil
	}
}

// HandlePREvent orchestrates the PR review workflow
func HandlePREvent(ctx context.Context, input map[string]any) (any, error) {
	prNumber, ok := input["pr_number"].(float64)
	if !ok {
		return nil, fmt.Errorf("pr_number is required")
	}

	repo, ok := input["repo"].(string)
	if !ok {
		return nil, fmt.Errorf("repo is required")
	}

	action, ok := input["action"].(string)
	if !ok {
		action = "opened"
	}

	log.Printf("Handling PR #%d event (action: %s)", int(prNumber), action)

	// Get agent from context
	agentInstance := GetAgentFromContext(ctx)
	if agentInstance == nil {
		return nil, fmt.Errorf("agent not found in context")
	}

	// Call analyzer reasoner to analyze the PR
	analyzerInput := map[string]any{
		"repo":      repo,
		"pr_number": int(prNumber),
	}

	result, err := agentInstance.CallLocal(ctx, "analyze_pr", analyzerInput)
	if err != nil {
		log.Printf("Failed to analyze PR: %v", err)
		return nil, err
	}

	log.Printf("PR analysis completed: %v", result)

	return map[string]any{
		"success": true,
		"message": fmt.Sprintf("PR #%d review workflow initiated", int(prNumber)),
		"result":  result,
	}, nil
}

// HandleCheckSuite handles check suite completion events
func HandleCheckSuite(ctx context.Context, input map[string]any) (any, error) {
	conclusion, ok := input["conclusion"].(string)
	if !ok {
		conclusion = "unknown"
	}

	log.Printf("Check suite completed with conclusion: %s", conclusion)

	// TODO: Trigger review workflow for the associated PR if configured to wait for CI

	return map[string]any{
		"success": true,
		"message": "Check suite processed",
	}, nil
}

// Helper functions

func handlePullRequestEvent(ctx context.Context, payload map[string]any) (any, error) {
	action, _ := payload["action"].(string)
	pr, ok := payload["pull_request"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing pull_request data in payload")
	}

	prNumber, _ := pr["number"].(float64)
	repo, _ := payload["repository"].(map[string]any)
	repoFullName, _ := repo["full_name"].(string)

	agentInstance := GetAgentFromContext(ctx)
	if agentInstance == nil {
		return nil, fmt.Errorf("agent not found in context")
	}

	// Call handle_pr_event reasoner
	prInput := map[string]any{
		"pr_number": prNumber,
		"repo":      repoFullName,
		"action":    action,
	}

	return agentInstance.CallLocal(ctx, "handle_pr_event", prInput)
}

func handleCheckSuiteEvent(ctx context.Context, payload map[string]any) (any, error) {
	checkSuite, ok := payload["check_suite"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing check_suite data in payload")
	}

	conclusion, _ := checkSuite["conclusion"].(string)

	agentInstance := GetAgentFromContext(ctx)
	if agentInstance == nil {
		return nil, fmt.Errorf("agent not found in context")
	}

	// Call handle_check_suite reasoner
	checkInput := map[string]any{
		"conclusion": conclusion,
	}

	return agentInstance.CallLocal(ctx, "handle_check_suite", checkInput)
}

func handleWorkflowRunEvent(ctx context.Context, payload map[string]any) (any, error) {
	workflowRun, ok := payload["workflow_run"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing workflow_run data in payload")
	}

	conclusion, _ := workflowRun["conclusion"].(string)

	log.Printf("Workflow run completed with conclusion: %s", conclusion)

	return map[string]any{
		"success": true,
		"message": "Workflow run processed",
	}, nil
}

// ValidateWebhookSignature is deprecated - use ValidateSignature from validator.go instead
// Kept for backwards compatibility
func ValidateWebhookSignature(payload map[string]any, signature, secret string) error {
	return fmt.Errorf("deprecated: use ValidateSignature with raw payload bytes instead")
}

// GetAgentFromContext retrieves the agent instance from context
func GetAgentFromContext(ctx context.Context) *agent.Agent {
	if agentInstance, ok := ctx.Value("agent").(*agent.Agent); ok {
		return agentInstance
	}
	return nil
}
