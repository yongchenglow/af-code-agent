package gitops

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/yourorg/github-code-agent/agents/fixer"
	"github.com/yourorg/github-code-agent/agents/reviewer"
)

// ReviewHistory tracks previous reviews for deduplication
type ReviewHistory struct {
	PRNumber   int              `json:"pr_number"`
	CommitSHA  string           `json:"commit_sha"`
	IssueIDs   []string         `json:"issue_ids"`
	ReviewedAt string           `json:"reviewed_at"`
	CommentIDs map[string]int64 `json:"comment_ids"` // issueID -> commentID
}

// saveReviewHistory persists review data to prevent duplicate comments
func saveReviewHistory(
	ctx context.Context,
	agentInstance *agent.Agent,
	repo string,
	prNumber int,
	commitSHA string,
	issues []*reviewer.Issue,
	commentIDs map[string]int64,
) error {
	issueIDs := make([]string, 0, len(issues))
	for _, issue := range issues {
		issueIDs = append(issueIDs, issue.ID)
	}

	history := ReviewHistory{
		PRNumber:   prNumber,
		CommitSHA:  commitSHA,
		IssueIDs:   issueIDs,
		ReviewedAt: time.Now().Format(time.RFC3339),
		CommentIDs: commentIDs,
	}

	// Use GlobalScope for persistence across webhook events
	memKey := fmt.Sprintf("review-history:%s:pr-%d", repo, prNumber)
	return agentInstance.Memory().GlobalScope().Set(ctx, memKey, history)
}

// getReviewHistory retrieves previous review data for deduplication
func getReviewHistory(
	ctx context.Context,
	agentInstance *agent.Agent,
	repo string,
	prNumber int,
) (*ReviewHistory, error) {
	memKey := fmt.Sprintf("review-history:%s:pr-%d", repo, prNumber)

	var history ReviewHistory
	err := agentInstance.Memory().GlobalScope().GetTyped(ctx, memKey, &history)
	if err != nil {
		return nil, err
	}
	if history.CommitSHA == "" {
		return nil, nil // No previous review
	}
	return &history, nil
}

