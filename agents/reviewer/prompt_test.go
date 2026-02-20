package reviewer

import (
	"strings"
	"testing"

	"github.com/yourorg/github-code-agent/agents/analyzer"
)

func TestBuildReviewPrompt(t *testing.T) {
	tests := []struct {
		name        string
		files       []*analyzer.FileChange
		prContext   map[string]interface{}
		expectWords []string
	}{
		{
			name: "single file no PR context",
			files: []*analyzer.FileChange{
				{
					Filename:  "main.go",
					Language:  "go",
					Patch:     "+fmt.Println()",
					Additions: 1,
					Deletions: 0,
				},
			},
			prContext:   map[string]interface{}{},
			expectWords: []string{"main.go", "go", "Please review"},
		},
		{
			name: "multiple files",
			files: []*analyzer.FileChange{
				{
					Filename:  "main.go",
					Language:  "go",
					Patch:     "+fmt.Println()",
					Additions: 1,
					Deletions: 0,
				},
				{
					Filename:  "utils.go",
					Language:  "go",
					Patch:     "-old\n+new",
					Additions: 1,
					Deletions: 1,
				},
			},
			prContext:   map[string]interface{}{},
			expectWords: []string{"main.go", "utils.go", "Please review"},
		},
		{
			name: "with PR title",
			files: []*analyzer.FileChange{
				{Filename: "main.go", Language: "go"},
			},
			prContext: map[string]interface{}{
				"title": "Fix bug in authentication",
			},
			expectWords: []string{"Fix bug in authentication", "main.go"},
		},
		{
			name: "with PR description",
			files: []*analyzer.FileChange{
				{Filename: "main.go", Language: "go"},
			},
			prContext: map[string]interface{}{
				"description": "This PR fixes a critical security issue",
			},
			expectWords: []string{"This PR fixes a critical security issue", "main.go"},
		},
		{
			name: "with full PR context",
			files: []*analyzer.FileChange{
				{
					Filename:  "auth.go",
					Language:  "go",
					Patch:     "+token := generateToken()",
					Additions: 1,
					Deletions: 0,
				},
			},
			prContext: map[string]interface{}{
				"title":       "Add authentication",
				"description": "Implements JWT authentication",
			},
			expectWords: []string{"Add authentication", "Implements JWT authentication", "auth.go"},
		},
		{
			name:  "empty files",
			files: []*analyzer.FileChange{},
			prContext: map[string]interface{}{
				"title": "Test PR",
			},
			expectWords: []string{"Please review"},
		},
		{
			name: "file without patch",
			files: []*analyzer.FileChange{
				{
					Filename:  "README.md",
					Language:  "markdown",
					Additions: 10,
					Deletions: 0,
				},
			},
			prContext:   map[string]interface{}{},
			expectWords: []string{"README.md", "markdown"},
		},
		{
			name: "file with large patch",
			files: []*analyzer.FileChange{
				{
					Filename:  "main.go",
					Language:  "go",
					Patch:     strings.Repeat("+line\n", 100),
					Additions: 100,
					Deletions: 0,
				},
			},
			prContext:   map[string]interface{}{},
			expectWords: []string{"main.go", "+line"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := buildReviewPrompt(tt.files, tt.prContext)

			if prompt == "" {
				t.Fatal("buildReviewPrompt() returned empty string")
			}

			for _, word := range tt.expectWords {
				if !strings.Contains(prompt, word) {
					t.Errorf("buildReviewPrompt() missing expected word %q in prompt", word)
				}
			}
		})
	}
}

func TestBuildReviewPromptWithInvalidContext(t *testing.T) {
	files := []*analyzer.FileChange{
		{Filename: "main.go", Language: "go"},
	}

	// Test with invalid context types
	prContext := map[string]interface{}{
		"title": 123, // title should be string
	}

	prompt := buildReviewPrompt(files, prContext)

	// Should still generate prompt, just without title
	if !strings.Contains(prompt, "main.go") {
		t.Error("buildReviewPrompt() should generate prompt even with invalid context")
	}
	if strings.Contains(prompt, "123") {
		t.Error("buildReviewPrompt() should not include invalid title value")
	}
}

