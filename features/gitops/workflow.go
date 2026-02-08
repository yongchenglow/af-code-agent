package gitops

import (
	"context"
	"fmt"
	"os"

	"github.com/yourorg/github-code-agent/features/fixer"
	"github.com/yourorg/github-code-agent/features/reviewer"
)

// PostReviewWithFixes orchestrates the complete review + fix workflow
func PostReviewWithFixes(
	ctx context.Context,
	ghClient *GitHubClient,
	owner, repo, repoPath string,
	prNumber int,
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

	// Step 1: Post initial review comments for all issues
	commentIDs := make(map[string]int64) // issueID -> commentID

	for _, issue := range issues {
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
		summaryBody := GenerateSummaryComment(len(issues), 0, "", YOLOMode)
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
		summaryBody := GenerateSummaryComment(len(issues), fixPR.Number, fixPR.HTMLURL, SafeMode)
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