// PostReviewWithFixes orchestrates the complete review + fix workflow
func PostReviewWithFixes(
	ctx context.Context,
	agentInstance *agent.Agent,
	ghClient *GitHubClient,
	owner, repo, repoPath string,
	prNumber int,
	currentCommitSHA string,
	issues []*reviewer.Issue,
	patches []*fixer.CodePatch,
	mode OperationMode,
) (*WorkflowResult, error) {

	result := &WorkflowResult{
		Mode:           mode,
		Success:        true,
		IssuesReviewed: len(issues),
		FixesApplied:   len(patches),
	}

	// Check previous review history for deduplication
	repoFullName := fmt.Sprintf("%s/%s", owner, repo)
	prevReview, err := getReviewHistory(ctx, agentInstance, repoFullName, prNumber)
	if err != nil {
		log.Printf("Warning: Failed to get review history: %v", err)
	}

	// Skip if commit hasn't changed since last review
	if prevReview != nil && prevReview.CommitSHA == currentCommitSHA {
		log.Printf("PR #%d: No new commits since last review (SHA: %s), skipping", prNumber, currentCommitSHA)
		result.Success = true
		result.IssuesReviewed = 0
		result.CommentsPosted = 0
		return result, nil
	}

	// Filter out duplicate issues from previous review
	var filteredIssues []*reviewer.Issue
	if prevReview != nil {
		prevIssueSet := make(map[string]bool)
		for _, issueID := range prevReview.IssueIDs {
			prevIssueSet[issueID] = true
		}

		for _, issue := range issues {
			if !prevIssueSet[issue.ID] {
				filteredIssues = append(filteredIssues, issue)
			}
		}

		duplicateCount := len(issues) - len(filteredIssues)
		if duplicateCount > 0 {
			log.Printf("PR #%d: Filtered %d duplicate issues, %d new issues to post",
				prNumber, duplicateCount, len(filteredIssues))
		}

		if len(filteredIssues) == 0 {
			log.Printf("PR #%d: No new issues found (commit SHA: %s)", prNumber, currentCommitSHA)
			// Save that we reviewed this commit even with no new issues
			if err := saveReviewHistory(ctx, agentInstance, repoFullName, prNumber, currentCommitSHA, issues, prevReview.CommentIDs); err != nil {
				log.Printf("Warning: Failed to save review history: %v", err)
			}
			result.Success = true
			result.IssuesReviewed = 0
			result.CommentsPosted = 0
			return result, nil
		}
	} else {
		// First review - post all issues
		filteredIssues = issues
		log.Printf("PR #%d: First review, posting all %d issues", prNumber, len(issues))
	}

	// Update result counts to reflect filtered issues
	result.IssuesReviewed = len(filteredIssues)

	// Clone the repository if we have patches to apply
	var actualRepoPath string
	if len(patches) > 0 {
		fmt.Printf("DEBUG: Starting to clone repository for %d patches\n", len(patches))

		// Get GitHub token from environment
		token := GetGitHubTokenFromEnv()
		if token == "" {
			result.Success = false
			result.Error = "GITHUB_TOKEN not found in environment"
			fmt.Printf("ERROR: GITHUB_TOKEN not found in environment\n")
			return result, fmt.Errorf("GITHUB_TOKEN not found")
		}

		repoURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
		fmt.Printf("DEBUG: Cloning repository from %s\n", repoURL)

		clonedPath, err := CloneRepository(ctx, repoURL, token)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to clone repository: %v", err)
			fmt.Printf("ERROR: Failed to clone repository: %v\n", err)
			return result, err
		}
		actualRepoPath = clonedPath
		fmt.Printf("SUCCESS: Cloned repository to %s\n", actualRepoPath)
	} else {
		fmt.Printf("DEBUG: No patches to apply, skipping repository operations\n")
	}

	// Step 1: Post initial review comments for filtered issues
	commentIDs := make(map[string]int64) // issueID -> commentID

	// Merge with previous comment IDs if they exist
	if prevReview != nil && prevReview.CommentIDs != nil {
		for issueID, commentID := range prevReview.CommentIDs {
			commentIDs[issueID] = commentID
		}
	}

	for _, issue := range filteredIssues {
		comment := &ReviewComment{
			FilePath: issue.FilePath,
			Line:     issue.Line,
			Body:     formatIssueBody(issue),
			IssueID:  issue.ID,
			Severity: issue.Severity,
		}

		commentID, err := ghClient.AddReviewComment(ctx, owner, repo, prNumber, comment)
		if err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to post comment for issue %s: %v\n", issue.ID, err)
			continue
		}

		commentIDs[issue.ID] = commentID
		result.CommentsPosted++
	}

	// Save review history after posting comments
	allIssues := issues // Store all issues (including filtered ones) for history
	if err := saveReviewHistory(ctx, agentInstance, repoFullName, prNumber, currentCommitSHA, allIssues, commentIDs); err != nil {
		log.Printf("Warning: Failed to save review history: %v", err)
	}

	// If no patches to apply, we're done
	if len(patches) == 0 {
		return result, nil
	}

	// Step 2: Apply fixes based on mode
	if mode == YOLOMode {
		// YOLO mode: Push directly to PR branch
		commitResult, err := applyFixesDirectly(ctx, actualRepoPath, patches)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to apply fixes in YOLO mode: %v", err)
			return result, err
		}

		result.CommitSHA = commitResult.SHA

		// Step 3: Update comments with commit links
		for _, patch := range patches {
			commentID, ok := commentIDs[patch.IssueID]
			if !ok {
				continue
			}

			commitURL := fmt.Sprintf("https://github.com/%s/%s/commit/%s", owner, repo, commitResult.SHA)
			fixLink := fmt.Sprintf("[View fix commit](%s)", commitURL)

			if err := ghClient.UpdateReviewComment(ctx, owner, repo, commentID, fixLink); err != nil {
				fmt.Printf("Warning: failed to update comment %d: %v\n", commentID, err)
			} else {
				result.CommentsUpdated++
			}
		}

		// Post summary comment
		summaryBody := GenerateSummaryComment(len(filteredIssues), 0, "", YOLOMode)
		if err := ghClient.AddIssueComment(ctx, owner, repo, prNumber, summaryBody); err != nil {
			fmt.Printf("Warning: failed to post summary comment: %v\n", err)
		}

	} else {
		// Safe mode: Create fix PR
		fmt.Printf("DEBUG: Creating fix PR in safe mode for %d patches\n", len(patches))
		fmt.Printf("DEBUG: Repository path: %s\n", actualRepoPath)

		fixPR, err := createFixPR(ctx, ghClient, owner, repo, actualRepoPath, prNumber, patches)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to create fix PR: %v", err)
			fmt.Printf("ERROR: Failed to create fix PR: %v\n", err)
			return result, err
		}

		fmt.Printf("SUCCESS: Created fix PR #%d: %s\n", fixPR.Number, fixPR.HTMLURL)

		result.FixPR = fixPR.Number
		result.FixPRURL = fixPR.HTMLURL

		// Step 3: Update comments with PR links
		for _, patch := range patches {
			commentID, ok := commentIDs[patch.IssueID]
			if !ok {
				continue
			}

			fixLink := fmt.Sprintf("[View fix PR #%d](%s)", fixPR.Number, fixPR.HTMLURL)

			if err := ghClient.UpdateReviewComment(ctx, owner, repo, commentID, fixLink); err != nil {
				fmt.Printf("Warning: failed to update comment %d: %v\n", commentID, err)
			} else {
				result.CommentsUpdated++
			}
		}

		// Post summary comment on original PR
		summaryBody := GenerateSummaryComment(len(filteredIssues), fixPR.Number, fixPR.HTMLURL, SafeMode)
		if err := ghClient.AddIssueComment(ctx, owner, repo, prNumber, summaryBody); err != nil {
			fmt.Printf("Warning: failed to post summary comment: %v\n", err)
		}
	}

	return result, nil
}