func TestBuildReviewPromptFormatting(t *testing.T) {
	files := []*analyzer.FileChange{
		{
			Filename:  "main.go",
			Language:  "go",
			Patch:     "+code",
			Additions: 1,
			Deletions: 0,
		},
	}

	prompt := buildReviewPrompt(files, map[string]interface{}{})

	// Check for expected formatting
	if !strings.Contains(prompt, "```") {
		t.Error("buildReviewPrompt() should include code blocks")
	}
	if !strings.Contains(prompt, "File:") {
		t.Error("buildReviewPrompt() should include file headers")
	}
}

func TestBuildSecurityPrompt(t *testing.T) {
	tests := []struct {
		name        string
		files       []*analyzer.FileChange
		expectWords []string
	}{
		{
			name: "single file",
			files: []*analyzer.FileChange{
				{
					Filename: "auth.go",
					Language: "go",
					Content:  "package auth",
				},
			},
			expectWords: []string{"auth.go", "security", "vulnerability"},
		},
		{
			name: "multiple files",
			files: []*analyzer.FileChange{
				{Filename: "auth.go", Language: "go", Content: "package auth"},
				{Filename: "handler.go", Language: "go", Content: "package handler"},
			},
			expectWords: []string{"auth.go", "handler.go", "security"},
		},
		{
			name:        "empty files",
			files:       []*analyzer.FileChange{},
			expectWords: []string{"security"},
		},
		{
			name: "file without content",
			files: []*analyzer.FileChange{
				{Filename: "main.go", Language: "go"},
			},
			expectWords: []string{"main.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := buildSecurityPrompt(tt.files)

			if prompt == "" {
				t.Fatal("buildSecurityPrompt() returned empty string")
			}

			for _, word := range tt.expectWords {
				if !strings.Contains(prompt, word) {
					t.Errorf("buildSecurityPrompt() missing expected word %q", word)
				}
			}
		})
	}
}

func TestBuildSecurityPromptFormatting(t *testing.T) {
	files := []*analyzer.FileChange{
		{
			Filename: "auth.go",
			Language: "go",
			Content:  "package auth\n\nfunc Login() {}",
		},
	}

	prompt := buildSecurityPrompt(files)

	// Check for expected formatting
	if !strings.Contains(prompt, "```") {
		t.Error("buildSecurityPrompt() should include code blocks")
	}
	if !strings.Contains(prompt, "security") {
		t.Error("buildSecurityPrompt() should mention security")
	}
}

func TestFilterReviewableFilesPrompt(t *testing.T) {
	tests := []struct {
		name          string
		files         []*analyzer.FileChange
		expectedCount int
		expectedFiles []string
	}{
		{
			name: "all reviewable",
			files: []*analyzer.FileChange{
				{Filename: "main.go", Language: "go", Status: "added"},
				{Filename: "utils.go", Language: "go", Status: "modified"},
			},
			expectedCount: 2,
			expectedFiles: []string{"main.go", "utils.go"},
		},
		{
			name: "skip removed files",
			files: []*analyzer.FileChange{
				{Filename: "main.go", Language: "go", Status: "added"},
				{Filename: "old.go", Language: "go", Status: "removed"},
			},
			expectedCount: 1,
			expectedFiles: []string{"main.go"},
		},
		{
			name: "skip binary files",
			files: []*analyzer.FileChange{
				{Filename: "main.go", Language: "go", Status: "added"},
				{Filename: "image.png", Language: "binary", Status: "added"},
			},
			expectedCount: 1,
			expectedFiles: []string{"main.go"},
		},
		{
			name: "skip vendor",
			files: []*analyzer.FileChange{
				{Filename: "main.go", Language: "go", Status: "added"},
				{Filename: "vendor/lib.go", Language: "go", Status: "added"},
			},
			expectedCount: 1,
			expectedFiles: []string{"main.go"},
		},
		{
			name: "skip node_modules",
			files: []*analyzer.FileChange{
				{Filename: "main.go", Language: "go", Status: "added"},
				{Filename: "node_modules/pkg.js", Language: "javascript", Status: "added"},
			},
			expectedCount: 1,
			expectedFiles: []string{"main.go"},
		},
		{
			name: "all filtered",
			files: []*analyzer.FileChange{
				{Filename: "image.png", Language: "binary", Status: "added"},
				{Filename: "vendor/lib.go", Language: "go", Status: "added"},
			},
			expectedCount: 0,
			expectedFiles: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterReviewableFiles(tt.files)

			if len(result) != tt.expectedCount {
				t.Errorf("filterReviewableFiles() returned %d files, want %d", len(result), tt.expectedCount)
			}

			// Check expected files are present
			resultFiles := make(map[string]bool)
			for _, f := range result {
				resultFiles[f.Filename] = true
			}

			for _, expected := range tt.expectedFiles {
				if !resultFiles[expected] {
					t.Errorf("filterReviewableFiles() missing expected file %q", expected)
				}
			}
		})
	}
}

