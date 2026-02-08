package gitops

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/yourorg/github-code-agent/features/fixer"
	"github.com/yourorg/github-code-agent/pkg/constants"
)

// GitOperations handles all Git command operations
type GitOperations struct{}

// NewGitOperations creates a new GitOperations instance
func NewGitOperations() *GitOperations {
	return &GitOperations{}
}

// CreateBranch creates a new Git branch
func (g *GitOperations) CreateBranch(ctx context.Context, repoPath, baseBranch, newBranch string) (*BranchInfo, error) {
	// Checkout base branch
	if err := g.checkout(ctx, repoPath, baseBranch); err != nil {
		return nil, fmt.Errorf("failed to checkout base branch: %w", err)
	}

	// Pull latest changes
	if err := g.pull(ctx, repoPath, baseBranch); err != nil {
		return nil, fmt.Errorf("failed to pull latest changes: %w", err)
	}

	// Create and checkout new branch
	if err := g.checkoutNewBranch(ctx, repoPath, newBranch); err != nil {
		return nil, fmt.Errorf("failed to create new branch: %w", err)
	}

	// Get current commit SHA
	sha, err := g.getCurrentSHA(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit SHA: %w", err)
	}

	return &BranchInfo{
		Name: newBranch,
		SHA:  sha,
	}, nil
}

// ApplyPatches applies code patches to files and commits them
func (g *GitOperations) ApplyPatches(ctx context.Context, repoPath string, patches []*fixer.CodePatch, commitMessage string) (*ApplyPatchesResult, error) {
	result := &ApplyPatchesResult{
		Success:      true,
		Applications: make([]*PatchApplication, 0, len(patches)),
		TotalPatches: len(patches),
	}

	// Apply each patch
	for _, patch := range patches {
		app := g.applyPatch(repoPath, patch)
		result.Applications = append(result.Applications, app)

		if !app.Success {
			result.Success = false
			result.FailureCount++
		} else {
			result.SuccessCount++
		}
	}

	// If any patches failed, don't commit
	if !result.Success {
		result.Error = "some patches failed to apply"
		return result, nil
	}

	// Stage all changes
	if err := g.stageAll(ctx, repoPath); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to stage changes: %v", err)
		return result, nil
	}

	// Commit changes
	commit, err := g.commit(ctx, repoPath, commitMessage)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to commit: %v", err)
		return result, nil
	}

	result.Commit = commit
	return result, nil
}

// PushBranch pushes a branch to remote
func (g *GitOperations) PushBranch(ctx context.Context, repoPath, branchName string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "push", "-u", "origin", branchName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push branch: %v - %s", err, string(output))
	}
	return nil
}

// CloneRepository clones a repository to a temporary directory
func (g *GitOperations) CloneRepository(ctx context.Context, repoURL, token string) (string, error) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "code-agent-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Add token to URL if provided
	cloneURL := repoURL
	if token != "" {
		// Convert https://github.com/user/repo to https://token@github.com/user/repo
		cloneURL = strings.Replace(repoURL, "https://", fmt.Sprintf("https://%s@", token), 1)
	}

	// Clone repository
	cmd := exec.CommandContext(ctx, "git", "clone", cloneURL, tmpDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to clone repository: %v - %s", err, string(output))
	}

	return tmpDir, nil
}

// Private helper methods

func (g *GitOperations) checkout(ctx context.Context, repoPath, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", branch)
	return cmd.Run()
}

func (g *GitOperations) pull(ctx context.Context, repoPath, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "pull", "origin", branch)
	return cmd.Run()
}

func (g *GitOperations) checkoutNewBranch(ctx context.Context, repoPath, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", "-b", branch)
	return cmd.Run()
}

func (g *GitOperations) getCurrentSHA(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (g *GitOperations) applyPatch(repoPath string, patch *fixer.CodePatch) *PatchApplication {
	app := &PatchApplication{
		FilePath: patch.FilePath,
		PatchID:  patch.IssueID,
		Success:  false,
	}

	// Full file path
	fullPath := repoPath + "/" + patch.FilePath

	// Write fixed code to file
	if err := os.WriteFile(fullPath, []byte(patch.FixedCode), 0644); err != nil {
		app.Error = fmt.Sprintf("failed to write file: %v", err)
		return app
	}

	app.Success = true
	return app
}

func (g *GitOperations) stageAll(ctx context.Context, repoPath string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "add", ".")
	return cmd.Run()
}

func (g *GitOperations) commit(ctx context.Context, repoPath, message string) (*CommitResult, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "commit", "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%v - %s", err, string(output))
	}

	// Get commit SHA
	sha, err := g.getCurrentSHA(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit SHA: %v", err)
	}

	// Count files changed (simple implementation)
	filesChanged := strings.Count(string(output), "file")

	return &CommitResult{
		SHA:          sha,
		Message:      message,
		Author:       constants.GitCommitAuthor,
		Timestamp:    time.Now(),
		FilesChanged: filesChanged,
	}, nil
}
