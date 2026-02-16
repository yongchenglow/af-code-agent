package analyzer

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/google/go-github/v57/github"
	ghpkg "github.com/yourorg/github-code-agent/pkg/github"
)

// RegisterReasoners registers all analyzer reasoners
func RegisterReasoners(app *agent.Agent) {
	app.RegisterReasoner("analyze_pr", AnalyzePRReasoner,
		agent.WithDescription("Analyzes PR files and extracts code metrics"))

	app.RegisterReasoner("analyze_push", AnalyzePushReasoner,
		agent.WithDescription("Analyzes push commits and creates/updates PR with fixes"))

	app.RegisterReasoner("fetch_pr_files", FetchPRFilesReasoner,
		agent.WithDescription("Fetches changed files from a pull request"))
}

// AnalyzePRReasoner orchestrates the complete PR analysis
func AnalyzePRReasoner(ctx context.Context, input map[string]any) (any, error) {
	repo, ok := input["repo"].(string)
	if !ok {
		return nil, fmt.Errorf("repo is required")
	}

	prNumber, ok := input["pr_number"].(int)
	if !ok {
		// Try float64 (JSON numbers)
		if prNumberFloat, ok := input["pr_number"].(float64); ok {
			prNumber = int(prNumberFloat)
		} else {
			return nil, fmt.Errorf("pr_number is required")
		}
	}

	log.Printf("Analyzing PR #%d in repo %s", prNumber, repo)

	// Get agent from context
	agentInstance := GetAgentFromContext(ctx)
	if agentInstance == nil {
		return nil, fmt.Errorf("agent not found in context")
	}

	// Fetch PR files
	filesInput := map[string]any{
		"repo":      repo,
		"pr_number": prNumber,
	}

	filesResult, err := agentInstance.CallLocal(ctx, "fetch_pr_files", filesInput)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR files: %w", err)
	}

	filesData, ok := filesResult.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid files result")
	}

	files, ok := filesData["files"].([]any)
	if !ok {
		return nil, fmt.Errorf("invalid files result")
	}

	// Extract commit SHA for review deduplication
	commitSHA, _ := filesData["commit_sha"].(string)

	log.Printf("Fetched %d files from PR #%d (commit: %s)", len(files), prNumber, commitSHA)

	// Store results in workflow memory
	memKey := fmt.Sprintf("pr-%d-analysis", prNumber)
	analysisData := map[string]any{
		"repo":        repo,
		"pr_number":   prNumber,
		"commit_sha":  commitSHA,
		"files_count": len(files),
		"files":       files,
	}

	if err := agentInstance.Memory().WorkflowScope().Set(ctx, memKey, analysisData); err != nil {
		log.Printf("Warning: Failed to store analysis in memory: %v", err)
	}

	return map[string]any{
		"success":     true,
		"repo":        repo,
		"pr_number":   prNumber,
		"commit_sha":  commitSHA,
		"files_count": len(files),
		"files":       files,
	}, nil
}

// AnalyzePushReasoner analyzes push commits and triggers PR workflow
func AnalyzePushReasoner(ctx context.Context, input map[string]any) (any, error) {
	repo, ok := input["repo"].(string)
	if !ok {
		return nil, fmt.Errorf("repo is required")
	}

	branch, ok := input["branch"].(string)
	if !ok {
		return nil, fmt.Errorf("branch is required")
	}

	commits, ok := input["commits"].([]any)
	if !ok {
		commits = []any{}
	}

	log.Printf("Analyzing push to %s in repo %s (%d commits)", branch, repo, len(commits))

	// Get agent from context
	agentInstance := GetAgentFromContext(ctx)
	if agentInstance == nil {
		return nil, fmt.Errorf("agent not found in context")
	}

	// Check if there's an existing open PR for this branch
	// Call check_pr_exists reasoner
	checkPRInput := map[string]any{
		"repo":   repo,
		"branch": branch,
	}

	prCheckResult, err := agentInstance.CallLocal(ctx, "check_pr_exists", checkPRInput)
	if err != nil {
		log.Printf("Failed to check for existing PR: %v", err)
		return nil, err
	}

	prCheckData, _ := prCheckResult.(map[string]any)
	prExists := false
	var prNumber int

	if exists, ok := prCheckData["exists"].(bool); ok && exists {
		prExists = true
		if num, ok := prCheckData["pr_number"].(float64); ok {
			prNumber = int(num)
		} else if num, ok := prCheckData["pr_number"].(int); ok {
			prNumber = num
		}
	}

	if prExists && prNumber > 0 {
		// PR exists, trigger the full PR workflow via handle_pr_event
		log.Printf("Found existing PR #%d for branch %s, triggering review workflow...", prNumber, branch)

		prEventInput := map[string]any{
			"repo":      repo,
			"pr_number": float64(prNumber), // Use float64 to match webhook format
			"action":    "synchronize",     // Treat push as synchronize event
		}

		result, err := agentInstance.CallLocal(ctx, "handle_pr_event", prEventInput)
		if err != nil {
			log.Printf("Failed to handle PR event: %v", err)
			return nil, err
		}

		return map[string]any{
			"success":   true,
			"pr_exists": true,
			"pr_number": prNumber,
			"message":   fmt.Sprintf("PR #%d review workflow triggered", prNumber),
			"result":    result,
		}, nil
	}

	// No PR exists yet, just log for now
	log.Printf("No existing PR found for branch %s", branch)

	return map[string]any{
		"success":   true,
		"pr_exists": false,
		"message":   fmt.Sprintf("No PR found for branch %s. Create a PR to trigger review.", branch),
		"commits":   len(commits),
	}, nil
}

