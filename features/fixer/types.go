package fixer

import "time"

// CodePatch represents a code fix patch
type CodePatch struct {
	IssueID     string    `json:"issue_id"`
	FilePath    string    `json:"file_path"`
	Language    string    `json:"language"`
	OriginalCode string   `json:"original_code"`
	FixedCode   string    `json:"fixed_code"`
	Description string    `json:"description"`
	Line        int       `json:"line"`
	EndLine     int       `json:"end_line,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ValidationResult represents the result of validating a fix
type ValidationResult struct {
	IsValid bool     `json:"is_valid"`
	Errors  []string `json:"errors"`
	Warnings []string `json:"warnings,omitempty"`
}

// FixAttempt tracks attempts to fix an issue
type FixAttempt struct {
	AttemptNumber int              `json:"attempt_number"`
	Patch         *CodePatch       `json:"patch"`
	Validation    *ValidationResult `json:"validation"`
	Timestamp     time.Time        `json:"timestamp"`
}

// FixResult represents the outcome of attempting to fix an issue
type FixResult struct {
	IssueID       string        `json:"issue_id"`
	Success       bool          `json:"success"`
	Patch         *CodePatch    `json:"patch,omitempty"`
	Attempts      []*FixAttempt `json:"attempts"`
	FinalError    string        `json:"final_error,omitempty"`
	TotalAttempts int           `json:"total_attempts"`
}

// BatchFixResult represents results from fixing multiple issues
type BatchFixResult struct {
	SuccessfulFixes []*CodePatch  `json:"successful_fixes"`
	FailedIssues    []string      `json:"failed_issues"`
	TotalIssues     int           `json:"total_issues"`
	SuccessCount    int           `json:"success_count"`
	FailureCount    int           `json:"failure_count"`
}

// ValidationConfig configures the validation behavior
type ValidationConfig struct {
	EnableSyntaxCheck    bool `json:"enable_syntax_check"`
	EnableLinting        bool `json:"enable_linting"`
	EnableFormatting     bool `json:"enable_formatting"`
	EnableSecurityScan   bool `json:"enable_security_scan"`
	AutoFormat           bool `json:"auto_format"`
	MaxAttempts          int  `json:"max_attempts"`
	TimeoutSeconds       int  `json:"timeout_seconds"`
}

// DefaultValidationConfig returns the default validation configuration
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		EnableSyntaxCheck:  true,
		EnableLinting:      true,
		EnableFormatting:   true,
		EnableSecurityScan: true,
		AutoFormat:         true,
		MaxAttempts:        3,
		TimeoutSeconds:     30,
	}
}

// LinterResult represents the output from a linter
type LinterResult struct {
	Tool     string   `json:"tool"`
	Passed   bool     `json:"passed"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// SecurityScanResult represents security scan findings
type SecurityScanResult struct {
	IssuesFound    int      `json:"issues_found"`
	CriticalCount  int      `json:"critical_count"`
	HighCount      int      `json:"high_count"`
	Descriptions   []string `json:"descriptions,omitempty"`
}
