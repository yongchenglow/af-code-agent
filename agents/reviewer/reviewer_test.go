package reviewer

import (
	"testing"

	"github.com/yourorg/github-code-agent/agents/analyzer"
	"github.com/yourorg/github-code-agent/pkg/utils"
)

func TestFilterReviewableFiles(t *testing.T) {
	tests := []struct {
		name     string
		files    []*analyzer.FileChange
		expected int
	}{
		{
			name: "filter out deleted files",
			files: []*analyzer.FileChange{
				{Filename: "main.go", Status: "modified"},
				{Filename: "test.go", Status: "removed"},
				{Filename: "util.go", Status: "added"},
			},
			expected: 2,
		},
		{
			name: "filter out binary files",
			files: []*analyzer.FileChange{
				{Filename: "main.go", Status: "modified"},
				{Filename: "logo.png", Status: "added"},
				{Filename: "doc.pdf", Status: "added"},
			},
			expected: 1,
		},
		{
			name: "filter out vendor and node_modules",
			files: []*analyzer.FileChange{
				{Filename: "main.go", Status: "modified"},
				{Filename: "vendor/package.go", Status: "added"},
				{Filename: "node_modules/lib.js", Status: "added"},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterReviewableFiles(tt.files)
			if len(result) != tt.expected {
				t.Errorf("Expected %d files, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestShouldSkipFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"main.go", false},
		{"test.py", false},
		{"logo.png", true},
		{"doc.pdf", true},
		{"package-lock.json", true},
		{"go.sum", true},
		{"vendor/lib.go", true},
		{"node_modules/package.js", true},
		{"dist/bundle.js", true},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := utils.ShouldSkipFile(tt.filename)
			if result != tt.expected {
				t.Errorf("Expected %v for %s, got %v", tt.expected, tt.filename, result)
			}
		})
	}
}

func TestIsCodeLanguage(t *testing.T) {
	tests := []struct {
		lang     string
		expected bool
	}{
		{"go", true},
		{"python", true},
		{"javascript", true},
		{"typescript", true},
		{"unknown", false},
		{"", false},
		{"markdown", false},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			result := utils.IsCodeLanguage(tt.lang)
			if result != tt.expected {
				t.Errorf("Expected %v for %s, got %v", tt.expected, tt.lang, result)
			}
		})
	}
}

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain JSON object",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON in markdown code block",
			input:    "```json\n{\"key\": \"value\"}\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with leading text",
			input:    "Here is the result:\n{\"key\": \"value\"}",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON array with leading text",
			input:    `Result: [{"key": "value"}]`,
			expected: `[{"key": "value"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.ExtractJSON(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetSeverityEmoji(t *testing.T) {
	tests := []struct {
		severity string
		expected string
	}{
		{SeverityCritical, "🔴"},
		{SeverityHigh, "🟠"},
		{SeverityMedium, "🟡"},
		{SeverityLow, "🔵"},
		{"Unknown", "⚪"},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			result := getSeverityEmoji(tt.severity)
			if result != tt.expected {
				t.Errorf("Expected %s for %s, got %s", tt.expected, tt.severity, result)
			}
		})
	}
}