func TestFilterCodeFiles(t *testing.T) {
	tests := []struct {
		name          string
		files         []*analyzer.FileChange
		expectedCount int
		expectedFiles []string
	}{
		{
			name: "all code files",
			files: []*analyzer.FileChange{
				{Filename: "main.go", Language: "go"},
				{Filename: "utils.go", Language: "go"},
			},
			expectedCount: 2,
			expectedFiles: []string{"main.go", "utils.go"},
		},
		{
			name: "skip non-code",
			files: []*analyzer.FileChange{
				{Filename: "main.go", Language: "go"},
				{Filename: "README.md", Language: "markdown"},
				{Filename: "config.yaml", Language: "yaml"},
			},
			expectedCount: 1,
			expectedFiles: []string{"main.go"},
		},
		{
			name: "multiple languages",
			files: []*analyzer.FileChange{
				{Filename: "main.go", Language: "go"},
				{Filename: "app.py", Language: "python"},
				{Filename: "index.js", Language: "javascript"},
			},
			expectedCount: 3,
			expectedFiles: []string{"main.go", "app.py", "index.js"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterCodeFiles(tt.files)

			if len(result) != tt.expectedCount {
				t.Errorf("filterCodeFiles() returned %d files, want %d", len(result), tt.expectedCount)
			}
		})
	}
}

func TestParseReviewResponse(t *testing.T) {
	tests := []struct {
		name          string
		response      string
		expectIssues  int
		expectSummary string
	}{
		{
			name: "valid JSON response",
			response: `{
				"summary": "Code review summary",
				"issues": [
					{
						"severity": "Critical",
						"title": "SQL Injection",
						"description": "Vulnerable code"
					}
				]
			}`,
			expectIssues:  1,
			expectSummary: "Code review summary",
		},
		{
			name: "empty issues",
			response: `{
				"summary": "No issues found",
				"issues": []
			}`,
			expectIssues:  0,
			expectSummary: "No issues found",
		},
		{
			name:          "invalid JSON",
			response:      "This is not JSON",
			expectIssues:  0,
			expectSummary: "Unable to parse AI response",
		},
		{
			name: "JSON with markdown",
			response: "Here's the review:\n\n```json\n" + `{
				"summary": "Review summary",
				"issues": []
			}` + "\n```",
			expectIssues:  0,
			expectSummary: "Review summary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := parseReviewResponse(tt.response)

			if err != nil {
				t.Fatalf("parseReviewResponse() error = %v", err)
			}

			if report.Summary != tt.expectSummary {
				t.Errorf("parseReviewResponse() Summary = %q, want %q", report.Summary, tt.expectSummary)
			}

			if len(report.Issues) != tt.expectIssues {
				t.Errorf("parseReviewResponse() returned %d issues, want %d", len(report.Issues), tt.expectIssues)
			}
		})
	}
}

