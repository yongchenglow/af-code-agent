package app

import (
	"fmt"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/yourorg/github-code-agent/pkg/config"
	"github.com/yourorg/github-code-agent/pkg/github"
)

// Container holds all application dependencies
type Container struct {
	Config       *config.Config
	EnvConfig    *config.EnvironmentConfig
	Agent        *agent.Agent
	GitHubClient *github.Client
}

// NewContainer creates and initializes the application container
func NewContainer(cfg *config.Config, envCfg *config.EnvironmentConfig, agentInstance *agent.Agent, ghClient *github.Client) (*Container, error) {
	// Validate all dependencies
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if envCfg == nil {
		return nil, fmt.Errorf("environment config is required")
	}
	if agentInstance == nil {
		return nil, fmt.Errorf("agent is required")
	}
	if ghClient == nil {
		return nil, fmt.Errorf("github client is required")
	}

	return &Container{
		Config:       cfg,
		EnvConfig:    envCfg,
		Agent:        agentInstance,
		GitHubClient: ghClient,
	}, nil
}
