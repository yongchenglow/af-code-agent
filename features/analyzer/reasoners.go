package analyzer

import (
	"github.com/Agent-Field/agentfield/sdk/go/agent"
)

// RegisterReasoners registers all analyzer reasoners
// Note: All analyzer functions are deterministic (data fetching/orchestration), so they've been moved to skills
func RegisterReasoners(app *agent.Agent) {
	// All analyzer functions have been moved to skills.go
	// Analyzing PRs and fetching files are deterministic operations, not AI-powered judgment
}
