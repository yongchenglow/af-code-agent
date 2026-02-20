package qasynth

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSynthesisInputStruct(t *testing.T) {
	input := &SynthesisInput{
		IssueID:       "ISSUE-001",
		Description:   "Test issue description",
		SyntaxValid:   true,
		LintingPassed: false,
		TestsPassed:   true,
		IterationHistory: []*IterationRecord{
			{Action: "FIX", Summary: "Attempted fix"},
		},
	}

	if input.IssueID != "ISSUE-001" {
		t.Errorf("expected IssueID to be 'ISSUE-001', got %q", input.IssueID)
	}
	if !input.SyntaxValid {
		t.Error("expected SyntaxValid to be true")
	}
	if input.LintingPassed {
		t.Error("expected LintingPassed to be false")
	}
	if len(input.IterationHistory) != 1 {
		t.Errorf("expected 1 iteration, got %d", len(input.IterationHistory))
	}
}

func TestIterationRecordStruct(t *testing.T) {
	record := &IterationRecord{
		Action:  "APPROVE",
		Summary: "All checks passed",
	}

	if record.Action != "APPROVE" {
		t.Errorf("expected Action to be 'APPROVE', got %q", record.Action)
	}
	if record.Summary != "All checks passed" {
		t.Errorf("expected Summary to be 'All checks passed', got %q", record.Summary)
	}
}

func TestSynthesisDecisionStruct(t *testing.T) {
	decision := &SynthesisDecision{
		Action:   "FIX",
		Summary:  "Needs fixing",
		Feedback: []string{"Fix linting errors", "Add tests"},
		Stuck:    false,
	}

	if decision.Action != "FIX" {
		t.Errorf("expected Action to be 'FIX', got %q", decision.Action)
	}
	if len(decision.Feedback) != 2 {
		t.Errorf("expected 2 feedback items, got %d", len(decision.Feedback))
	}
	if decision.Stuck {
		t.Error("expected Stuck to be false")
	}
}

func TestSynthesisDecisionWithStuck(t *testing.T) {
	decision := &SynthesisDecision{
		Action:   "BLOCK",
		Summary:  "Repeated failures",
		Feedback: []string{"Multiple attempts failed"},
		Stuck:    true,
	}

	if decision.Action != "BLOCK" {
		t.Errorf("expected Action to be 'BLOCK', got %q", decision.Action)
	}
	if !decision.Stuck {
		t.Error("expected Stuck to be true")
	}
}

func TestNewQASynthesizer(t *testing.T) {
	// Test that constructor returns non-nil synthesizer
	// Note: We can't fully test without a real agent
	t.Skip("Skipping - requires agent mock")
}

func TestBuildQASynthesisPrompt(t *testing.T) {
	input := &SynthesisInput{
		IssueID:          "ISSUE-001",
		Description:      "Test description",
		SyntaxValid:      true,
		LintingPassed:    false,
		TestsPassed:      true,
		IterationHistory: []*IterationRecord{},
	}

	prompt := buildQASynthesisPrompt(input)

	if !strings.Contains(prompt, "ISSUE-001") {
		t.Error("expected issue ID in prompt")
	}
	if !strings.Contains(prompt, "Test description") {
		t.Error("expected description in prompt")
	}
	if !strings.Contains(prompt, "true") {
		t.Error("expected validation results in prompt")
	}
}

func TestBuildQASynthesisPromptWithHistory(t *testing.T) {
	input := &SynthesisInput{
		IssueID:       "ISSUE-001",
		Description:   "Test description",
		SyntaxValid:   true,
		LintingPassed: true,
		TestsPassed:   true,
		IterationHistory: []*IterationRecord{
			{Action: "FIX", Summary: "First attempt"},
			{Action: "FIX", Summary: "Second attempt"},
		},
	}

	prompt := buildQASynthesisPrompt(input)

	if !strings.Contains(prompt, "## Iteration History") {
		t.Error("expected iteration history section in prompt")
	}
	if !strings.Contains(prompt, "**FIX**") {
		t.Error("expected action in iteration history")
	}
}

func TestBuildQASynthesisPromptEmptyHistory(t *testing.T) {
	input := &SynthesisInput{
		IssueID:          "ISSUE-001",
		Description:      "Test description",
		SyntaxValid:      true,
		LintingPassed:    true,
		TestsPassed:      true,
		IterationHistory: []*IterationRecord{},
	}

	prompt := buildQASynthesisPrompt(input)

	if strings.Contains(prompt, "## Iteration History") {
		t.Error("should not include iteration history section when empty")
	}
}

