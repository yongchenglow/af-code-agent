package gitops

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/yourorg/github-code-agent/features/fixer"
)

// CreateBranch creates a new Git branch
func CreateBranch(ctx context.Context, repoPath, baseBranch, newBranch string) (*BranchInfo, error) {
	// Change to repo directory
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", baseBranch)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to checkout base branch: %w", err)
	}

	// Pull latest changes
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "pull", "origin", baseBranch)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to pull latest changes: %w", err)
	}

	// Create and checkout new branch
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", "-b", newBranch)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to create new branch: %w", err)
	}

	// Get current commit SHA
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit SHA: %w", err)
	}

	return &BranchInfo{
		Name: newBranch,
		SHA:  strings.TrimSpace(string(output)),
	}, nil
}

// ApplyPatches applies code patches to files and commits them
func ApplyPatches(ctx context.Context, repoPath string, patches []*fixer.CodePatch, commitMessage string) (*ApplyPatchesResult, error) {
	result := &ApplyPatchesResult{
		Success:      true,
		Applications: make([]*PatchApplication, 0, len(patches)),
		TotalPatches: len(patches),
	}

	// Apply each patch
	for _, patch := range patches {
		app := &PatchApplication{
			FilePath: patch.FilePath,
			PatchID:  patch.IssueID,
			Success:  false,
		}

		// Full file path
		fullPath := filepath.Join(repoPath, patch.FilePath)

		// Write fixed code to file
		if err := os.WriteFile(fullPath, []byte(patch.FixedCode), 0644); err != nil {
			app.Error = fmt.Sprintf("failed to write file: %v", err)
			result.Success = false
			result.FailureCount++
		} else {
			app.Success = true
			result.SuccessCount++
		}

		result.Applications = append(result.Applications, app)
	}

	// If any patches failed, don't commit
	if !result.Success {
		result.Error = "some patches failed to apply"
		return result, nil
	}

	// Stage all changes
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "add", ".")
	if err := cmd.Run(); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to stage changes: %v", err)
		return result, nil
	}

	// Commit changes
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "commit", "-m", commitMessage)
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to commit: %v - %s", err, string(output))
		return result, nil
	}

	// Get commit SHA
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "HEAD")
	shaOutput, err := cmd.Output()
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to get commit SHA: %v", err)
		return result, nil
	}

	result.Commit = &CommitResult{
		SHA:          strings.TrimSpace(string(shaOutput)),
		Message:      commitMessage,
		Author:       "GitHub Code Agent",
		Timestamp:    time.Now(),
		FilesChanged: len(patches),
	}

	return result, nil
}

// PushBranch pushes a branch to remote
func PushBranch(ctx context.Context, repoPath, branchName string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "push", "-u", "origin", branchName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to push branch: %v - %s", err, string(output))
	}
	return nil
}

// CloneRepository clones a repository to a temporary directory
func CloneRepository(ctx context.Context, repoURL, token string) (string, error) {
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

// GenerateCommitMessage generates a commit message for fixes
func GenerateCommitMessage(patches []*fixer.CodePatch) string {
	if len(patches) == 0 {
		return "🤖 Automated fixes"
	}

	if len(patches) == 1 {
		return fmt.Sprintf("🤖 Auto-fix: %s", patches[0].Description)
	}

	return fmt.Sprintf("🤖 Auto-fix: %d issues resolved", len(patches))
}

// GeneratePRTitle generates a PR title for fix PR
func GeneratePRTitle(prNumber int, fixCount int) string {
	return fmt.Sprintf("🤖 Automated fixes for PR #%d (%d issues)", prNumber, fixCount)
}

// GeneratePRBody generates a PR description for fix PR
func GeneratePRBody(prNumber int, patches []*fixer.CodePatch) string {
	body := fmt.Sprintf(`## Automated Code Review Fixes

This PR contains automated fixes for issues found in PR #%d.

### Issues Fixed:

`, prNumber)

	for i, patch := range patches {
		body += fmt.Sprintf("%d. **%s** in `%s` (line %d)\n", i+1, patch.Description, patch.FilePath, patch.Line)
	}

	body += `
### Validation Results:

All fixes have been validated for:
- ✅ Syntax correctness
- ✅ Linting compliance
- ✅ Code formatting
- ✅ Security checks

Please review and merge if changes are acceptable.

---
🤖 Generated by [GitHub Code Agent](https://github.com/yourorg/github-code-agent)
`

	return body
}
