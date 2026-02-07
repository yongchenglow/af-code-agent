package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Agent: AgentConfig{
			Enabled: true,
			Mode:    "safe",
		},
		Webhooks: WebhooksConfig{
			Triggers: []string{
				"pull_request.opened",
				"pull_request.synchronize",
				"check_suite.completed",
			},
			WaitForCI:       true,
			DebounceSeconds: 30,
		},
		Review: ReviewConfig{
			AutoReview:        true,
			AutoFix:           true,
			SeverityThreshold: "medium",
			IgnorePaths: []string{
				"*.md",
				"docs/**",
				"tests/fixtures/**",
			},
		},
		Validation: ValidationConfig{
			Enabled:        true,
			MaxFixAttempts: 3,
			Checks: []string{
				"syntax",
				"linting",
				"formatting",
				"security",
			},
			TimeoutSeconds: 30,
			AutoFormat:     true,
		},
		Standards: StandardsConfig{
			Coding: CodingStandards{
				MaxLineLength:     100,
				MaxFunctionLength: 50,
				MaxComplexity:     10,
				NamingConventions: map[string]string{
					"functions": "snake_case",
					"classes":   "PascalCase",
					"constants": "UPPER_SNAKE_CASE",
				},
			},
			Documentation: DocumentationStandards{
				RequireDocstrings: true,
				DocstringStyle:    "google",
				RequireTypeHints:  true,
				RequireModuleDocs: true,
			},
			Security: SecurityStandards{
				CheckDependencies: true,
				CheckSecrets:      true,
				OWASPChecks:       true,
			},
		},
		AI: AIConfig{
			Model:       "deepseek-chat",
			Temperature: 0.2,
			MaxTokens:   4000,
		},
		Notifications: NotificationConfig{
			OnReviewComplete: true,
			OnFixesApplied:   true,
			MentionAuthor:    true,
		},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	// Start with defaults
	cfg := DefaultConfig()

	// If file doesn't exist, return defaults
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// LoadEnvironmentConfig loads configuration from environment variables
func LoadEnvironmentConfig() (*EnvironmentConfig, error) {
	temp, err := strconv.ParseFloat(getEnvOrDefault("AI_TEMPERATURE", "0.2"), 64)
	if err != nil {
		temp = 0.2
	}

	maxTokens, err := strconv.Atoi(getEnvOrDefault("AI_MAX_TOKENS", "4000"))
	if err != nil {
		maxTokens = 4000
	}

	return &EnvironmentConfig{
		AgentFieldURL:       getEnvOrDefault("AGENTFIELD_URL", "http://localhost:8080"),
		GitHubToken:         os.Getenv("GITHUB_TOKEN"),
		GitHubWebhookSecret: os.Getenv("GITHUB_WEBHOOK_SECRET"),
		AIAPIKey:            os.Getenv("AI_API_KEY"),
		AIBaseURL:           getEnvOrDefault("AI_BASE_URL", "https://api.deepseek.com"),
		AIModel:             getEnvOrDefault("AI_MODEL", "deepseek-chat"),
		AITemperature:       temp,
		AIMaxTokens:         maxTokens,
		LogLevel:            getEnvOrDefault("LOG_LEVEL", "info"),
		Port:                getEnvOrDefault("PORT", "8080"),
	}, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Agent.Mode != "yolo" && c.Agent.Mode != "safe" {
		return fmt.Errorf("invalid agent mode: %s (must be 'yolo' or 'safe')", c.Agent.Mode)
	}

	if c.Validation.MaxFixAttempts < 1 || c.Validation.MaxFixAttempts > 10 {
		return fmt.Errorf("max_fix_attempts must be between 1 and 10")
	}

	if c.AI.Temperature < 0 || c.AI.Temperature > 2 {
		return fmt.Errorf("AI temperature must be between 0 and 2")
	}

	return nil
}

// Validate validates environment configuration
func (e *EnvironmentConfig) Validate() error {
	if e.GitHubToken == "" {
		return fmt.Errorf("GITHUB_TOKEN is required")
	}

	if e.GitHubWebhookSecret == "" {
		return fmt.Errorf("GITHUB_WEBHOOK_SECRET is required")
	}

	if e.AIAPIKey == "" {
		return fmt.Errorf("AI_API_KEY is required")
	}

	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
