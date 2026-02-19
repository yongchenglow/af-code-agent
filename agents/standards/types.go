package standards

// ValidationReport contains the results of standards validation
type ValidationReport struct {
	Violations       []*Violation   `json:"violations"`
	TotalViolations  int            `json:"total_violations"`
	ViolationsByType map[string]int `json:"violations_by_type"`
	PassedChecks     []string       `json:"passed_checks"`
	FailedChecks     []string       `json:"failed_checks"`
	Summary          string         `json:"summary"`
}

// Violation represents a standards violation
type Violation struct {
	ID          string `json:"id"`
	FilePath    string `json:"file_path"`
	Line        int    `json:"line"`
	Column      int    `json:"column,omitempty"`
	Type        string `json:"type"`     // coding, documentation, security
	Rule        string `json:"rule"`     // specific rule violated
	Severity    string `json:"severity"` // Error, Warning, Info
	Message     string `json:"message"`
	Suggestion  string `json:"suggestion,omitempty"`
	AutoFixable bool   `json:"auto_fixable"`
}

// Rule represents a validation rule
type Rule struct {
	ID          string
	Name        string
	Description string
	Type        string // coding, documentation, security
	Severity    string
	Enabled     bool
	Validator   func(context RuleContext) ([]*Violation, error)
}

// RuleContext provides context for rule validation
type RuleContext struct {
	FilePath   string
	Content    string
	Language   string
	LineNumber int
	Config     interface{}
}

// Violation types
const (
	ViolationTypeCoding        = "coding"
	ViolationTypeDocumentation = "documentation"
	ViolationTypeSecurity      = "security"
	ViolationTypeArchitecture  = "architecture"
)

// Severity levels
const (
	SeverityError   = "Error"
	SeverityWarning = "Warning"
	SeverityInfo    = "Info"
)
