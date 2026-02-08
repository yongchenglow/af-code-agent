package context

import (
	"context"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/yourorg/github-code-agent/pkg/config"
	ghpkg "github.com/yourorg/github-code-agent/pkg/github"
)

// contextKey is a private type for context keys to avoid collisions
type contextKey string

const (
	keyAgent        contextKey = "agent"
	keyConfig       contextKey = "config"
	keyGitHubClient contextKey = "github_client"
)

// WithAgent adds an agent instance to the context
func WithAgent(ctx context.Context, a *agent.Agent) context.Context {
	return context.WithValue(ctx, keyAgent, a)
}

// GetAgent retrieves the agent instance from the context
func GetAgent(ctx context.Context) (*agent.Agent, bool) {
	a, ok := ctx.Value(keyAgent).(*agent.Agent)
	return a, ok
}

// MustGetAgent retrieves the agent instance from context or panics
func MustGetAgent(ctx context.Context) *agent.Agent {
	a, ok := GetAgent(ctx)
	if !ok {
		panic("agent not found in context")
	}
	return a
}

// WithConfig adds a config instance to the context
func WithConfig(ctx context.Context, cfg *config.Config) context.Context {
	return context.WithValue(ctx, keyConfig, cfg)
}

// GetConfig retrieves the config instance from the context
func GetConfig(ctx context.Context) (*config.Config, bool) {
	cfg, ok := ctx.Value(keyConfig).(*config.Config)
	return cfg, ok
}

// MustGetConfig retrieves the config instance from context or panics
func MustGetConfig(ctx context.Context) *config.Config {
	cfg, ok := GetConfig(ctx)
	if !ok {
		panic("config not found in context")
	}
	return cfg
}

// WithGitHubClient adds a GitHub client instance to the context
func WithGitHubClient(ctx context.Context, client *ghpkg.Client) context.Context {
	return context.WithValue(ctx, keyGitHubClient, client)
}

// GetGitHubClient retrieves the GitHub client instance from the context
func GetGitHubClient(ctx context.Context) (*ghpkg.Client, bool) {
	client, ok := ctx.Value(keyGitHubClient).(*ghpkg.Client)
	return client, ok
}

// MustGetGitHubClient retrieves the GitHub client instance from context or panics
func MustGetGitHubClient(ctx context.Context) *ghpkg.Client {
	client, ok := GetGitHubClient(ctx)
	if !ok {
		panic("github_client not found in context")
	}
	return client
}

// WithAll adds all common dependencies to the context
func WithAll(ctx context.Context, a *agent.Agent, cfg *config.Config, client *ghpkg.Client) context.Context {
	ctx = WithAgent(ctx, a)
	ctx = WithConfig(ctx, cfg)
	ctx = WithGitHubClient(ctx, client)
	return ctx
}