func TestParseSecurityIssues(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		expectIssues int
	}{
		{
			name: "valid JSON array",
			response: `[
				{
					"file_path": "auth.go",
					"line": 42,
					"type": "injection",
					"severity": "Critical",
					"title": "SQL Injection"
				}
			]`,
			expectIssues: 1,
		},
		{
			name:         "empty array",
			response:     "[]",
			expectIssues: 0,
		},
		{
			name:         "invalid JSON",
			response:     "Not JSON",
			expectIssues: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues, err := parseSecurityIssues(tt.response)

			if err != nil {
				t.Fatalf("parseSecurityIssues() error = %v", err)
			}

			if len(issues) != tt.expectIssues {
				t.Errorf("parseSecurityIssues() returned %d issues, want %d", len(issues), tt.expectIssues)
			}
		})
	}
}

func TestBuildReviewPromptWithSpecialCharacters(t *testing.T) {
	files := []*analyzer.FileChange{
		{
			Filename:  "file with spaces.go",
			Language:  "go",
			Patch:     "+code with \"quotes\" and 'apostrophes'",
			Additions: 1,
			Deletions: 0,
		},
	}

	prompt := buildReviewPrompt(files, map[string]interface{}{})

	if !strings.Contains(prompt, "file with spaces.go") {
		t.Error("buildReviewPrompt() should handle filenames with spaces")
	}
}

func TestBuildReviewPromptWithUnicode(t *testing.T) {
	files := []*analyzer.FileChange{
		{
			Filename:  "こんにちは.go",
			Language:  "go",
			Patch:     "+コメント",
			Additions: 1,
			Deletions: 0,
		},
	}

	prompt := buildReviewPrompt(files, map[string]interface{}{})

	if !strings.Contains(prompt, "こんにちは.go") {
		t.Error("buildReviewPrompt() should handle Unicode filenames")
	}
}

func TestBuildSecurityPromptWithLargeContent(t *testing.T) {
	// Create file with large content
	largeContent := strings.Repeat("line\n", 1000)
	files := []*analyzer.FileChange{
		{
			Filename: "large.go",
			Language: "go",
			Content:  largeContent,
		},
	}

	prompt := buildSecurityPrompt(files)

	if prompt == "" {
		t.Error("buildSecurityPrompt() should handle large files")
	}
}

func TestBuildReviewPromptPreservesPatchFormatting(t *testing.T) {
	patch := `+func New() *Handler {
+    return &Handler{}
+}
-old func
+new func`

	files := []*analyzer.FileChange{
		{
			Filename:  "handler.go",
			Language:  "go",
			Patch:     patch,
			Additions: 4,
			Deletions: 1,
		},
	}

	prompt := buildReviewPrompt(files, map[string]interface{}{})

	if !strings.Contains(prompt, patch) {
		t.Error("buildReviewPrompt() should preserve patch formatting")
	}
}

func TestFilterReviewableFilesWithEmptyInput(t *testing.T) {
	result := filterReviewableFiles(nil)
	if len(result) != 0 {
		t.Error("filterReviewableFiles(nil) should return empty slice")
	}
}

func TestFilterCodeFilesWithEmptyInput(t *testing.T) {
	result := filterCodeFiles(nil)
	if len(result) != 0 {
		t.Error("filterCodeFiles(nil) should return empty slice")
	}
}

func TestParseReviewResponseWithMalformedJSON(t *testing.T) {
	// Test various malformed JSON cases
	tests := []struct {
		name     string
		response string
	}{
		{"incomplete JSON", `{"summary": "test"`},
		{"missing quotes", `{summary: "test"}`},
		{"trailing comma", `{"summary": "test",}`},
		{"single quote", `{'summary': 'test'}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := parseReviewResponse(tt.response)

			if err != nil {
				t.Errorf("parseReviewResponse() should handle malformed JSON gracefully: %v", err)
			}

			if report == nil {
				t.Error("parseReviewResponse() should return non-nil report")
			}
		})
	}
}

func TestBuildReviewPromptWithNilContext(t *testing.T) {
	files := []*analyzer.FileChange{
		{Filename: "main.go", Language: "go"},
	}

	prompt := buildReviewPrompt(files, nil)

	if prompt == "" {
		t.Error("buildReviewPrompt() should handle nil context")
	}
	if !strings.Contains(prompt, "main.go") {
		t.Error("buildReviewPrompt() should still include file info with nil context")
	}
}
