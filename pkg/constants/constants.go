package constants

import "time"

// AI Configuration Constants
const (
	DefaultAITemperature = 0.2
	LowAITemperature     = 0.1
	DefaultAIMaxTokens   = 4000
	ReviewAIMaxTokens    = 4000
	SecurityAIMaxTokens  = 3000
	FixerAIMaxTokens     = 2000
	TestAIMaxTokens      = 3000
	DefaultAITimeout     = 10 * time.Minute
	DefaultAIModel       = "deepseek-chat"
)

// Validation Constants
const (
	MaxFixAttempts           = 3
	DefaultValidationTimeout = 30 * time.Second
	MaxFileContextLines      = 5
	MaxReviewableFiles       = 5
	MaxPatchLength           = 800
	MaxContentLength         = 800
	MaxSecurityContentLength = 2000
)

// Severity Levels
const (
	SeverityCritical = "Critical"
	SeverityHigh     = "High"
	SeverityMedium   = "Medium"
	SeverityLow      = "Low"
	SeverityWarning  = "Warning"
	SeverityInfo     = "Info"
	SeverityError    = "Error"
)

// Violation Types
const (
	ViolationTypeBug             = "bug"
	ViolationTypeSecurity        = "security"
	ViolationTypePerformance     = "performance"
	ViolationTypeMaintainability = "maintainability"
	ViolationTypeStyle           = "style"
	ViolationTypeCoding          = "coding"
	ViolationTypeDocumentation   = "documentation"
)

// File Status
const (
	FileStatusAdded    = "added"
	FileStatusModified = "modified"
	FileStatusRemoved  = "removed"
	FileStatusRenamed  = "renamed"
)

// Git Constants
const (
	GitCommitAuthor = "GitHub Code Agent"
	GitCommitEmoji  = "🤖"
)

// HTTP Constants
const (
	DefaultServerPort       = "8080"
	WebhookEndpoint         = "/webhook"
	DebounceCleanupInterval = 5 * time.Minute
	DefaultHTTPTimeout      = 30 * time.Second
)

// Agent Configuration
const (
	AgentNodeID      = "github-code-agent"
	AgentVersion     = "1.0.0"
	AgentTeamID      = "code-review"
	DefaultAgentMode = "safe"
)

// Environment Variable Keys
const (
	EnvAgentFieldURL       = "AGENTFIELD_URL"
	EnvGitHubAppID         = "GITHUB_APP_ID"
	EnvGitHubPrivateKey    = "GITHUB_PRIVATE_KEY"
	EnvGitHubWebhookSecret = "GITHUB_WEBHOOK_SECRET"
	EnvAIBaseURL           = "AI_BASE_URL"
	EnvAIModel             = "AI_MODEL"
	EnvAITemperature       = "AI_TEMPERATURE"
	EnvAIMaxTokens         = "AI_MAX_TOKENS"
	EnvLogLevel            = "LOG_LEVEL"
	EnvPort                = "PORT"
)

// Default Values
const (
	DefaultAgentFieldURL = "http://localhost:8080"
	DefaultAIBaseURL     = "https://api.deepseek.com"
	DefaultLogLevel      = "info"
)
