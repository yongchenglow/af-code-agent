package gitops

import (
	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/google/go-github/v57/github"
)

// RegisterReasoners registers all gitops reasoners with the agent
// Note: All gitops functions are deterministic (Git/GitHub operations), so they've been moved to skills
func RegisterReasoners(app *agent.Agent, ghClient *github.Client) {
	// All gitops functions have been moved to skills.go
	// Git operations and GitHub API calls are deterministic, not AI-powered judgment
}
