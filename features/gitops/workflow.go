package gitops

import (
	"context"
	"fmt"

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
		commitResult, err := applyFixesDirectly(ctx, repoPath, patches)
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
		fixPR, err := createFixPR(ctx, ghClient, owner, repo, repoPath, prNumber, patches)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to create fix PR: %v", err)
			return result, err
		}

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

	// Get the base branch from the original PR
	// For now, assume it's targeting the PR's head branch
	baseBranch := fmt.Sprintf("pr-%d", originalPRNumber)
	fixBranchName := fmt.Sprintf("agent-fixes/pr-%d", originalPRNumber)

	// Create fix branch
	_, err := CreateBranch(ctx, repoPath, baseBranch, fixBranchName)
	if err != nil {
		return nil, fmt.Errorf("failed to create fix branch: %w", err)
	}

	// Apply patches
	commitMsg := GenerateCommitMessage(patches)
	applyResult, err := ApplyPatches(ctx, repoPath, patches, commitMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to apply patches: %w", err)
	}

	if !applyResult.Success {
		return nil, fmt.Errorf("patch application failed: %s", applyResult.Error)
	}

	// Push fix branch
	if err := PushBranch(ctx, repoPath, fixBranchName); err != nil {
		return nil, fmt.Errorf("failed to push fix branch: %w", err)
	}

	// Create PR
	prTitle := GeneratePRTitle(originalPRNumber, len(patches))
	prBody := GeneratePRBody(originalPRNumber, patches)

	pr, err := ghClient.CreatePullRequest(ctx, owner, repo, prTitle, prBody, fixBranchName, baseBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to create PR: %w", err)
	}

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
