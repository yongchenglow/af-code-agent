package reviewer

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v57/github"
)

// CommentPoster handles posting review comments to GitHub
type CommentPoster struct {
	githubClient *github.Client
}

// NewCommentPoster creates a new comment poster
func NewCommentPoster(client *github.Client) *CommentPoster {
	return &CommentPoster{
		githubClient: client,
	}
}

// ReviewComment represents a code review comment
type ReviewComment struct {
	FilePath   string
	Line       int
	Body       string
	IssueID    string
	Severity   string
	CommitSHA  string
	Side       string // "RIGHT" or "LEFT"
}

// PostReview posts a complete code review with all comments
func (cp *CommentPoster) PostReview(ctx context.Context, owner, repo string, prNumber int, issues *PrioritizedIssues, commitSHA string) error {
	if len(issues.All) == 0 {
		// Post approval if no issues
		return cp.postApprovalReview(ctx, owner, repo, prNumber, commitSHA)
	}

	// Create review comments for each issue
	comments := make([]*github.DraftReviewComment, 0, len(issues.All))

	for _, issue := range issues.All {
		body := formatIssueComment(issue)

		comment := &github.DraftReviewComment{
			Path: github.String(issue.FilePath),
			Line: github.Int(issue.Line),
			Body: github.String(body),
			Side: github.String("RIGHT"), // Comment on the new version
		}

		comments = append(comments, comment)
	}

	// Create the review
	reviewBody := formatReviewSummary(issues)

	review := &github.PullRequestReviewRequest{
		CommitID: github.String(commitSHA),
		Body:     github.String(reviewBody),
		Event:    github.String("COMMENT"), // Use COMMENT to avoid blocking
		Comments: comments,
	}

	_, _, err := cp.githubClient.PullRequests.CreateReview(ctx, owner, repo, prNumber, review)
	if err != nil {
		return fmt.Errorf("failed to create review: %w", err)
	}

	return nil
}

// PostInlineComment posts a single inline comment
func (cp *CommentPoster) PostInlineComment(ctx context.Context, owner, repo string, prNumber int, comment *ReviewComment) (int64, error) {
	ghComment := &github.PullRequestComment{
		Path:     github.String(comment.FilePath),
		Line:     github.Int(comment.Line),
		Body:     github.String(comment.Body),
		CommitID: github.String(comment.CommitSHA),
		Side:     github.String(comment.Side),
	}

	created, _, err := cp.githubClient.PullRequests.CreateComment(ctx, owner, repo, prNumber, ghComment)
	if err != nil {
		return 0, fmt.Errorf("failed to create comment: %w", err)
	}

	return created.GetID(), nil
}

// UpdateComment updates an existing review comment
func (cp *CommentPoster) UpdateComment(ctx context.Context, owner, repo string, commentID int64, newBody string) error {
	comment := &github.PullRequestComment{
		Body: github.String(newBody),
	}

	_, _, err := cp.githubClient.PullRequests.EditComment(ctx, owner, repo, commentID, comment)
	if err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	return nil
}

// PostSummaryComment posts a summary comment on the PR
func (cp *CommentPoster) PostSummaryComment(ctx context.Context, owner, repo string, prNumber int, summary string) error {
	comment := &github.IssueComment{
		Body: github.String(summary),
	}

	_, _, err := cp.githubClient.Issues.CreateComment(ctx, owner, repo, prNumber, comment)
	if err != nil {
		return fmt.Errorf("failed to post summary comment: %w", err)
	}

	return nil
}

// postApprovalReview posts an approval review when no issues found
func (cp *CommentPoster) postApprovalReview(ctx context.Context, owner, repo string, prNumber int, commitSHA string) error {
	review := &github.PullRequestReviewRequest{
		CommitID: github.String(commitSHA),
		Body:     github.String("✅ **Automated Code Review Passed**\n\nNo issues detected. Code looks good!"),
		Event:    github.String("APPROVE"),
	}

	_, _, err := cp.githubClient.PullRequests.CreateReview(ctx, owner, repo, prNumber, review)
	return err
}

// formatIssueComment formats a single issue as a comment
func formatIssueComment(issue *Issue) string {
	emoji := getSeverityEmoji(issue.Severity)

	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("%s **%s**: %s\n\n", emoji, issue.Severity, issue.Title))

	if issue.Description != "" && issue.Description != issue.Title {
		builder.WriteString("**Details:**\n")
		builder.WriteString(issue.Description)
		builder.WriteString("\n\n")
	}

	if issue.Suggestion != "" {
		builder.WriteString("**Suggestion:**\n")
		builder.WriteString(issue.Suggestion)
		builder.WriteString("\n\n")
	}

	if issue.CodeSnippet != "" {
		builder.WriteString("**Code:**\n```\n")
		builder.WriteString(issue.CodeSnippet)
		builder.WriteString("\n```\n\n")
	}

	builder.WriteString(fmt.Sprintf("_Category: %s | Automated review by GitHub Code Agent_", issue.Category))

	return builder.String()
}

// formatReviewSummary creates a summary of the review
func formatReviewSummary(issues *PrioritizedIssues) string {
	var builder strings.Builder

	builder.WriteString("🤖 **Automated Code Review Complete**\n\n")

	totalIssues := len(issues.All)
	if totalIssues == 0 {
		builder.WriteString("No issues detected. Code looks good!")
		return builder.String()
	}

	builder.WriteString(fmt.Sprintf("Found **%d** issue(s):\n\n", totalIssues))

	// Summary by severity
	if len(issues.Critical) > 0 {
		builder.WriteString(fmt.Sprintf("- 🔴 **%d** Critical\n", len(issues.Critical)))
	}
	if len(issues.High) > 0 {
		builder.WriteString(fmt.Sprintf("- 🟠 **%d** High\n", len(issues.High)))
	}
	if len(issues.Medium) > 0 {
		builder.WriteString(fmt.Sprintf("- 🟡 **%d** Medium\n", len(issues.Medium)))
	}
	if len(issues.Low) > 0 {
		builder.WriteString(fmt.Sprintf("- 🔵 **%d** Low\n", len(issues.Low)))
	}

	builder.WriteString("\n")

	// Top issues
	if len(issues.Critical) > 0 || len(issues.High) > 0 {
		builder.WriteString("**⚠️ High Priority Issues:**\n\n")

		highPriority := append(issues.Critical, issues.High...)
		if len(highPriority) > 5 {
			highPriority = highPriority[:5]
		}

		for _, issue := range highPriority {
			emoji := getSeverityEmoji(issue.Severity)
			builder.WriteString(fmt.Sprintf("%s %s in `%s:%d`\n", emoji, issue.Title, issue.FilePath, issue.Line))
		}

		builder.WriteString("\n")
	}

	builder.WriteString("_Please review the inline comments for detailed feedback._")

	return builder.String()
}

// getSeverityEmoji returns an emoji for the severity level
func getSeverityEmoji(severity string) string {
	switch severity {
	case SeverityCritical:
		return "🔴"
	case SeverityHigh:
		return "🟠"
	case SeverityMedium:
		return "🟡"
	case SeverityLow:
		return "🔵"
	default:
		return "⚪"
	}
}
