package config

import "time"

// Config represents the complete agent configuration
type Config struct {
	Agent         AgentConfig         `yaml:"agent"`
	Webhooks      WebhooksConfig      `yaml:"webhooks"`
	Review        ReviewConfig        `yaml:"review"`
	Validation    ValidationConfig    `yaml:"validation"`
	Standards     StandardsConfig     `yaml:"standards"`
	Languages     LanguagesConfig     `yaml:"languages"`
	AI            AIConfig            `yaml:"ai"`
	Notifications NotificationConfig  `yaml:"notifications"`
}

// AgentConfig contains agent-level settings
type AgentConfig struct {
	Enabled bool   `yaml:"enabled"`
	Mode    string `yaml:"mode"` // "yolo" or "safe"
}

// WebhooksConfig configures webhook handling
type WebhooksConfig struct {
	Triggers []string `yaml:"triggers"`
	WaitForCI bool   `yaml:"wait_for_ci"`
	DebounceSeconds int `yaml:"debounce_seconds"`
}

// ReviewConfig configures code review behavior
type ReviewConfig struct {
	AutoReview        bool     `yaml:"auto_review"`
	AutoFix           bool     `yaml:"auto_fix"`
	SeverityThreshold string   `yaml:"severity_threshold"`
	IgnorePaths       []string `yaml:"ignore_paths"`
}

// ValidationConfig configures the fix validation loop
type ValidationConfig struct {
	Enabled         bool     `yaml:"enabled"`
	MaxFixAttempts  int      `yaml:"max_fix_attempts"`
	Checks          []string `yaml:"checks"`
	TimeoutSeconds  int      `yaml:"timeout_seconds"`
	AutoFormat      bool     `yaml:"auto_format"`
}

// StandardsConfig defines coding standards
type StandardsConfig struct {
	Coding        CodingStandards        `yaml:"coding"`
	Documentation DocumentationStandards `yaml:"documentation"`
	Security      SecurityStandards      `yaml:"security"`
}

// CodingStandards defines code style rules
type CodingStandards struct {
	MaxLineLength       int                       `yaml:"max_line_length"`
	MaxFunctionLength   int                       `yaml:"max_function_length"`
	MaxComplexity       int                       `yaml:"max_complexity"`
	NamingConventions   map[string]string         `yaml:"naming_conventions"`
}

// DocumentationStandards defines documentation requirements
type DocumentationStandards struct {
	RequireDocstrings  bool   `yaml:"require_docstrings"`
	DocstringStyle     string `yaml:"docstring_style"`
	RequireTypeHints   bool   `yaml:"require_type_hints"`
	RequireModuleDocs  bool   `yaml:"require_module_docs"`
}

// SecurityStandards defines security checks
type SecurityStandards struct {
	CheckDependencies bool `yaml:"check_dependencies"`
	CheckSecrets      bool `yaml:"check_secrets"`
	OWASPChecks       bool `yaml:"owasp_checks"`
}

// LanguagesConfig contains language-specific settings
type LanguagesConfig struct {
	Python     *LanguageSettings `yaml:"python,omitempty"`
	JavaScript *LanguageSettings `yaml:"javascript,omitempty"`
	Go         *LanguageSettings `yaml:"go,omitempty"`
}

// LanguageSettings defines settings for a specific language
type LanguageSettings struct {
	Linters          []string `yaml:"linters"`
	MinTestCoverage  int      `yaml:"min_test_coverage,omitempty"`
	Frameworks       []string `yaml:"frameworks,omitempty"`
}

// AIConfig configures AI model defaults (actual AI is handled by AgentField SDK)
// These are used as fallback defaults if environment variables are not set
type AIConfig struct {
	Model       string  `yaml:"model"`        // Default model name
	Temperature float64 `yaml:"temperature"`  // Default temperature
	MaxTokens   int     `yaml:"max_tokens"`   // Default max tokens
}

// NotificationConfig configures notifications
type NotificationConfig struct {
	OnReviewComplete bool `yaml:"on_review_complete"`
	OnFixesApplied   bool `yaml:"on_fixes_applied"`
	MentionAuthor    bool `yaml:"mention_author"`
}

// EnvironmentConfig holds environment-based configuration
type EnvironmentConfig struct {
	AgentFieldURL        string
	GitHubAppID          string
	GitHubPrivateKeyPath string
	GitHubWebhookSecret  string
	AIAPIKey             string
	AIBaseURL            string
	AIModel              string
	AITemperature        float64
	AIMaxTokens          int
	LogLevel             string
	Port                 string
}

// GetTimeout returns the validation timeout as a duration
func (v *ValidationConfig) GetTimeout() time.Duration {
	if v.TimeoutSeconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(v.TimeoutSeconds) * time.Second
}

// GetMaxAttempts returns the max fix attempts with a default
func (v *ValidationConfig) GetMaxAttempts() int {
	if v.MaxFixAttempts <= 0 {
		return 3
	}
	return v.MaxFixAttempts
}

// GetDebounceTimeout returns the debounce timeout as a duration
func (w *WebhooksConfig) GetDebounceTimeout() time.Duration {
	if w.DebounceSeconds <= 0 {
		return 0
	}
	return time.Duration(w.DebounceSeconds) * time.Second
}
