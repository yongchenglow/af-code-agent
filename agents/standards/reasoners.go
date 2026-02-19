package standards

import (
	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/yourorg/github-code-agent/pkg/config"
)

// RegisterReasoners registers all standards validation reasoners
// Note: Standards validation is rule-based (deterministic), so it's been moved to skills
func RegisterReasoners(app *agent.Agent, cfg *config.Config) {
	// All standards validation functions have been moved to skills.go
	// Standards validation uses pattern matching and rules, not AI-powered judgment
}