func TestParseDecisionValidJSON(t *testing.T) {
	s := &QASynthesizer{}
	response := `{
		"action": "APPROVE",
		"summary": "All checks passed",
		"stuck": false
	}`

	decision, err := s.parseDecision(response)

	if err != nil {
		t.Fatalf("parseDecision failed: %v", err)
	}

	if decision.Action != "APPROVE" {
		t.Errorf("expected Action to be 'APPROVE', got %q", decision.Action)
	}
	if decision.Summary != "All checks passed" {
		t.Errorf("expected Summary to be 'All checks passed', got %q", decision.Summary)
	}
	if decision.Stuck {
		t.Error("expected Stuck to be false")
	}
}

func TestParseDecisionWithFeedback(t *testing.T) {
	s := &QASynthesizer{}
	response := `{
		"action": "FIX",
		"summary": "Linting errors found",
		"feedback": ["Fix line length", "Add comments"],
		"stuck": false
	}`

	decision, err := s.parseDecision(response)

	if err != nil {
		t.Fatalf("parseDecision failed: %v", err)
	}

	if decision.Action != "FIX" {
		t.Errorf("expected Action to be 'FIX', got %q", decision.Action)
	}
	if len(decision.Feedback) != 2 {
		t.Errorf("expected 2 feedback items, got %d", len(decision.Feedback))
	}
}

func TestParseDecisionInvalidJSON(t *testing.T) {
	s := &QASynthesizer{}
	response := `This is not valid JSON`

	decision, err := s.parseDecision(response)

	if err != nil {
		t.Fatalf("parseDecision should not return error for invalid JSON: %v", err)
	}

	if decision.Action != "FIX" {
		t.Errorf("expected default Action to be 'FIX', got %q", decision.Action)
	}
	if decision.Summary != "Unable to parse synthesis decision, defaulting to FIX" {
		t.Errorf("expected default summary, got %q", decision.Summary)
	}
}

func TestParseDecisionInvalidAction(t *testing.T) {
	s := &QASynthesizer{}
	response := `{
		"action": "INVALID_ACTION",
		"summary": "Test"
	}`

	decision, err := s.parseDecision(response)

	if err != nil {
		t.Fatalf("parseDecision failed: %v", err)
	}

	if decision.Action != "FIX" {
		t.Errorf("expected Action to default to 'FIX', got %q", decision.Action)
	}
	if !strings.Contains(decision.Summary, "Invalid action") {
		t.Errorf("expected summary to mention invalid action, got %q", decision.Summary)
	}
}

func TestParseDecisionMarkdownJSON(t *testing.T) {
	s := &QASynthesizer{}
	response := `Here is the decision:

` + "```json" + `
{
	"action": "BLOCK",
	"summary": "Critical issues found",
	"stuck": true
}
` + "```"

	decision, err := s.parseDecision(response)

	if err != nil {
		t.Fatalf("parseDecision failed: %v", err)
	}

	if decision.Action != "BLOCK" {
		t.Errorf("expected Action to be 'BLOCK', got %q", decision.Action)
	}
	if decision.Stuck {
		t.Error("expected Stuck to be true")
	}
}

func TestParseDecisionAllValidActions(t *testing.T) {
	actions := []string{"APPROVE", "FIX", "BLOCK"}

	for _, action := range actions {
		t.Run(action, func(t *testing.T) {
			s := &QASynthesizer{}
			response := `{"action": "` + action + `", "summary": "Test"}`

			decision, err := s.parseDecision(response)

			if err != nil {
				t.Fatalf("parseDecision failed: %v", err)
			}

			if decision.Action != action {
				t.Errorf("expected Action to be %q, got %q", action, decision.Action)
			}
		})
	}
}

func TestExtractJSONFromMarkdown(t *testing.T) {
	text := `Some text before
` + "```json" + `
{"key": "value"}
` + "```" + `
Some text after`

	result := extractJSON(text)

	expected := `{"key": "value"}`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExtractJSONDirectObject(t *testing.T) {
	text := `{"action": "APPROVE", "summary": "Test"}`

	result := extractJSON(text)

	if result != text {
		t.Errorf("expected %q, got %q", text, result)
	}
}

func TestExtractJSONFromArray(t *testing.T) {
	text := `[{"id": 1}, {"id": 2}]`

	result := extractJSON(text)

	if result != text {
		t.Errorf("expected %q, got %q", text, result)
	}
}

func TestExtractJSONWithLanguage(t *testing.T) {
	text := "```go\n{\"key\": \"value\"}\n```"

	result := extractJSON(text)

	expected := `{"key": "value"}`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExtractJSONEmpty(t *testing.T) {
	result := extractJSON("")

	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		input  string
		expect []string
	}{
		{"", []string{}},
		{"single", []string{"single"}},
		{"line1\nline2", []string{"line1", "line2"}},
		{"line1\nline2\nline3", []string{"line1", "line2", "line3"}},
		{"\n", []string{"", ""}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitLines(tt.input)
			if len(result) != len(tt.expect) {
				t.Errorf("splitLines(%q) returned %d lines, expected %d", tt.input, len(result), len(tt.expect))
			}
			for i, exp := range tt.expect {
				if i >= len(result) || result[i] != exp {
					t.Errorf("splitLines(%q) line %d = %q, want %q", tt.input, i, result[i], exp)
				}
			}
		})
	}
}

