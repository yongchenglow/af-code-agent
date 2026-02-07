package reviewer

// Issue represents a code review issue
type Issue struct {
	ID          string `json:"id"`
	FilePath    string `json:"file_path"`
	Line        int    `json:"line"`
	EndLine     int    `json:"end_line,omitempty"`
	Column      int    `json:"column,omitempty"`
	Severity    string `json:"severity"` // Critical, High, Medium, Low
	Category    string `json:"category"` // bug, security, performance, maintainability, style
	Title       string `json:"title"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion,omitempty"`
	CodeSnippet string `json:"code_snippet,omitempty"`
}

// ReviewReport represents the complete review report
type ReviewReport struct {
	Issues        []*Issue          `json:"issues"`
	Summary       string            `json:"summary"`
	TotalIssues   int               `json:"total_issues"`
	IssuesBySeverity map[string]int `json:"issues_by_severity"`
	IssuesByCategory map[string]int `json:"issues_by_category"`
	Model         string            `json:"model"`
}

// SecurityIssue represents a security vulnerability
type SecurityIssue struct {
	ID          string   `json:"id"`
	FilePath    string   `json:"file_path"`
	Line        int      `json:"line"`
	Severity    string   `json:"severity"` // Critical, High, Medium, Low
	Type        string   `json:"type"`     // sql_injection, xss, secrets, etc.
	Title       string   `json:"title"`
	Description string   `json:"description"`
	CWE         string   `json:"cwe,omitempty"`        // Common Weakness Enumeration ID
	OWASP       string   `json:"owasp,omitempty"`      // OWASP Top 10 category
	Remediation string   `json:"remediation,omitempty"`
	References  []string `json:"references,omitempty"`
}

// ReviewOptions configures the review behavior
type ReviewOptions struct {
	IncludeSecurity     bool
	IncludePerformance  bool
	IncludeMaintainability bool
	IncludeStyle        bool
	MaxIssuesPerFile    int
	FocusOnChangedLines bool
}

// Severity levels
const (
	SeverityCritical = "Critical"
	SeverityHigh     = "High"
	SeverityMedium   = "Medium"
	SeverityLow      = "Low"
)

// Categories
const (
	CategoryBug             = "bug"
	CategorySecurity        = "security"
	CategoryPerformance     = "performance"
	CategoryMaintainability = "maintainability"
	CategoryStyle           = "style"
)
