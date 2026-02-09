package app

import (
	"fmt"
	"log"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/Agent-Field/agentfield/sdk/go/ai"
	"github.com/joho/godotenv"
	"github.com/yourorg/github-code-agent/pkg/config"
	"github.com/yourorg/github-code-agent/pkg/constants"
	"github.com/yourorg/github-code-agent/pkg/github"
)

// Bootstrap handles application initialization
type Bootstrap struct{}

// NewBootstrap creates a new bootstrap instance
func NewBootstrap() *Bootstrap {
	return &Bootstrap{}
}

// LoadEnvironment loads environment variables from .env file
func (b *Bootstrap) LoadEnvironment() error {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}
	return nil
}

// LoadConfig loads application configuration
func (b *Bootstrap) LoadConfig(configPath string) (*config.Config, error) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Printf("Warning: Failed to load config: %v", err)
		log.Println("Using default configuration")
		return config.DefaultConfig(), nil
	}
	return cfg, nil
}

// LoadEnvConfig loads environment configuration
func (b *Bootstrap) LoadEnvConfig() (*config.EnvironmentConfig, error) {
	envCfg, err := config.LoadEnvironmentConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load environment config: %w", err)
	}

	if err := envCfg.Validate(); err != nil {
		return nil, fmt.Errorf("environment config validation failed: %w", err)
	}

	return envCfg, nil
}

// CreateAIConfig creates AI configuration
func (b *Bootstrap) CreateAIConfig() *ai.Config {
	aiConfig := ai.DefaultConfig()
	aiConfig.Timeout = constants.DefaultAITimeout
	return aiConfig
}

// CreateAgent creates the AgentField agent
func (b *Bootstrap) CreateAgent(envCfg *config.EnvironmentConfig, aiConfig *ai.Config) (*agent.Agent, error) {
	app, err := agent.New(agent.Config{
		NodeID:        constants.AgentNodeID,
		Version:       constants.AgentVersion,
		TeamID:        constants.AgentTeamID,
		AgentFieldURL: envCfg.AgentFieldURL,
		AIConfig:      aiConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}
	return app, nil
}

// CreateGitHubClient creates the GitHub client with App authentication
func (b *Bootstrap) CreateGitHubClient(appID, privateKey string) (*github.Client, error) {
	ghClient, err := github.NewClientWithAppCredentials(appID, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}
	return ghClient, nil
}

// Initialize performs full application initialization
func (b *Bootstrap) Initialize() (*Container, error) {
	// Load environment
	if err := b.LoadEnvironment(); err != nil {
		return nil, err
	}

	// Load configs
	cfg, err := b.LoadConfig(".github/code-agent.yml")
	if err != nil {
		return nil, err
	}

	envCfg, err := b.LoadEnvConfig()
	if err != nil {
		return nil, err
	}

	// Create AI config
	aiConfig := b.CreateAIConfig()

	// Create agent
	agentInstance, err := b.CreateAgent(envCfg, aiConfig)
	if err != nil {
		return nil, err
	}

	// Create GitHub client
	ghClient, err := b.CreateGitHubClient(envCfg.GitHubAppID, envCfg.GitHubPrivateKey)
	if err != nil {
		return nil, err
	}

	// Create container
	container, err := NewContainer(cfg, envCfg, agentInstance, ghClient)
	if err != nil {
		return nil, err
	}

	return container, nil
}
