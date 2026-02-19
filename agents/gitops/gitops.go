package gitops

import (
	"context"

	"github.com/yourorg/github-code-agent/agents/fixer"
)

var (
	gitOps = NewGitOperations()
	msgGen = NewMessageGenerator()
)

// CreateBranch creates a new Git branch.
//
// Deprecated: Use GitOperations.CreateBranch instead.
func CreateBranch(ctx context.Context, repoPath, baseBranch, newBranch string) (*BranchInfo, error) {
	return gitOps.CreateBranch(ctx, repoPath, baseBranch, newBranch)
}

// ApplyPatches applies code patches to files and commits them.
//
// Deprecated: Use GitOperations.ApplyPatches instead.
func ApplyPatches(ctx context.Context, repoPath string, patches []*fixer.CodePatch, commitMessage string) (*ApplyPatchesResult, error) {
	return gitOps.ApplyPatches(ctx, repoPath, patches, commitMessage)
}

// PushBranch pushes a branch to remote.
//
// Deprecated: Use GitOperations.PushBranch instead.
func PushBranch(ctx context.Context, repoPath, branchName string) error {
	return gitOps.PushBranch(ctx, repoPath, branchName)
}

// CloneRepository clones a repository to a temporary directory.
//
// Deprecated: Use GitOperations.CloneRepository instead.
func CloneRepository(ctx context.Context, repoURL, token string) (string, error) {
	return gitOps.CloneRepository(ctx, repoURL, token)
}

// GenerateCommitMessage generates a commit message for fixes.
//
// Deprecated: Use MessageGenerator.GenerateCommitMessage instead.
func GenerateCommitMessage(patches []*fixer.CodePatch) string {
	return msgGen.GenerateCommitMessage(patches)
}

// GeneratePRTitle generates a PR title for fix PR.
//
// Deprecated: Use MessageGenerator.GeneratePRTitle instead.
func GeneratePRTitle(prNumber int, fixCount int) string {
	return msgGen.GeneratePRTitle(prNumber, fixCount)
}

// GeneratePRBody generates a PR description for fix PR.
//
// Deprecated: Use MessageGenerator.GeneratePRBody instead.
func GeneratePRBody(prNumber int, patches []*fixer.CodePatch) string {
	return msgGen.GeneratePRBody(prNumber, patches)
}
