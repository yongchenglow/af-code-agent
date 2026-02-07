package mocks

import (
	"context"

	"github.com/google/go-github/v57/github"
)

// MockGitHubClient provides a mock GitHub client for testing
type MockGitHubClient struct {
	// PR data
	PRData       *github.PullRequest
	PRFiles      []*github.CommitFile
	PRErr        error

	// Comment data
	Comments     []*github.PullRequestComment
	CommentErr   error

	// CI Status
	CIStatus     string
	CIErr        error

	// Created resources
	CreatedBranch string
	CreatedPR     *github.PullRequest
	CreatedComments []string
}

// GetPR mocks getting a pull request
func (m *MockGitHubClient) GetPR(ctx context.Context, owner, repo string, number int) (*github.PullRequest, error) {
	if m.PRErr != nil {
		return nil, m.PRErr
	}
	return m.PRData, nil
}

// ListFiles mocks listing PR files
func (m *MockGitHubClient) ListFiles(ctx context.Context, owner, repo string, number int) ([]*github.CommitFile, error) {
	if m.PRErr != nil {
		return nil, m.PRErr
	}
	return m.PRFiles, nil
}

// CreateComment mocks creating a PR comment
func (m *MockGitHubClient) CreateComment(ctx context.Context, owner, repo string, number int, comment *github.PullRequestComment) (*github.PullRequestComment, error) {
	if m.CommentErr != nil {
		return nil, m.CommentErr
	}

	m.CreatedComments = append(m.CreatedComments, comment.GetBody())
	return comment, nil
}

// GetCombinedStatus mocks getting CI status
func (m *MockGitHubClient) GetCombinedStatus(ctx context.Context, owner, repo, ref string) (string, error) {
	if m.CIErr != nil {
		return "", m.CIErr
	}
	return m.CIStatus, nil
}

// NewMockGitHubClient creates a new mock GitHub client with default data
func NewMockGitHubClient() *MockGitHubClient {
	prTitle := "Test PR"
	prBody := "Test PR body"
	prNumber := 123
	prState := "open"
	prBranch := "feature-branch"
	prBaseBranch := "main"
	prAuthor := "testauthor"

	return &MockGitHubClient{
		PRData: &github.PullRequest{
			Number: &prNumber,
			Title:  &prTitle,
			Body:   &prBody,
			State:  &prState,
			Head: &github.PullRequestBranch{
				Ref: &prBranch,
			},
			Base: &github.PullRequestBranch{
				Ref: &prBaseBranch,
			},
			User: &github.User{
				Login: &prAuthor,
			},
		},
		PRFiles: []*github.CommitFile{
			{
				Filename:  github.String("test.go"),
				Status:    github.String("modified"),
				Additions: github.Int(10),
				Deletions: github.Int(5),
				Changes:   github.Int(15),
				Patch:     github.String("@@ -1,5 +1,10 @@\n func test() {\n+  // new code\n }"),
			},
		},
		CIStatus: "success",
		Comments: []*github.PullRequestComment{},
		CreatedComments: []string{},
	}
}
