package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Agent.Mode != "safe" {
		t.Errorf("Expected default mode 'safe', got %q", cfg.Agent.Mode)
	}

	if !cfg.Agent.Enabled {
		t.Error("Expected agent to be enabled by default")
	}

	if cfg.Validation.MaxFixAttempts != 3 {
		t.Errorf("Expected max fix attempts 3, got %d", cfg.Validation.MaxFixAttempts)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid safe mode",
			config: &Config{
				Agent:      AgentConfig{Mode: "safe"},
				Validation: ValidationConfig{MaxFixAttempts: 3},
				AI:         AIConfig{Temperature: 0.2},
			},
			wantErr: false,
		},
		{
			name: "valid yolo mode",
			config: &Config{
				Agent:      AgentConfig{Mode: "yolo"},
				Validation: ValidationConfig{MaxFixAttempts: 3},
				AI:         AIConfig{Temperature: 0.2},
			},
			wantErr: false,
		},
		{
			name: "invalid mode",
			config: &Config{
				Agent:      AgentConfig{Mode: "invalid"},
				Validation: ValidationConfig{MaxFixAttempts: 3},
				AI:         AIConfig{Temperature: 0.2},
			},
			wantErr: true,
		},
		{
			name: "invalid max attempts",
			config: &Config{
				Agent:      AgentConfig{Mode: "safe"},
				Validation: ValidationConfig{MaxFixAttempts: 0},
				AI:         AIConfig{Temperature: 0.2},
			},
			wantErr: true,
		},
		{
			name: "invalid temperature",
			config: &Config{
				Agent:      AgentConfig{Mode: "safe"},
				Validation: ValidationConfig{MaxFixAttempts: 3},
				AI:         AIConfig{Temperature: 3.0},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnvironmentConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *EnvironmentConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &EnvironmentConfig{
				GitHubToken:         "ghp_test_token",
				GitHubWebhookSecret: "secret",
			},
			wantErr: false,
		},
		{
			name: "missing token",
			config: &EnvironmentConfig{
				GitHubWebhookSecret: "secret",
			},
			wantErr: true,
		},
		{
			name: "missing webhook secret",
			config: &EnvironmentConfig{
				GitHubToken: "ghp_test_token",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetTimeout(t *testing.T) {
	cfg := &ValidationConfig{TimeoutSeconds: 60}
	timeout := cfg.GetTimeout()

	if timeout.Seconds() != 60 {
		t.Errorf("Expected timeout 60s, got %v", timeout)
	}

	// Test default
	cfg = &ValidationConfig{TimeoutSeconds: 0}
	timeout = cfg.GetTimeout()

	if timeout.Seconds() != 30 {
		t.Errorf("Expected default timeout 30s, got %v", timeout)
	}
}

func TestLoadEnvironmentConfig(t *testing.T) {
	// Set test environment variables
	_ = os.Setenv("GITHUB_TOKEN", "ghp_test_token")
	_ = os.Setenv("GITHUB_WEBHOOK_SECRET", "test-secret")
	_ = os.Setenv("OPENAI_API_KEY", "test-api-key")
	_ = os.Setenv("AI_TEMPERATURE", "0.5")
	_ = os.Setenv("AI_MAX_TOKENS", "2000")

	defer func() {
		_ = os.Unsetenv("GITHUB_TOKEN")
		_ = os.Unsetenv("GITHUB_WEBHOOK_SECRET")
		_ = os.Unsetenv("OPENAI_API_KEY")
		_ = os.Unsetenv("AI_TEMPERATURE")
		_ = os.Unsetenv("AI_MAX_TOKENS")
	}()

	cfg, err := LoadEnvironmentConfig()
	if err != nil {
		t.Fatalf("LoadEnvironmentConfig() error = %v", err)
	}

	if cfg.GitHubToken != "ghp_test_token" {
		t.Errorf("Expected GitHubToken 'ghp_test_token', got %q", cfg.GitHubToken)
	}

	if cfg.AITemperature != 0.5 {
		t.Errorf("Expected AITemperature 0.5, got %f", cfg.AITemperature)
	}

	if cfg.AIMaxTokens != 2000 {
		t.Errorf("Expected AIMaxTokens 2000, got %d", cfg.AIMaxTokens)
	}
}
