package webhook

import (
	"context"
	"fmt"
	"log"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/yourorg/github-code-agent/pkg/config"
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
	case "push":
		return handlePushEvent(ctx, payload)
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

	analysisResult, err := agentInstance.CallLocal(ctx, "analyze_pr", analyzerInput)
	if err != nil {
		log.Printf("Failed to analyze PR: %v", err)
		return nil, err
	}

	log.Printf("PR analysis completed: %v", analysisResult)

	// Extract files from analysis result
	analysisData, ok := analysisResult.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid analysis result format")
	}

	files, ok := analysisData["files"].([]any)
	if !ok || len(files) == 0 {
		log.Printf("No files to review in PR #%d", int(prNumber))
		return map[string]any{
			"success": true,
			"message": fmt.Sprintf("PR #%d has no files to review", int(prNumber)),
			"result":  analysisResult,
		}, nil
	}

	// Call review_code reasoner to review the files
	reviewInput := map[string]any{
		"files": files,
		"pr_context": map[string]any{
			"repo":      repo,
			"pr_number": int(prNumber),
		},
	}

	reviewResult, err := agentInstance.CallLocal(ctx, "review_code", reviewInput)
	if err != nil {
		log.Printf("Failed to review code: %v", err)
		return nil, err
	}

	log.Printf("Code review completed for PR #%d", int(prNumber))

	// Extract issues from review result
	reviewData, ok := reviewResult.(map[string]any)
	if !ok {
		log.Printf("Review result is not a map: %T", reviewResult)
		return nil, fmt.Errorf("invalid review result format")
	}

	log.Printf("Review data keys: %v", getKeys(reviewData))

	reportData, ok := reviewData["report"].(map[string]any)
	if !ok {
		log.Printf("No report in review result for PR #%d (report type: %T)", int(prNumber), reviewData["report"])
		return map[string]any{
			"success":        true,
			"message":        fmt.Sprintf("PR #%d review completed (no report data)", int(prNumber)),
			"analysis":       analysisResult,
			"review":         reviewResult,
			"files_reviewed": len(files),
		}, nil
	}

	log.Printf("Report data keys: %v", getKeys(reportData))

	issues, ok := reportData["issues"].([]any)
	if !ok {
		log.Printf("Issues not found or wrong type: %T", reportData["issues"])
	}
	if !ok || len(issues) == 0 {
		log.Printf("No issues found in PR #%d (ok=%v, len=%d)", int(prNumber), ok, len(issues))
		return map[string]any{
			"success":        true,
			"message":        fmt.Sprintf("PR #%d review completed (no issues found)", int(prNumber)),
			"analysis":       analysisResult,
			"review":         reviewResult,
			"files_reviewed": len(files),
		}, nil
	}

	log.Printf("Found %d issues in PR #%d, generating fixes...", len(issues), int(prNumber))

	// Generate fixes with validation
	fixerInput := map[string]any{
		"issues": issues,
		"files":  files,
	}

	fixerResult, err := agentInstance.CallLocal(ctx, "generate_fixes_with_validation", fixerInput)
	if err != nil {
		log.Printf("Failed to generate fixes: %v", err)
		// Continue even if fixes fail - we still want to post review comments
	}

	log.Printf("Fix generation completed for PR #%d", int(prNumber))

	// Extract successful fixes (patches)
	var patches []any
	if fixerResult != nil {
		if fixerData, ok := fixerResult.(map[string]any); ok {
			if successfulFixes, ok := fixerData["successful_fixes"].([]any); ok {
				patches = successfulFixes
			}
		}
	}

	// Get config from context to determine mode
	cfg := GetConfigFromContext(ctx)
	mode := "safe" // default
	if cfg != nil && cfg.Agent.Mode == "yolo" {
		mode = "yolo"
	}

	// Parse owner/repo
	parts := []string{}
	for i := 0; i < len(repo); i++ {
		if repo[i] == '/' {
			parts = append(parts, repo[:i])
			parts = append(parts, repo[i+1:])
			break
		}
	}
	if len(parts) != 2 {
		log.Printf("Invalid repo format: %s", repo)
		return map[string]any{
			"success":        true,
			"message":        fmt.Sprintf("PR #%d review completed but could not post to GitHub (invalid repo format)", int(prNumber)),
			"analysis":       analysisResult,
			"review":         reviewResult,
			"fixes":          fixerResult,
			"files_reviewed": len(files),
			"issues_found":   len(issues),
		}, nil
	}
	owner, repoName := parts[0], parts[1]

	// Post review comments and apply fixes using the gitops workflow
	workflowInput := map[string]any{
		"owner":      owner,
		"repo":       repoName,
		"repo_path":  "/tmp/" + repoName, // Temporary path for git operations
		"pr_number":  int(prNumber),
		"mode":       mode,
		"issues":     issues,
		"patches":    patches,
	}

	workflowResult, err := agentInstance.CallLocal(ctx, "post_review_with_fixes", workflowInput)
	if err != nil {
		log.Printf("Failed to post review and apply fixes: %v", err)
		return map[string]any{
			"success":        false,
			"message":        fmt.Sprintf("PR #%d review completed but failed to post to GitHub: %v", int(prNumber), err),
			"analysis":       analysisResult,
			"review":         reviewResult,
			"fixes":          fixerResult,
			"files_reviewed": len(files),
			"issues_found":   len(issues),
		}, nil
	}

	log.Printf("Successfully posted review and fixes for PR #%d", int(prNumber))

	return map[string]any{
		"success":        true,
		"message":        fmt.Sprintf("PR #%d review workflow completed with %d issues found and posted to GitHub", int(prNumber), len(issues)),
		"analysis":       analysisResult,
		"review":         reviewResult,
		"fixes":          fixerResult,
		"workflow":       workflowResult,
		"files_reviewed": len(files),
		"issues_found":   len(issues),
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

func handlePushEvent(ctx context.Context, payload map[string]any) (any, error) {
	ref, _ := payload["ref"].(string)
	repo, _ := payload["repository"].(map[string]any)
	repoFullName, _ := repo["full_name"].(string)
	commits, _ := payload["commits"].([]any)

	log.Printf("Push event received: %s to %s (%d commits)", ref, repoFullName, len(commits))

	// Only process pushes to feature branches (not main/master)
	if ref == "refs/heads/main" || ref == "refs/heads/master" {
		return map[string]any{
			"success": true,
			"message": "Ignoring push to main/master branch",
		}, nil
	}

	// Extract branch name from ref (refs/heads/branch-name -> branch-name)
	branchName := ref
	if len(ref) > 11 && ref[:11] == "refs/heads/" {
		branchName = ref[11:]
	}

	agentInstance := GetAgentFromContext(ctx)
	if agentInstance == nil {
		return nil, fmt.Errorf("agent not found in context")
	}

	// Call analyze_push reasoner to analyze the commits and create/update PR
	pushInput := map[string]any{
		"repo":        repoFullName,
		"branch":      branchName,
		"commits":     commits,
		"head_commit": payload["head_commit"],
	}

	result, err := agentInstance.CallLocal(ctx, "analyze_push", pushInput)
	if err != nil {
		log.Printf("Failed to analyze push: %v", err)
		return nil, err
	}

	return map[string]any{
		"success": true,
		"message": fmt.Sprintf("Push to %s analyzed successfully", branchName),
		"result":  result,
	}, nil
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

// getKeys returns the keys of a map for debugging
func getKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// GetConfigFromContext retrieves the config instance from context
func GetConfigFromContext(ctx context.Context) *config.Config {
	if cfg, ok := ctx.Value("config").(*config.Config); ok {
		return cfg
	}
	return nil
}
