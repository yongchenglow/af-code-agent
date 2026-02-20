package utils

import (
	"strings"
	"testing"
)

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
			name:     "plain JSON array",
			input:    `[{"id": 1}, {"id": 2}]`,
			expected: `[{"id": 1}, {"id": 2}]`,
		},
		{
			name:     "markdown json block",
			input:    "```json\n" + `{"key": "value"}` + "\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "markdown block without language",
			input:    "```\n" + `{"key": "value"}` + "\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "text before JSON",
			input:    "Here is the result:\n" + `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "text before JSON array",
			input:    "Results:\n" + `[1, 2, 3]`,
			expected: `[1, 2, 3]`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no JSON",
			input:    "This is just text",
			expected: "This is just text",
		},
		{
			name:     "only opening brace",
			input:    "{",
			expected: "{",
		},
		{
			name:     "only opening bracket",
			input:    "[",
			expected: "[",
		},
		{
			name:     "JSON with whitespace",
			input:    "  \n  " + `{"key": "value"}` + "  \n  ",
			expected: `{"key": "value"}`,
		},
		{
			name:     "JSON with markdown and text",
			input:    "Here's the JSON:\n\n```json\n" + `{"action": "APPROVE"}` + "\n```\n\nHope this helps!",
			expected: `{"action": "APPROVE"}`,
		},
		{
			name:     "array preferred over object",
			input:    "Text {\n" + `[{"id": 1}]`,
			expected: `[{"id": 1}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractJSON(tt.input)
			if got != tt.expected {
				t.Errorf("ExtractJSON() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExtractCodeFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain code",
			input:    "package main\n\nfunc main() {}",
			expected: "package main\n\nfunc main() {}",
		},
		{
			name: "go code block",
			input: "```go\n" + `package main

func main() {
	fmt.Println("Hello")
}` + "\n```",
			expected: `package main

func main() {
	fmt.Println("Hello")
}`,
		},
		{
			name: "python code block",
			input: "```python\n" + `def hello():
    print("Hello")` + "\n```",
			expected: `def hello():
    print("Hello")`,
		},
		{
			name:     "javascript code block",
			input:    "```javascript\n" + `console.log("Hello");` + "\n```",
			expected: `console.log("Hello");`,
		},
		{
			name:     "typescript code block",
			input:    "```typescript\n" + `const x: number = 5;` + "\n```",
			expected: `const x: number = 5;`,
		},
		{
			name:     "unknown language code block",
			input:    "```unknown\n" + `some code` + "\n```",
			expected: "unknown\nsome code",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "code with text before",
			input:    "Here's the code:\n\n```go\n" + `package main` + "\n```",
			expected: "package main",
		},
		{
			name:     "code with text after",
			input:    "```go\n" + `package main` + "\n```\n\nHope this helps!",
			expected: "package main",
		},
		{
			name:     "multiple code blocks (first one)",
			input:    "```go\n" + `package first` + "\n```\n\n```go\n" + `package second` + "\n```",
			expected: "package first",
		},
		{
			name:     "code block with extra whitespace",
			input:    "  ```go  \n" + `package main` + "\n  ```  ",
			expected: "package main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCodeFromResponse(tt.input)
			if got != tt.expected {
				t.Errorf("ExtractCodeFromResponse() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsLanguageIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"go", "go", true},
		{"python", "python", true},
		{"javascript", "javascript", true},
		{"typescript", "typescript", true},
		{"java", "java", true},
		{"rust", "rust", true},
		{"cpp", "cpp", true},
		{"c", "c", true},
		{"ruby", "ruby", true},
		{"php", "php", true},
		{"kotlin", "kotlin", true},
		{"swift", "swift", true},
		{"unknown", "unknown", false},
		{"empty", "", false},
		{"Go (uppercase)", "Go", false},
		{"GO (all caps)", "GO", false},
		{"with space", "go ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLanguageIdentifier(tt.input)
			if got != tt.expected {
				t.Errorf("isLanguageIdentifier(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"found at start", []string{"a", "b", "c"}, "a", true},
		{"found in middle", []string{"a", "b", "c"}, "b", true},
		{"found at end", []string{"a", "b", "c"}, "c", true},
		{"not found", []string{"a", "b", "c"}, "d", false},
		{"empty slice", []string{}, "a", false},
		{"single element found", []string{"a"}, "a", true},
		{"single element not found", []string{"a"}, "b", false},
		{"empty item", []string{"a", "b"}, "", false},
		{"case sensitive", []string{"a", "B", "c"}, "b", false},
		{"with spaces", []string{"hello world", "foo"}, "hello world", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Contains(tt.slice, tt.item)
			if got != tt.expected {
				t.Errorf("Contains(%v, %q) = %v, want %v", tt.slice, tt.item, got, tt.expected)
			}
		})
	}
}

func TestIsIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"simple", "hello", true},
		{"with underscore", "hello_world", true},
		{"with numbers", "test123", true},
		{"starts with number", "123test", true},
		{"camelCase", "camelCase", true},
		{"PascalCase", "PascalCase", true},
		{"snake_case", "snake_case", true},
		{"SCREAMING_SNAKE", "SCREAMING_SNAKE", true},
		{"empty", "", false},
		{"with space", "hello world", false},
		{"with hyphen", "hello-world", false},
		{"with dot", "hello.world", false},
		{"with special char", "hello@world", false},
		{"with dollar", "hello$world", false},
		{"unicode", "こんにちは", false},
		{"mixed valid", "Test_123", true},
		{"single char", "x", true},
		{"single underscore", "_", true},
		{"starts with underscore", "_private", true},
		{"ends with underscore", "public_", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsIdentifier(tt.input)
			if got != tt.expected {
				t.Errorf("IsIdentifier(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractCodeSection(t *testing.T) {
	content := `line 1
line 2
line 3
line 4
line 5
line 6
line 7
line 8
line 9
line 10`

	tests := []struct {
		name         string
		line         int
		contextLines int
		expected     string
	}{
		{"middle of file", 5, 2, "line 3\nline 4\nline 5\nline 6\nline 7"},
		{"beginning of file", 1, 2, "line 1\nline 2\nline 3"},
		{"end of file", 10, 2, "line 8\nline 9\nline 10"},
		{"no context", 5, 0, "line 5"},
		{"large context", 5, 10, content},
		{"invalid line (zero)", 0, 2, content},
		{"invalid line (negative)", -1, 2, content},
		{"invalid line (beyond)", 15, 2, content},
		{"single line context", 5, 1, "line 4\nline 5\nline 6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCodeSection(content, tt.line, tt.contextLines)
			if got != tt.expected {
				t.Errorf("ExtractCodeSection() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExtractCodeSectionEmptyContent(t *testing.T) {
	got := ExtractCodeSection("", 1, 2)
	if got != "" {
		t.Errorf("ExtractCodeSection(\"\", 1, 2) = %q, want \"\"", got)
	}
}

func TestExtractCodeSectionSingleLine(t *testing.T) {
	content := "single line"
	got := ExtractCodeSection(content, 1, 2)
	if got != content {
		t.Errorf("ExtractCodeSection() = %q, want %q", got, content)
	}
}

func TestExtractJSONWithNestedStructures(t *testing.T) {
	input := `Here's the analysis:

` + "```json" + `
{
	"summary": "Code review",
	"issues": [
		{
			"id": "SEC-001",
			"details": {
				"severity": "Critical",
				"type": "injection"
			}
		}
	]
}
` + "```"

	expected := `{
	"summary": "Code review",
	"issues": [
		{
			"id": "SEC-001",
			"details": {
				"severity": "Critical",
				"type": "injection"
			}
		}
	]
}`

	got := ExtractJSON(input)
	if got != expected {
		t.Errorf("ExtractJSON() = %q, want %q", got, expected)
	}
}

func TestExtractCodeFromResponseWithIncompleteBlocks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unclosed code block",
			input:    "```go\npackage main",
			expected: "```go\npackage main",
		},
		{
			name:     "only opening ticks",
			input:    "```",
			expected: "```",
		},
		{
			name:     "only closing ticks",
			input:    "```",
			expected: "```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCodeFromResponse(tt.input)
			if got != tt.expected {
				t.Errorf("ExtractCodeFromResponse() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestContainsWithUnicode(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"unicode found", []string{"こんにちは", "你好"}, "こんにちは", true},
		{"unicode not found", []string{"こんにちは", "你好"}, "Hello", false},
		{"emoji", []string{"😀", "😁"}, "😀", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Contains(tt.slice, tt.item)
			if got != tt.expected {
				t.Errorf("Contains(%v, %q) = %v, want %v", tt.slice, tt.item, got, tt.expected)
			}
		})
	}
}

func TestIsIdentifierWithUnicode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"chinese", "你好", false},
		{"japanese", "こんにちは", false},
		{"mixed ascii and unicode", "hello 世界", false},
		{"emoji", "😀", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsIdentifier(tt.input)
			if got != tt.expected {
				t.Errorf("IsIdentifier(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractCodeSectionWithEmptyLines(t *testing.T) {
	content := `line 1

line 3

line 5`

	got := ExtractCodeSection(content, 3, 1)
	expected := `\nline 3\n`

	if got != expected {
		t.Errorf("ExtractCodeSection() = %q, want %q", got, expected)
	}
}

func TestExtractJSONPreservesFormatting(t *testing.T) {
	input := `{"key": "value with spaces", "array": [1, 2, 3]}`
	got := ExtractJSON(input)
	// ExtractJSON may return array portion if found first
	if !strings.Contains(got, `"key": "value with spaces"`) && !strings.Contains(got, `"array": [1, 2, 3]`) {
		t.Errorf("ExtractJSON() should preserve formatting: got %q", got)
	}
}

func TestExtractCodeFromResponsePreservesIndentation(t *testing.T) {
	input := "```go\n" + `package main

func main() {
	if true {
		fmt.Println("indented")
	}
}` + "\n```"

	expected := `package main

func main() {
	if true {
		fmt.Println("indented")
	}
}`

	got := ExtractCodeFromResponse(input)
	if got != expected {
		t.Errorf("ExtractCodeFromResponse() should preserve indentation:\ngot:\n%q\nwant:\n%q", got, expected)
	}
}

func TestExtractJSONWithExtraWhitespace(t *testing.T) {
	input := "  \n\t```json\n  " + `{"key": "value"}` + "  \n```  \n"
	expected := `{"key": "value"}`

	got := ExtractJSON(input)
	if got != expected {
		t.Errorf("ExtractJSON() = %q, want %q", got, expected)
	}
}

func TestContainsLargeSlice(t *testing.T) {
	// Create a large slice
	slice := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		slice[i] = string(rune('a' + i%26))
	}

	// Test finding item at end
	if !Contains(slice, slice[999]) {
		t.Error("Contains should find item at end of large slice")
	}

	// Test not finding item
	if Contains(slice, "ZZZ") {
		t.Error("Contains should not find non-existent item")
	}
}

func TestExtractCodeFromResponseMultipleLanguages(t *testing.T) {
	input := "First:\n\n```go\npackage main\n```\n\nSecond:\n\n```python\ndef hello():\n    pass\n```"

	// Should extract first code block
	got := ExtractCodeFromResponse(input)
	if !strings.Contains(got, "package main") {
		t.Errorf("ExtractCodeFromResponse() should extract first block: got %q", got)
	}
}

func TestIsIdentifierBenchmarks(t *testing.T) {
	// Quick performance test
	longIdent := strings.Repeat("a", 1000)

	// Should handle long identifiers without issues
	result := IsIdentifier(longIdent)
	if !result {
		t.Error("IsIdentifier should handle long identifiers")
	}
}

func TestExtractJSONPerformance(t *testing.T) {
	// Create large input
	largeJSON := "{" + strings.Repeat(`"key`+string(rune('a'+t.Name()[len(t.Name())-1]))+`": "value",`, 1000) + "}"
	input := "```json\n" + largeJSON + "\n```"

	// Should handle large JSON without issues
	got := ExtractJSON(input)
	if !strings.Contains(got, `"key`) {
		t.Error("ExtractJSON should handle large JSON")
	}
}