// FetchPRFilesReasoner fetches changed files from a PR
func FetchPRFilesReasoner(ctx context.Context, input map[string]any) (any, error) {
	repo, ok := input["repo"].(string)
	if !ok {
		return nil, fmt.Errorf("repo is required")
	}

	prNumber, ok := input["pr_number"].(int)
	if !ok {
		if prNumberFloat, ok := input["pr_number"].(float64); ok {
			prNumber = int(prNumberFloat)
		} else {
			return nil, fmt.Errorf("pr_number is required")
		}
	}

	log.Printf("Fetching files for PR #%d in %s", prNumber, repo)

	// Parse owner/repo
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format: %s (expected owner/repo)", repo)
	}
	owner, repoName := parts[0], parts[1]

	// Get GitHub client from context
	ghClient := GetGitHubClientFromContext(ctx)
	if ghClient == nil {
		log.Printf("GitHub client not found in context, using mock data")
		return fetchMockPRFiles(repo, prNumber), nil
	}

	// Fetch PR details to get the head ref
	pr, err := ghpkg.GetPR(ctx, ghClient, owner, repoName, prNumber)
	if err != nil {
		log.Printf("Failed to fetch PR details: %v, using mock data", err)
		return fetchMockPRFiles(repo, prNumber), nil
	}

	headRef := pr.GetHead().GetRef()

	// Fetch changed files
	fileChanges, err := ghpkg.GetPRFiles(ctx, ghClient, owner, repoName, prNumber)
	if err != nil {
		log.Printf("Failed to fetch PR files: %v, using mock data", err)
		return fetchMockPRFiles(repo, prNumber), nil
	}

	// Fetch file contents
	files := make([]any, 0, len(fileChanges))
	for _, file := range fileChanges {
		// Skip deleted files
		if file.Status == "removed" {
			continue
		}

		// Fetch file content
		content, err := ghpkg.GetFileContent(ctx, ghClient, owner, repoName, file.Filename, headRef)
		if err != nil {
			log.Printf("Failed to fetch content for %s: %v, skipping", file.Filename, err)
			continue
		}

		language := ghpkg.DetectLanguage(file.Filename)

		files = append(files, map[string]any{
			"filename":  file.Filename,
			"path":      file.Filename,
			"status":    file.Status,
			"additions": file.Additions,
			"deletions": file.Deletions,
			"changes":   file.Changes,
			"patch":     file.Patch,
			"language":  language,
			"content":   content,
		})
	}

	log.Printf("Successfully fetched %d files from PR #%d", len(files), prNumber)

	// Get commit SHA for deduplication
	commitSHA := pr.GetHead().GetSHA()

	return map[string]any{
		"repo":       repo,
		"pr_number":  prNumber,
		"commit_sha": commitSHA,
		"files":      files,
	}, nil
}

// fetchMockPRFiles returns mock data for testing
func fetchMockPRFiles(repo string, prNumber int) map[string]any {
	mockFiles := []any{
		map[string]any{
			"filename":  "main.go",
			"path":      "main.go",
			"status":    "modified",
			"additions": 10,
			"deletions": 5,
			"language":  "go",
			"content": `package main

import (
	"fmt"
	"os"
)

func main() {
	// Missing input validation
	input := os.Args[1]

	// Potential nil pointer dereference
	config := loadConfig()
	fmt.Println(config.Name)

	// Inefficient string concatenation
	result := ""
	for i := 0; i < 100; i++ {
		result += "item"
	}
}

func loadConfig() *Config {
	return &Config{Name: "test"}
}

type Config struct {
	Name string
}
`,
		},
	}

	return map[string]any{
		"repo":      repo,
		"pr_number": prNumber,
		"files":     mockFiles,
	}
}

// GetAgentFromContext retrieves the agent instance from context
func GetAgentFromContext(ctx context.Context) *agent.Agent {
	if agentInstance, ok := ctx.Value("agent").(*agent.Agent); ok {
		return agentInstance
	}
	return nil
}

// GetGitHubClientFromContext retrieves the GitHub client from context
func GetGitHubClientFromContext(ctx context.Context) *github.Client {
	if client, ok := ctx.Value("github_client").(*ghpkg.Client); ok {
		return client.GetClient()
	}
	return nil
}