// applyFixesDirectly applies fixes directly to the current branch (YOLO mode)
func applyFixesDirectly(ctx context.Context, repoPath string, patches []*fixer.CodePatch) (*CommitResult, error) {
	commitMsg := GenerateCommitMessage(patches)

	result, err := ApplyPatches(ctx, repoPath, patches, commitMsg)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("failed to apply patches: %s", result.Error)
	}

	return result.Commit, nil
}

// createFixPR creates a new PR with fixes (Safe mode)
func createFixPR(
	ctx context.Context,
	ghClient *GitHubClient,
	owner, repo, repoPath string,
	originalPRNumber int,
	patches []*fixer.CodePatch,
) (*PullRequestInfo, error) {

	// First, fetch the original PR to get the actual head branch
	fmt.Printf("DEBUG: Fetching original PR #%d details\n", originalPRNumber)
	// For now, we'll use a placeholder - this needs the actual PR head branch
	// TODO: Fetch the actual PR details to get head branch
	baseBranch := "feature/performance-issues" // This should come from the original PR
	fixBranchName := fmt.Sprintf("agent-fixes/pr-%d", originalPRNumber)

	fmt.Printf("DEBUG: Base branch: %s, Fix branch: %s\n", baseBranch, fixBranchName)

	// Create fix branch
	fmt.Printf("DEBUG: Creating branch %s from %s\n", fixBranchName, baseBranch)
	_, err := CreateBranch(ctx, repoPath, baseBranch, fixBranchName)
	if err != nil {
		fmt.Printf("ERROR: Failed to create branch: %v\n", err)
		return nil, fmt.Errorf("failed to create fix branch: %w", err)
	}
	fmt.Printf("SUCCESS: Created branch %s\n", fixBranchName)

	// Apply patches
	commitMsg := GenerateCommitMessage(patches)
	fmt.Printf("DEBUG: Applying %d patches with message: %s\n", len(patches), commitMsg)
	applyResult, err := ApplyPatches(ctx, repoPath, patches, commitMsg)
	if err != nil {
		fmt.Printf("ERROR: Failed to apply patches: %v\n", err)
		return nil, fmt.Errorf("failed to apply patches: %w", err)
	}

	if !applyResult.Success {
		fmt.Printf("ERROR: Patch application failed: %s\n", applyResult.Error)
		return nil, fmt.Errorf("patch application failed: %s", applyResult.Error)
	}
	fmt.Printf("SUCCESS: Applied patches successfully\n")

	// Push fix branch
	fmt.Printf("DEBUG: Pushing branch %s\n", fixBranchName)
	if err := PushBranch(ctx, repoPath, fixBranchName); err != nil {
		fmt.Printf("ERROR: Failed to push branch: %v\n", err)
		return nil, fmt.Errorf("failed to push fix branch: %w", err)
	}
	fmt.Printf("SUCCESS: Pushed branch %s\n", fixBranchName)

	// Create PR
	prTitle := GeneratePRTitle(originalPRNumber, len(patches))
	prBody := GeneratePRBody(originalPRNumber, patches)

	fmt.Printf("DEBUG: Creating PR: %s -> %s\n", fixBranchName, baseBranch)
	fmt.Printf("DEBUG: PR title: %s\n", prTitle)

	pr, err := ghClient.CreatePullRequest(ctx, owner, repo, prTitle, prBody, fixBranchName, baseBranch)
	if err != nil {
		fmt.Printf("ERROR: Failed to create PR: %v\n", err)
		return nil, fmt.Errorf("failed to create PR: %w", err)
	}

	fmt.Printf("SUCCESS: Created PR #%d: %s\n", pr.Number, pr.HTMLURL)
	return pr, nil
}

// formatIssueBody formats an issue for a review comment
func formatIssueBody(issue *reviewer.Issue) string {
	body := fmt.Sprintf("**%s**\n\n%s", issue.Title, issue.Description)

	if issue.Suggestion != "" {
		body += fmt.Sprintf("\n\n**Suggested Fix:**\n%s", issue.Suggestion)
	}

	body += fmt.Sprintf("\n\n_Category: %s_", issue.Category)

	return body
}

// GetGitHubTokenFromEnv retrieves GitHub token from environment
func GetGitHubTokenFromEnv() string {
	return os.Getenv("GITHUB_TOKEN")
}
