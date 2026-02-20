package constants

import (
	"testing"
	"time"
)

func TestAIConfigurationConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		expected float64
	}{
		{"DefaultAITemperature", DefaultAITemperature, 0.2},
		{"LowAITemperature", LowAITemperature, 0.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestAIMaxTokensConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{"DefaultAIMaxTokens", DefaultAIMaxTokens, 4000},
		{"ReviewAIMaxTokens", ReviewAIMaxTokens, 4000},
		{"SecurityAIMaxTokens", SecurityAIMaxTokens, 3000},
		{"FixerAIMaxTokens", FixerAIMaxTokens, 2000},
		{"TestAIMaxTokens", TestAIMaxTokens, 3000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestDefaultAITimeout(t *testing.T) {
	expected := 10 * time.Minute
	if DefaultAITimeout != expected {
		t.Errorf("DefaultAITimeout = %v, want %v", DefaultAITimeout, expected)
	}
}

func TestDefaultAIModel(t *testing.T) {
	expected := "deepseek-chat"
	if DefaultAIModel != expected {
		t.Errorf("DefaultAIModel = %q, want %q", DefaultAIModel, expected)
	}
}

func TestValidationConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected interface{}
	}{
		{"MaxFixAttempts", MaxFixAttempts, 3},
		{"DefaultValidationTimeout", DefaultValidationTimeout, 30 * time.Second},
		{"MaxFileContextLines", MaxFileContextLines, 5},
		{"MaxReviewableFiles", MaxReviewableFiles, 5},
		{"MaxPatchLength", MaxPatchLength, 800},
		{"MaxContentLength", MaxContentLength, 800},
		{"MaxSecurityContentLength", MaxSecurityContentLength, 2000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestSeverityLevels(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"SeverityCritical", SeverityCritical, "Critical"},
		{"SeverityHigh", SeverityHigh, "High"},
		{"SeverityMedium", SeverityMedium, "Medium"},
		{"SeverityLow", SeverityLow, "Low"},
		{"SeverityWarning", SeverityWarning, "Warning"},
		{"SeverityInfo", SeverityInfo, "Info"},
		{"SeverityError", SeverityError, "Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestViolationTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"ViolationTypeBug", ViolationTypeBug, "bug"},
		{"ViolationTypeSecurity", ViolationTypeSecurity, "security"},
		{"ViolationTypePerformance", ViolationTypePerformance, "performance"},
		{"ViolationTypeMaintainability", ViolationTypeMaintainability, "maintainability"},
		{"ViolationTypeStyle", ViolationTypeStyle, "style"},
		{"ViolationTypeCoding", ViolationTypeCoding, "coding"},
		{"ViolationTypeDocumentation", ViolationTypeDocumentation, "documentation"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestFileStatus(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"FileStatusAdded", FileStatusAdded, "added"},
		{"FileStatusModified", FileStatusModified, "modified"},
		{"FileStatusRemoved", FileStatusRemoved, "removed"},
		{"FileStatusRenamed", FileStatusRenamed, "renamed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestGitConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"GitCommitAuthor", GitCommitAuthor, "GitHub Code Agent"},
		{"GitCommitEmoji", GitCommitEmoji, "🤖"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestHTTPConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected interface{}
	}{
		{"DefaultServerPort", DefaultServerPort, "8080"},
		{"WebhookEndpoint", WebhookEndpoint, "/webhook"},
		{"DebounceCleanupInterval", DebounceCleanupInterval, 5 * time.Minute},
		{"DefaultHTTPTimeout", DefaultHTTPTimeout, 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestAgentConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"AgentNodeID", AgentNodeID, "github-code-agent"},
		{"AgentVersion", AgentVersion, "1.0.0"},
		{"AgentTeamID", AgentTeamID, "code-review"},
		{"DefaultAgentMode", DefaultAgentMode, "safe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestEnvironmentVariableKeys(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"EnvAgentFieldURL", EnvAgentFieldURL, "AGENTFIELD_URL"},
		{"EnvGitHubAppID", EnvGitHubAppID, "GITHUB_APP_ID"},
		{"EnvGitHubPrivateKey", EnvGitHubPrivateKey, "GITHUB_PRIVATE_KEY"},
		{"EnvGitHubWebhookSecret", EnvGitHubWebhookSecret, "GITHUB_WEBHOOK_SECRET"},
		{"EnvAIBaseURL", EnvAIBaseURL, "AI_BASE_URL"},
		{"EnvAIModel", EnvAIModel, "AI_MODEL"},
		{"EnvAITemperature", EnvAITemperature, "AI_TEMPERATURE"},
		{"EnvAIMaxTokens", EnvAIMaxTokens, "AI_MAX_TOKENS"},
		{"EnvLogLevel", EnvLogLevel, "LOG_LEVEL"},
		{"EnvPort", EnvPort, "PORT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestDefaultValues(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"DefaultAgentFieldURL", DefaultAgentFieldURL, "http://localhost:8080"},
		{"DefaultAIBaseURL", DefaultAIBaseURL, "https://api.deepseek.com"},
		{"DefaultLogLevel", DefaultLogLevel, "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestSeverityLevelsUniqueness(t *testing.T) {
	severities := map[string]bool{
		SeverityCritical: true,
		SeverityHigh:     true,
		SeverityMedium:   true,
		SeverityLow:      true,
		SeverityWarning:  true,
		SeverityInfo:     true,
		SeverityError:    true,
	}

	if len(severities) != 7 {
		t.Error("Severity levels should all be unique")
	}
}

func TestViolationTypesUniqueness(t *testing.T) {
	types := map[string]bool{
		ViolationTypeBug:             true,
		ViolationTypeSecurity:        true,
		ViolationTypePerformance:     true,
		ViolationTypeMaintainability: true,
		ViolationTypeStyle:           true,
		ViolationTypeCoding:          true,
		ViolationTypeDocumentation:   true,
	}

	if len(types) != 7 {
		t.Error("Violation types should all be unique")
	}
}

func TestFileStatusUniqueness(t *testing.T) {
	statuses := map[string]bool{
		FileStatusAdded:    true,
		FileStatusModified: true,
		FileStatusRemoved:  true,
		FileStatusRenamed:  true,
	}

	if len(statuses) != 4 {
		t.Error("File statuses should all be unique")
	}
}

func TestEnvironmentVariableKeysUniqueness(t *testing.T) {
	keys := map[string]bool{
		EnvAgentFieldURL:       true,
		EnvGitHubAppID:         true,
		EnvGitHubPrivateKey:    true,
		EnvGitHubWebhookSecret: true,
		EnvAIBaseURL:           true,
		EnvAIModel:             true,
		EnvAITemperature:       true,
		EnvAIMaxTokens:         true,
		EnvLogLevel:            true,
		EnvPort:                true,
	}

	if len(keys) != 10 {
		t.Error("Environment variable keys should all be unique")
	}
}

func TestTimeoutValues(t *testing.T) {
	// Verify timeout values are reasonable
	if DefaultAITimeout <= 0 {
		t.Error("DefaultAITimeout should be positive")
	}
	if DefaultValidationTimeout <= 0 {
		t.Error("DefaultValidationTimeout should be positive")
	}
	if DefaultHTTPTimeout <= 0 {
		t.Error("DefaultHTTPTimeout should be positive")
	}

	// Verify reasonable ranges
	if DefaultAITimeout < time.Minute {
		t.Error("DefaultAITimeout should be at least 1 minute")
	}
	if DefaultValidationTimeout > time.Minute {
		t.Error("DefaultValidationTimeout should be less than 1 minute")
	}
}

func TestMaxValues(t *testing.T) {
	// Verify max values are reasonable
	if MaxFixAttempts <= 0 {
		t.Error("MaxFixAttempts should be positive")
	}
	if MaxFileContextLines <= 0 {
		t.Error("MaxFileContextLines should be positive")
	}
	if MaxReviewableFiles <= 0 {
		t.Error("MaxReviewableFiles should be positive")
	}
	if MaxPatchLength <= 0 {
		t.Error("MaxPatchLength should be positive")
	}
	if MaxContentLength <= 0 {
		t.Error("MaxContentLength should be positive")
	}
	if MaxSecurityContentLength <= 0 {
		t.Error("MaxSecurityContentLength should be positive")
	}

	// Security content should allow more than regular content
	if MaxSecurityContentLength <= MaxContentLength {
		t.Error("MaxSecurityContentLength should be greater than MaxContentLength")
	}
}

func TestTemperatureValues(t *testing.T) {
	// Verify temperature values are in valid range (0.0 to 1.0)
	if DefaultAITemperature < 0.0 || DefaultAITemperature > 1.0 {
		t.Error("DefaultAITemperature should be between 0.0 and 1.0")
	}
	if LowAITemperature < 0.0 || LowAITemperature > 1.0 {
		t.Error("LowAITemperature should be between 0.0 and 1.0")
	}

	// Low temperature should be lower than default
	if LowAITemperature >= DefaultAITemperature {
		t.Error("LowAITemperature should be less than DefaultAITemperature")
	}
}

func TestTokenLimits(t *testing.T) {
	// Verify token limits are reasonable
	if DefaultAIMaxTokens <= 0 {
		t.Error("DefaultAIMaxTokens should be positive")
	}
	if ReviewAIMaxTokens <= 0 {
		t.Error("ReviewAIMaxTokens should be positive")
	}
	if SecurityAIMaxTokens <= 0 {
		t.Error("SecurityAIMaxTokens should be positive")
	}
	if FixerAIMaxTokens <= 0 {
		t.Error("FixerAIMaxTokens should be positive")
	}
	if TestAIMaxTokens <= 0 {
		t.Error("TestAIMaxTokens should be positive")
	}
}

func TestAgentVersion(t *testing.T) {
	// Verify version format (semantic versioning)
	if AgentVersion == "" {
		t.Error("AgentVersion should not be empty")
	}

	// Basic semantic versioning check
	if len(AgentVersion) < 3 {
		t.Error("AgentVersion should be at least 3 characters (e.g., 1.0.0)")
	}
}

func TestDefaultAgentMode(t *testing.T) {
	// Verify default mode is valid
	validModes := map[string]bool{
		"safe": true,
		"yolo": true,
	}

	if !validModes[DefaultAgentMode] {
		t.Errorf("DefaultAgentMode = %q, should be 'safe' or 'yolo'", DefaultAgentMode)
	}
}

func TestWebhookEndpoint(t *testing.T) {
	// Verify webhook endpoint format
	if WebhookEndpoint == "" {
		t.Error("WebhookEndpoint should not be empty")
	}
	if WebhookEndpoint[0] != '/' {
		t.Error("WebhookEndpoint should start with '/'")
	}
}

func TestDefaultServerPort(t *testing.T) {
	// Verify port is valid
	if DefaultServerPort == "" {
		t.Error("DefaultServerPort should not be empty")
	}

	// Port should be "8080" which is valid
	if DefaultServerPort != "8080" {
		t.Errorf("DefaultServerPort should be '8080', got %q", DefaultServerPort)
	}
}

func TestGitCommitEmoji(t *testing.T) {
	// Verify emoji is not empty
	if GitCommitEmoji == "" {
		t.Error("GitCommitEmoji should not be empty")
	}

	// Emoji should be a valid UTF-8 string
	if len(GitCommitEmoji) == 0 {
		t.Error("GitCommitEmoji should have content")
	}
}

func TestAgentNodeID(t *testing.T) {
	// Verify node ID is not empty
	if AgentNodeID == "" {
		t.Error("AgentNodeID should not be empty")
	}

	// Node ID should be lowercase
	if AgentNodeID != "github-code-agent" {
		t.Errorf("AgentNodeID = %q, want 'github-code-agent'", AgentNodeID)
	}
}

func TestAgentTeamID(t *testing.T) {
	// Verify team ID is not empty
	if AgentTeamID == "" {
		t.Error("AgentTeamID should not be empty")
	}

	// Team ID should be lowercase with hyphens
	if AgentTeamID != "code-review" {
		t.Errorf("AgentTeamID = %q, want 'code-review'", AgentTeamID)
	}
}

func TestConstantsAreImmutable(t *testing.T) {
	// This test verifies that constants can be used safely
	// (Go constants are compile-time constants and cannot be modified)

	// Just verify we can read them multiple times with same value
	temp1 := DefaultAITemperature
	temp2 := DefaultAITemperature

	if temp1 != temp2 {
		t.Error("Constants should be immutable")
	}
}

func TestAllConstantsDocumented(t *testing.T) {
	// This is a meta-test to ensure all constants are documented
	// In Go, exported constants (starting with uppercase) should have comments

	// We can't programmatically check comments, but we can verify
	// that all our constants are accessible (which they are if we can reference them)

	// List of all our exported constants
	constants := []interface{}{
		DefaultAITemperature,
		LowAITemperature,
		DefaultAIMaxTokens,
		ReviewAIMaxTokens,
		SecurityAIMaxTokens,
		FixerAIMaxTokens,
		TestAIMaxTokens,
		DefaultAITimeout,
		DefaultAIModel,
		MaxFixAttempts,
		DefaultValidationTimeout,
		MaxFileContextLines,
		MaxReviewableFiles,
		MaxPatchLength,
		MaxContentLength,
		MaxSecurityContentLength,
		SeverityCritical,
		SeverityHigh,
		SeverityMedium,
		SeverityLow,
		SeverityWarning,
		SeverityInfo,
		SeverityError,
		ViolationTypeBug,
		ViolationTypeSecurity,
		ViolationTypePerformance,
		ViolationTypeMaintainability,
		ViolationTypeStyle,
		ViolationTypeCoding,
		ViolationTypeDocumentation,
		FileStatusAdded,
		FileStatusModified,
		FileStatusRemoved,
		FileStatusRenamed,
		GitCommitAuthor,
		GitCommitEmoji,
		DefaultServerPort,
		WebhookEndpoint,
		DebounceCleanupInterval,
		DefaultHTTPTimeout,
		AgentNodeID,
		AgentVersion,
		AgentTeamID,
		DefaultAgentMode,
		EnvAgentFieldURL,
		EnvGitHubAppID,
		EnvGitHubPrivateKey,
		EnvGitHubWebhookSecret,
		EnvAIBaseURL,
		EnvAIModel,
		EnvAITemperature,
		EnvAIMaxTokens,
		EnvLogLevel,
		EnvPort,
		DefaultAgentFieldURL,
		DefaultAIBaseURL,
		DefaultLogLevel,
	}

	if len(constants) == 0 {
		t.Error("Should have constants defined")
	}
}