func TestJoinLines(t *testing.T) {
	tests := []struct {
		input  []string
		expect string
	}{
		{[]string{}, ""},
		{[]string{"single"}, "single"},
		{[]string{"line1", "line2"}, "line1\nline2"},
	}

	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			result := joinLines(tt.input)
			if result != tt.expect {
				t.Errorf("joinLines(%v) = %q, want %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		expect bool
	}{
		{"hello world", "world", true},
		{"hello world", "Hello", false},
		{"hello world", "", true},
		{"", "world", false},
		{"", "", true},
		{"test", "test", true},
		{"test", "testing", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"-"+tt.substr, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			if result != tt.expect {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expect)
			}
		})
	}
}

func TestFindSubstring(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		expect bool
	}{
		{"hello world", "world", true},
		{"hello world", "worf", false},
		{"", "", true},
		{"test", "", true},
		{"", "test", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"-"+tt.substr, func(t *testing.T) {
			result := findSubstring(tt.s, tt.substr)
			if result != tt.expect {
				t.Errorf("findSubstring(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expect)
			}
		})
	}
}

func TestSynthesisDecisionJSONMarshaling(t *testing.T) {
	decision := &SynthesisDecision{
		Action:   "FIX",
		Summary:  "Test summary",
		Feedback: []string{"Feedback 1", "Feedback 2"},
		Stuck:    true,
	}

	data, err := json.Marshal(decision)
	if err != nil {
		t.Fatalf("failed to marshal SynthesisDecision: %v", err)
	}

	var unmarshaled SynthesisDecision
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal SynthesisDecision: %v", err)
	}

	if unmarshaled.Action != decision.Action {
		t.Errorf("expected Action %q, got %q", decision.Action, unmarshaled.Action)
	}
	if unmarshaled.Summary != decision.Summary {
		t.Errorf("expected Summary %q, got %q", decision.Summary, unmarshaled.Summary)
	}
	if len(unmarshaled.Feedback) != len(decision.Feedback) {
		t.Errorf("expected %d feedback items, got %d", len(decision.Feedback), len(unmarshaled.Feedback))
	}
	if unmarshaled.Stuck != decision.Stuck {
		t.Errorf("expected Stuck %v, got %v", decision.Stuck, unmarshaled.Stuck)
	}
}

func TestSynthesisInputJSONMarshaling(t *testing.T) {
	input := &SynthesisInput{
		IssueID:       "TEST-001",
		Description:   "Test description",
		SyntaxValid:   true,
		LintingPassed: false,
		TestsPassed:   true,
		IterationHistory: []*IterationRecord{
			{Action: "FIX", Summary: "Attempt 1"},
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal SynthesisInput: %v", err)
	}

	var unmarshaled SynthesisInput
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal SynthesisInput: %v", err)
	}

	if unmarshaled.IssueID != input.IssueID {
		t.Errorf("expected IssueID %q, got %q", input.IssueID, unmarshaled.IssueID)
	}
	if unmarshaled.SyntaxValid != input.SyntaxValid {
		t.Errorf("expected SyntaxValid %v, got %v", input.SyntaxValid, unmarshaled.SyntaxValid)
	}
}

func TestSynthesizeContextTimeout(t *testing.T) {
	// This test would require mocking the agent
	t.Skip("Skipping - requires agent mock")
}

func TestQASynthesizerWithNilInput(t *testing.T) {
	s := &QASynthesizer{}

	// Test that parseDecision handles edge cases
	decision, err := s.parseDecision("")
	if err != nil {
		t.Fatalf("parseDecision should handle empty input: %v", err)
	}

	if decision.Action != "FIX" {
		t.Errorf("expected default action 'FIX', got %q", decision.Action)
	}
}

func TestBuildQASynthesisPromptValidationFlags(t *testing.T) {
	tests := []struct {
		name          string
		syntaxValid   bool
		lintingPassed bool
		testsPassed   bool
	}{
		{"all true", true, true, true},
		{"all false", false, false, false},
		{"mixed", true, false, true},
		{"only syntax valid", true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &SynthesisInput{
				IssueID:       "TEST-001",
				Description:   "Test",
				SyntaxValid:   tt.syntaxValid,
				LintingPassed: tt.lintingPassed,
				TestsPassed:   tt.testsPassed,
			}

			prompt := buildQASynthesisPrompt(input)

			// Prompt should contain the boolean values
			if tt.syntaxValid && !strings.Contains(prompt, "true") {
				t.Error("expected 'true' for syntax valid")
			}
			if !tt.lintingPassed && !strings.Contains(prompt, "false") {
				t.Error("expected 'false' for linting passed")
			}
		})
	}
}
