package qasynth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/Agent-Field/agentfield/sdk/go/ai"
	"github.com/yourorg/github-code-agent/pkg/constants"
)

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
	prompt := s.buildSynthesisPrompt(input)

	// Create context with timeout
	aiCtx, cancel := context.WithTimeout(ctx, constants.DefaultAITimeout)
	defer cancel()

	// Use AI for synthesis
	response, err := s.agent.AI(aiCtx, prompt,
		ai.WithSystem(s.buildSynthesisSystemPrompt()),
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

// buildSynthesisPrompt builds the synthesis prompt
func (s *QASynthesizer) buildSynthesisPrompt(input *SynthesisInput) string {
	b := &promptBuilder{}
	b.WriteString("## QA Synthesis Task\n\n")
	b.WriteString(fmt.Sprintf("**Issue**: %s\n", input.IssueID))
	b.WriteString(fmt.Sprintf("**Description**: %s\n\n", input.Description))

	b.WriteString("## Validation Results\n")
	b.WriteString(fmt.Sprintf("- Syntax Valid: %v\n", input.SyntaxValid))
	b.WriteString(fmt.Sprintf("- Linting Passed: %v\n", input.LintingPassed))
	b.WriteString(fmt.Sprintf("- Tests Passed: %v\n\n", input.TestsPassed))

	if len(input.IterationHistory) > 0 {
		b.WriteString("## Iteration History\n")
		for i, iter := range input.IterationHistory {
			b.WriteString(fmt.Sprintf("%d. **%s**: %s\n", i+1, iter.Action, iter.Summary))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Decision Criteria\n\n")
	b.WriteString("Make a decision based on:\n")
	b.WriteString("1. **APPROVE** if all validations pass and the fix is complete\n")
	b.WriteString("2. **FIX** if validations fail or the fix is incomplete\n")
	b.WriteString("3. **BLOCK** if there are critical issues that cannot be fixed automatically\n\n")

	b.WriteString("Output your decision as JSON:\n")
	b.WriteString("```json\n")
	b.WriteString("{\n")
	b.WriteString("  \"action\": \"APPROVE|FIX|BLOCK\",\n")
	b.WriteString("  \"summary\": \"...\",\n")
	b.WriteString("  \"feedback\": [\"...\"],\n")
	b.WriteString("  \"stuck\": false\n")
	b.WriteString("}\n")
	b.WriteString("```\n")

	return b.String()
}

// buildSynthesisSystemPrompt builds the system prompt for synthesis
func (s *QASynthesizer) buildSynthesisSystemPrompt() string {
	return `You are a QA Synthesizer making decisions about fix quality.

## Your Role
You aggregate validation results and iteration history to make go/no-go decisions on fixes.

## Decision Rules
1. **APPROVE** when:
   - All validations pass (syntax, linting, tests)
   - No critical issues remain
   - Fix is complete and minimal

2. **FIX** when:
   - Any validation fails
   - Fix is incomplete
   - Minor issues remain

3. **BLOCK** when:
   - Critical security issues remain
   - Fix introduces new bugs
   - Multiple retry attempts have failed

## Stuck Detection
Mark as "stuck" if:
- Same validation errors appear in 2+ consecutive iterations
- 3 or more fix attempts have been made
- Errors are contradictory or unfixable

Be concise but actionable in feedback.`
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
