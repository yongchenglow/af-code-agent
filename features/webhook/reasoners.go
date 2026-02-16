package webhook

import (
	"github.com/Agent-Field/agentfield/sdk/go/agent"
)

// RegisterReasoners registers all webhook-related reasoners
// Note: Webhook processing is deterministic routing, so all functions are now skills
func RegisterReasoners(app *agent.Agent) {
	// All webhook handlers have been moved to skills.go
	// Webhooks are deterministic event routing, not AI-powered judgment
}
