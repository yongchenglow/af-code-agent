package fixer

import (
	"strings"
	"testing"

	"github.com/yourorg/github-code-agent/pkg/utils"
)

func TestExtractCodeSection(t *testing.T) {
	code := `line 1
line 2
line 3
line 4
line 5
line 6
line 7`

	tests := []struct {
		name         string
		line         int
		contextLines int
		want         string
	}{
		{
			name:         "middle section",
			line:         4,
			contextLines: 1,
			want:         "line 3\nline 4\nline 5",
		},
		{
			name:         "start section",
			line:         1,
			contextLines: 2,
			want:         "line 1\nline 2\nline 3",
		},
		{
			name:         "end section",
			line:         7,
			contextLines: 2,
			want:         "line 5\nline 6\nline 7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.ExtractCodeSection(code, tt.line, tt.contextLines)
			if got != tt.want {
				t.Errorf("utils.ExtractCodeSection() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractCodeFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     string
	}{
		{
			name:     "plain code",
			response: "fmt.Println(\"hello\")",
			want:     "fmt.Println(\"hello\")",
		},
		{
			name:     "code with markdown",
			response: "```go\nfmt.Println(\"hello\")\n```",
			want:     "fmt.Println(\"hello\")",
		},
		{
			name:     "code with markdown no language",
			response: "```\nfmt.Println(\"hello\")\n```",
			want:     "fmt.Println(\"hello\")",
		},
		{
			name:     "code with explanation",
			response: "Here's the fix:\n```python\nprint(\"hello\")\n```\nThis fixes the issue.",
			want:     "print(\"hello\")",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.ExtractCodeFromResponse(tt.response)
			if got != tt.want {
				t.Errorf("utils.ExtractCodeFromResponse() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestGenerateFixDescription removed - function moved to gitops/message_generator.go

func TestBuildFixPrompt(t *testing.T) {
	issue := map[string]any{
		"title":       "Null pointer dereference",
		"description": "Function doesn't check for nil",
		"suggestion":  "Add nil check",
	}

	code := "x := ptr.Value"
	language := "go"

	prompt := buildFixPrompt(issue, code, language)

	// Check prompt contains key elements
	if !strings.Contains(prompt, "Null pointer dereference") {
		t.Error("Prompt should contain issue title")
	}
	if !strings.Contains(prompt, "Function doesn't check for nil") {
		t.Error("Prompt should contain description")
	}
	if !strings.Contains(prompt, "Add nil check") {
		t.Error("Prompt should contain suggestion")
	}
	if !strings.Contains(prompt, code) {
		t.Error("Prompt should contain original code")
	}
}

func TestCreateBatchFixSummary(t *testing.T) {
	result := &BatchFixResult{
		TotalIssues:  5,
		SuccessCount: 3,
		FailureCount: 2,
		SuccessfulFixes: []*CodePatch{
			{FilePath: "main.go", Line: 10},
			{FilePath: "utils.go", Line: 25},
			{FilePath: "config.go", Line: 5},
		},
		FailedIssues: []string{"issue-4", "issue-5"},
	}

	summary := CreateBatchFixSummary(result)

	// Check summary contains key information
	if !strings.Contains(summary, "Total issues: 5") {
		t.Error("Summary should contain total issues")
	}
	if !strings.Contains(summary, "Successfully fixed: 3") {
		t.Error("Summary should contain success count")
	}
	if !strings.Contains(summary, "Failed to fix: 2") {
		t.Error("Summary should contain failure count")
	}
	if !strings.Contains(summary, "main.go") {
		t.Error("Summary should list successful fixes")
	}
	if !strings.Contains(summary, "issue-4") {
		t.Error("Summary should list failed issues")
	}
}
