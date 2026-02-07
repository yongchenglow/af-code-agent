package gitops

import (
	"context"
	"fmt"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/google/go-github/v57/github"
	"github.com/yourorg/github-code-agent/features/fixer"
	"github.com/yourorg/github-code-agent/features/reviewer"
)

// GitOps handles git operations with GitHub integration
type GitOps struct {
	agent    *agent.Agent
	ghClient *GitHubClient
}

// NewGitOps creates a new GitOps instance
func NewGitOps(app *agent.Agent, ghClient *github.Client) *GitOps {
	return &GitOps{
		agent:    app,
		ghClient: NewGitHubClient(ghClient),
	}
}

// RegisterReasoners registers all gitops reasoners with the agent
func RegisterReasoners(app *agent.Agent, ghClient *github.Client) {
	gitops := NewGitOps(app, ghClient)

	app.RegisterReasoner("create_branch",
		func(ctx context.Context, input map[string]any) (any, error) {
			return createBranchReasoner(ctx, gitops, input)
		},
		agent.WithDescription("Creates a new Git branch"))

	app.RegisterReasoner("apply_patches",
		func(ctx context.Context, input map[string]any) (any, error) {
			return applyPatchesReasoner(ctx, gitops, input)
		},
		agent.WithDescription("Applies code patches and creates commits"))

	app.RegisterReasoner("create_pull_request",
		func(ctx context.Context, input map[string]any) (any, error) {
			return createPullRequestReasoner(ctx, gitops, input)
		},
		agent.WithDescription("Creates a GitHub pull request"))

	app.RegisterReasoner("add_review_comment",
		func(ctx context.Context, input map[string]any) (any, error) {
			return addReviewCommentReasoner(ctx, gitops, input)
		},
		agent.WithDescription("Adds a code review comment to a PR"))

	app.RegisterReasoner("update_review_comment",
		func(ctx context.Context, input map[string]any) (any, error) {
			return updateReviewCommentReasoner(ctx, gitops, input)
		},
		agent.WithDescription("Updates an existing review comment"))

	app.RegisterReasoner("post_review_with_fixes",
		func(ctx context.Context, input map[string]any) (any, error) {
			return postReviewWithFixesReasoner(ctx, gitops, input)
		},
		agent.WithDescription("Orchestrates posting review comments and applying fixes"))

	app.RegisterReasoner("check_pr_exists",
		func(ctx context.Context, input map[string]any) (any, error) {
			return checkPRExistsReasoner(ctx, gitops, input)
		},
		agent.WithDescription("Checks if a PR exists for a given branch"))
}

// createBranchReasoner creates a new Git branch
func createBranchReasoner(ctx context.Context, gitops *GitOps, input map[string]any) (any, error) {
	repoPath := getString(input, "repo_path")
	baseBranch := getString(input, "base_branch")
	newBranch := getString(input, "new_branch")

	if repoPath == "" || baseBranch == "" || newBranch == "" {
		return nil, fmt.Errorf("missing required parameters")
	}

	branch, err := CreateBranch(ctx, repoPath, baseBranch, newBranch)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"branch_name": branch.Name,
		"sha":         branch.SHA,
	}, nil
}

// applyPatchesReasoner applies patches and commits
func applyPatchesReasoner(ctx context.Context, gitops *GitOps, input map[string]any) (any, error) {
	repoPath := getString(input, "repo_path")
	commitMessage := getString(input, "commit_message")

	patchesData, ok := input["patches"].([]any)
	if !ok {
		return nil, fmt.Errorf("patches must be a list")
	}

	// Convert to CodePatch array
	patches := make([]*fixer.CodePatch, 0, len(patchesData))
	for _, p := range patchesData {
		patchMap := p.(map[string]any)
		patch := &fixer.CodePatch{
			IssueID:      getString(patchMap, "issue_id"),
			FilePath:     getString(patchMap, "file_path"),
			Language:     getString(patchMap, "language"),
			OriginalCode: getString(patchMap, "original_code"),
			FixedCode:    getString(patchMap, "fixed_code"),
			Description:  getString(patchMap, "description"),
			Line:         getInt(patchMap, "line"),
		}
		patches = append(patches, patch)
	}

	result, err := ApplyPatches(ctx, repoPath, patches, commitMessage)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"success":       result.Success,
		"commit":        result.Commit,
		"applications":  result.Applications,
		"success_count": result.SuccessCount,
		"failure_count": result.FailureCount,
	}, nil
}

// createPullRequestReasoner creates a PR
func createPullRequestReasoner(ctx context.Context, gitops *GitOps, input map[string]any) (any, error) {
	owner := getString(input, "owner")
	repo := getString(input, "repo")
	title := getString(input, "title")
	body := getString(input, "body")
	headBranch := getString(input, "head_branch")
	baseBranch := getString(input, "base_branch")

	pr, err := gitops.ghClient.CreatePullRequest(ctx, owner, repo, title, body, headBranch, baseBranch)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"number":      pr.Number,
		"title":       pr.Title,
		"url":         pr.URL,
		"html_url":    pr.HTMLURL,
		"head_branch": pr.HeadBranch,
		"base_branch": pr.BaseBranch,
	}, nil
}

