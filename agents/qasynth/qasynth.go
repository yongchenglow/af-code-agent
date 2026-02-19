package qasynth

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/Agent-Field/agentfield/sdk/go/ai"
	"github.com/yourorg/github-code-agent/pkg/constants"
)

//go:embed prompts/system.md
var qaSynthesisSystemPrompt string

//go:embed prompts/task.md
var qaSynthesisTaskPrompt string

// QASynthesizer aggregates QA and review results into decisions
type QASynthesizer struct {
	agent *agent.Agent
}

// NewQASynthesizer creates a new QA synthesizer
func NewQASynthesizer(a *agent.Agent) *QASynthesizer {
	return &QASynthesizer{
		agent: a,
	}
}

// SynthesisInput contains inputs for synthesis
type SynthesisInput struct {
	// Issue being evaluated
	IssueID     string `json:"issue_id"`
	Description string `json:"description"`

	// Validation results
	SyntaxValid   bool `json:"syntax_valid"`
	LintingPassed bool `json:"linting_passed"`
	TestsPassed   bool `json:"tests_passed"`

	// Iteration history
	IterationHistory []*IterationRecord `json:"iteration_history"`
}

// IterationRecord tracks a previous iteration
type IterationRecord struct {
	Action  string `json:"action"` // APPROVE, FIX, BLOCK
	Summary string `json:"summary"`
}

// SynthesisDecision is the output decision
type SynthesisDecision struct {
	// Action is the decision: APPROVE, FIX, or BLOCK
	Action string `json:"action"`

	// Summary is a concise explanation
	Summary string `json:"summary"`

	// Feedback is actionable feedback for FIX decisions
	Feedback []string `json:"feedback,omitempty"`

	// Stuck indicates recurring failures (for retry advisor)
	Stuck bool `json:"stuck"`
}

// Synthesize aggregates results into a decision
func (s *QASynthesizer) Synthesize(ctx context.Context, input *SynthesisInput) (*SynthesisDecision, error) {
	prompt := buildQASynthesisPrompt(input)

	// Create context with timeout
	aiCtx, cancel := context.WithTimeout(ctx, constants.DefaultAITimeout)
	defer cancel()

	// Use AI for synthesis
	response, err := s.agent.AI(aiCtx, prompt,
		ai.WithSystem(qaSynthesisSystemPrompt),
		ai.WithTemperature(constants.LowAITemperature),
		ai.WithMaxTokens(1000))

	if err != nil {
		return nil, fmt.Errorf("QA synthesis failed: %w", err)
	}

	// Parse decision from response
	decision, err := s.parseDecision(response.Text())
	if err != nil {
		return nil, fmt.Errorf("failed to parse synthesis decision: %w", err)
	}

	return decision, nil
}

// buildQASynthesisPrompt builds the synthesis prompt from template
func buildQASynthesisPrompt(input *SynthesisInput) string {
	iterationHistory := ""
	if len(input.IterationHistory) > 0 {
		iterationHistory += "## Iteration History\n"
		for i, iter := range input.IterationHistory {
			iterationHistory += fmt.Sprintf("%d. **%s**: %s\n", i+1, iter.Action, iter.Summary)
		}
		iterationHistory += "\n"
	}

	return fmt.Sprintf(qaSynthesisTaskPrompt,
		input.IssueID,
		input.Description,
		input.SyntaxValid,
		input.LintingPassed,
		input.TestsPassed,
		iterationHistory,
	)
}

// parseDecision parses the decision from AI response
func (s *QASynthesizer) parseDecision(text string) (*SynthesisDecision, error) {
	jsonText := extractJSON(text)

	var decision SynthesisDecision
	if err := json.Unmarshal([]byte(jsonText), &decision); err != nil {
		// Return default decision if parsing fails
		return &SynthesisDecision{
			Action:  "FIX",
			Summary: "Unable to parse synthesis decision, defaulting to FIX",
			Stuck:   false,
		}, nil
	}

	// Validate action
	if decision.Action != "APPROVE" && decision.Action != "FIX" && decision.Action != "BLOCK" {
		decision.Action = "FIX"
		decision.Summary = "Invalid action, defaulting to FIX"
	}

	return &decision, nil
}

// extractJSON extracts JSON from text
func extractJSON(text string) string {
	// Try to find JSON block
	start := -1
	end := -1

	lines := splitLines(text)
	for i, line := range lines {
		if contains(line, "```json") {
			start = i + 1
		} else if start != -1 && contains(line, "```") {
			end = i
			break
		}
	}

	if start != -1 && end != -1 {
		return joinLines(lines[start:end])
	}

	// Try to find JSON object directly
	for i, ch := range text {
		if ch == '{' {
			start = i
		} else if ch == '}' && start != -1 {
			end = i + 1
			break
		}
	}

	if start != -1 && end != -1 {
		return text[start:end]
	}

	return text
}

// Helper functions

func splitLines(s string) []string {
	result := []string{}
	current := ""
	for _, ch := range s {
		if ch == '\n' {
			result = append(result, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func joinLines(lines []string) string {
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

type promptBuilder struct {
	content string
}

func (pb *promptBuilder) WriteString(s string) {
	pb.content += s
}

func (pb *promptBuilder) String() string {
	return pb.content
}
