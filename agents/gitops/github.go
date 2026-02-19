package gitops

import (
	"context"
	"fmt"

	"github.com/google/go-github/v57/github"
)

// GitHubClient wraps GitHub API operations
type GitHubClient struct {
	client *github.Client
}

// NewGitHubClient creates a new GitHub client wrapper
func NewGitHubClient(client *github.Client) *GitHubClient {
	return &GitHubClient{
		client: client,
	}
}

// CreatePullRequest creates a new pull request
func (gh *GitHubClient) CreatePullRequest(ctx context.Context, owner, repo, title, body, headBranch, baseBranch string) (*PullRequestInfo, error) {
	pr := &github.NewPullRequest{
		Title: github.String(title),
		Body:  github.String(body),
		Head:  github.String(headBranch),
		Base:  github.String(baseBranch),
	}

	createdPR, _, err := gh.client.PullRequests.Create(ctx, owner, repo, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	return &PullRequestInfo{
		Number:     createdPR.GetNumber(),
		Title:      createdPR.GetTitle(),
		Body:       createdPR.GetBody(),
		HeadBranch: createdPR.GetHead().GetRef(),
		BaseBranch: createdPR.GetBase().GetRef(),
		State:      createdPR.GetState(),
		URL:        createdPR.GetURL(),
		HTMLURL:    createdPR.GetHTMLURL(),
	}, nil
}

// AddReviewComment posts a review comment to a PR using the Review API
func (gh *GitHubClient) AddReviewComment(ctx context.Context, owner, repo string, prNumber int, comment *ReviewComment) (int64, error) {
	body := formatCommentBody(comment)

	// Get the latest commit SHA for the PR
	pr, _, err := gh.client.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return 0, fmt.Errorf("failed to get PR: %w", err)
	}

	commitSHA := pr.GetHead().GetSHA()

	// Create a review with a single comment
	// Use Side: "RIGHT" to comment on the new code
	reviewComment := &github.DraftReviewComment{
		Path: github.String(comment.FilePath),
		Body: github.String(body),
		Line: github.Int(comment.Line),
		Side: github.String("RIGHT"), // Comment on new code (additions)
	}

	review := &github.PullRequestReviewRequest{
		CommitID: github.String(commitSHA),
		Body:     github.String("Automated code review"),
		Event:    github.String("COMMENT"), // Use COMMENT instead of APPROVE/REQUEST_CHANGES
		Comments: []*github.DraftReviewComment{reviewComment},
	}

	createdReview, _, err := gh.client.PullRequests.CreateReview(ctx, owner, repo, prNumber, review)
	if err != nil {
		return 0, fmt.Errorf("failed to create review comment: %w", err)
	}

	// Return the review ID (we'll need to track individual comments differently)
	return createdReview.GetID(), nil
}

// UpdateReviewComment updates an existing comment with fix link
func (gh *GitHubClient) UpdateReviewComment(ctx context.Context, owner, repo string, commentID int64, fixLink string) error {
	// Get existing comment
	comment, _, err := gh.client.PullRequests.GetComment(ctx, owner, repo, commentID)
	if err != nil {
		return fmt.Errorf("failed to get comment: %w", err)
	}

	// Append fix link to comment body
	updatedBody := fmt.Sprintf("%s\n\n✅ **Fix available:** %s", comment.GetBody(), fixLink)

	update := &github.PullRequestComment{
		Body: github.String(updatedBody),
	}

	_, _, err = gh.client.PullRequests.EditComment(ctx, owner, repo, commentID, update)
	if err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	return nil
}

// AddIssueComment posts a comment on the PR issue
func (gh *GitHubClient) AddIssueComment(ctx context.Context, owner, repo string, prNumber int, body string) error {
	comment := &github.IssueComment{
		Body: github.String(body),
	}

	_, _, err := gh.client.Issues.CreateComment(ctx, owner, repo, prNumber, comment)
	if err != nil {
		return fmt.Errorf("failed to create issue comment: %w", err)
	}

	return nil
}

// formatCommentBody creates a well-formatted comment body
func formatCommentBody(comment *ReviewComment) string {
	emoji := getSeverityEmoji(comment.Severity)

	return fmt.Sprintf(`%s **%s**: %s

_Automated review by GitHub Code Agent_`,
		emoji, comment.Severity, comment.Body)
}

// getSeverityEmoji returns an emoji for the severity level
func getSeverityEmoji(severity string) string {
	switch severity {
	case "Critical":
		return "🔴"
	case "High":
		return "🟠"
	case "Medium":
		return "🟡"
	case "Low":
		return "🔵"
	default:
		return "⚪"
	}
}

// FindPRByBranch finds an open PR for the given head branch
func (gh *GitHubClient) FindPRByBranch(ctx context.Context, owner, repo, branch string) (*PullRequestInfo, bool, error) {
	opts := &github.PullRequestListOptions{
		State: "open",
		Head:  fmt.Sprintf("%s:%s", owner, branch),
		ListOptions: github.ListOptions{
			PerPage: 10,
		},
	}

	prs, _, err := gh.client.PullRequests.List(ctx, owner, repo, opts)
	if err != nil {
		return nil, false, fmt.Errorf("failed to list pull requests: %w", err)
	}

	if len(prs) == 0 {
		return nil, false, nil
	}

	// Return the first matching PR
	pr := prs[0]
	return &PullRequestInfo{
		Number:     pr.GetNumber(),
		Title:      pr.GetTitle(),
		Body:       pr.GetBody(),
		HeadBranch: pr.GetHead().GetRef(),
		BaseBranch: pr.GetBase().GetRef(),
		State:      pr.GetState(),
		URL:        pr.GetURL(),
		HTMLURL:    pr.GetHTMLURL(),
	}, true, nil
}

// GenerateSummaryComment creates a summary comment for the original PR
func GenerateSummaryComment(issueCount, fixPRNumber int, fixPRURL string, mode OperationMode) string {
	if mode == YOLOMode {
		return fmt.Sprintf(`🤖 **Automated Code Review Complete**

Found %d issue(s) and applied fixes directly to this PR.

_All fixes have been validated and committed._`, issueCount)
	}

	return fmt.Sprintf(`🤖 **Automated Code Review Complete**

Found %d issue(s). Fixes are available in PR #%d

👉 [Review fixes](%s)

_Please review and merge the fix PR if changes are acceptable._`, issueCount, fixPRNumber, fixPRURL)
}