// addReviewCommentReasoner adds a review comment
func addReviewCommentReasoner(ctx context.Context, gitops *GitOps, input map[string]any) (any, error) {
	owner := getString(input, "owner")
	repo := getString(input, "repo")
	prNumber := getInt(input, "pr_number")

	commentData := input["comment"].(map[string]any)
	comment := &ReviewComment{
		FilePath: getString(commentData, "file_path"),
		Line:     getInt(commentData, "line"),
		Body:     getString(commentData, "body"),
		IssueID:  getString(commentData, "issue_id"),
		Severity: getString(commentData, "severity"),
	}

	commentID, err := gitops.ghClient.AddReviewComment(ctx, owner, repo, prNumber, comment)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"comment_id": commentID,
	}, nil
}

// updateReviewCommentReasoner updates a review comment
func updateReviewCommentReasoner(ctx context.Context, gitops *GitOps, input map[string]any) (any, error) {
	owner := getString(input, "owner")
	repo := getString(input, "repo")
	commentID := getInt64(input, "comment_id")
	fixLink := getString(input, "fix_link")

	err := gitops.ghClient.UpdateReviewComment(ctx, owner, repo, commentID, fixLink)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"success": true,
	}, nil
}

// postReviewWithFixesReasoner orchestrates the complete workflow
func postReviewWithFixesReasoner(ctx context.Context, gitops *GitOps, input map[string]any) (any, error) {
	owner := getString(input, "owner")
	repo := getString(input, "repo")
	repoPath := getString(input, "repo_path")
	prNumber := getInt(input, "pr_number")
	modeStr := getString(input, "mode")

	mode := SafeMode
	if modeStr == "yolo" {
		mode = YOLOMode
	}

	// Convert issues
	issuesData, ok := input["issues"].([]any)
	if !ok {
		return nil, fmt.Errorf("issues must be a list")
	}

	issues := make([]*reviewer.Issue, 0, len(issuesData))
	for _, i := range issuesData {
		issueMap := i.(map[string]any)
		issue := &reviewer.Issue{
			ID:          getString(issueMap, "id"),
			FilePath:    getString(issueMap, "file_path"),
			Line:        getInt(issueMap, "line"),
			Severity:    getString(issueMap, "severity"),
			Category:    getString(issueMap, "category"),
			Title:       getString(issueMap, "title"),
			Description: getString(issueMap, "description"),
			Suggestion:  getString(issueMap, "suggestion"),
		}
		issues = append(issues, issue)
	}

	// Convert patches
	patchesData, ok := input["patches"].([]any)
	if !ok {
		return nil, fmt.Errorf("patches must be a list")
	}

	patches := make([]*fixer.CodePatch, 0, len(patchesData))
	for _, p := range patchesData {
		patchMap := p.(map[string]any)
		patch := &fixer.CodePatch{
			IssueID:      getString(patchMap, "issue_id"),
			FilePath:     getString(patchMap, "file_path"),
			Language:     getString(patchMap, "language"),
			OriginalCode: getString(patchMap, "original_code"),
			FixedCode:    getString(patchMap, "fixed_code"),
			Description:  getString(patchMap, "description"),
			Line:         getInt(patchMap, "line"),
		}
		patches = append(patches, patch)
	}

	// Execute workflow
	result, err := PostReviewWithFixes(ctx, gitops.ghClient, owner, repo, repoPath, prNumber, issues, patches, mode)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"mode":             string(result.Mode),
		"success":          result.Success,
		"issues_reviewed":  result.IssuesReviewed,
		"fixes_applied":    result.FixesApplied,
		"commit_sha":       result.CommitSHA,
		"fix_pr":           result.FixPR,
		"fix_pr_url":       result.FixPRURL,
		"comments_posted":  result.CommentsPosted,
		"comments_updated": result.CommentsUpdated,
		"error":            result.Error,
	}, nil
}

// checkPRExistsReasoner checks if a PR exists for a given branch
func checkPRExistsReasoner(ctx context.Context, gitops *GitOps, input map[string]any) (any, error) {
	repo := getString(input, "repo")
	branch := getString(input, "branch")

	if repo == "" || branch == "" {
		return nil, fmt.Errorf("repo and branch are required")
	}

	// Parse owner/repo
	owner, repoName, err := parseRepo(repo)
	if err != nil {
		return nil, err
	}

	// Check for existing PRs from this branch
	pr, exists, err := gitops.ghClient.FindPRByBranch(ctx, owner, repoName, branch)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing PR: %w", err)
	}

	if !exists {
		return map[string]any{
			"exists": false,
		}, nil
	}

	return map[string]any{
		"exists":    true,
		"pr_number": pr.Number,
		"pr_url":    pr.HTMLURL,
		"title":     pr.Title,
	}, nil
}

// Helper functions
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key].(int); ok {
		return v
	}
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getInt64(m map[string]any, key string) int64 {
	if v, ok := m[key].(int64); ok {
		return v
	}
	if v, ok := m[key].(float64); ok {
		return int64(v)
	}
	return 0
}

func parseRepo(fullRepo string) (owner, repo string, err error) {
	parts := splitString(fullRepo, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repo format, expected owner/repo, got %s", fullRepo)
	}
	return parts[0], parts[1], nil
}

func splitString(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep[0] {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}
