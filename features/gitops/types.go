package gitops

import "time"

// BranchInfo represents Git branch information
type BranchInfo struct {
	Name string `json:"name"`
	SHA  string `json:"sha"`
}

// CommitResult represents the result of a commit operation
type CommitResult struct {
	SHA       string    `json:"sha"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Timestamp time.Time `json:"timestamp"`
	FilesChanged int    `json:"files_changed"`
}

// PatchApplication represents a single patch being applied
type PatchApplication struct {
	FilePath    string `json:"file_path"`
	PatchID     string `json:"patch_id"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}

// ReviewComment represents a code review comment
type ReviewComment struct {
	FilePath  string `json:"file_path"`
	Line      int    `json:"line"`
	Body      string `json:"body"`
	IssueID   string `json:"issue_id"`
	Severity  string `json:"severity"`
	FixCommit string `json:"fix_commit,omitempty"` // Commit SHA (YOLO mode)
	FixPR     int    `json:"fix_pr,omitempty"`     // PR number (Safe mode)
}

// PullRequestInfo represents GitHub PR information
type PullRequestInfo struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	HeadBranch  string `json:"head_branch"`
	BaseBranch  string `json:"base_branch"`
	State       string `json:"state"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
}

// ApplyPatchesResult represents the result of applying multiple patches
type ApplyPatchesResult struct {
	Success      bool                `json:"success"`
	Commit       *CommitResult       `json:"commit,omitempty"`
	Applications []*PatchApplication `json:"applications"`
	TotalPatches int                 `json:"total_patches"`
	SuccessCount int                 `json:"success_count"`
	FailureCount int                 `json:"failure_count"`
	Error        string              `json:"error,omitempty"`
}

// OperationMode represents the operating mode (YOLO or Safe)
type OperationMode string

const (
	// YOLOMode pushes fixes directly to the source branch
	YOLOMode OperationMode = "yolo"
	// SafeMode creates a new PR with fixes
	SafeMode OperationMode = "safe"
)

// WorkflowResult represents the complete workflow execution result
type WorkflowResult struct {
	Mode            OperationMode `json:"mode"`
	Success         bool          `json:"success"`
	IssuesReviewed  int           `json:"issues_reviewed"`
	FixesApplied    int           `json:"fixes_applied"`
	CommitSHA       string        `json:"commit_sha,omitempty"`       // YOLO mode
	FixPR           int           `json:"fix_pr,omitempty"`           // Safe mode
	FixPRURL        string        `json:"fix_pr_url,omitempty"`       // Safe mode
	CommentsPosted  int           `json:"comments_posted"`
	CommentsUpdated int           `json:"comments_updated"`
	Error           string        `json:"error,omitempty"`
}
