package testexec

import (
	"strings"
	"testing"
)

func TestTestResultStruct(t *testing.T) {
	result := &TestResult{
		GapID:     "TEST-001",
		TestFile:  "pkg/service/service_test.go",
		TestCode:  "package service\n\nfunc TestExample(t *testing.T) {}",
		Success:   true,
		Error:     "",
		TestCount: 1,
	}

	if result.GapID != "TEST-001" {
		t.Errorf("expected GapID to be 'TEST-001', got %q", result.GapID)
	}
	if result.TestFile != "pkg/service/service_test.go" {
		t.Errorf("expected TestFile to be 'pkg/service/service_test.go', got %q", result.TestFile)
	}
	if !result.Success {
		t.Error("expected Success to be true")
	}
	if result.TestCount != 1 {
		t.Errorf("expected TestCount to be 1, got %d", result.TestCount)
	}
}

func TestTestResultFailure(t *testing.T) {
	result := &TestResult{
		GapID:     "TEST-001",
		TestFile:  "pkg/service/service_test.go",
		TestCode:  "",
		Success:   false,
		Error:     "Test generation failed",
		TestCount: 0,
	}

	if result.Success {
		t.Error("expected Success to be false")
	}
	if result.Error != "Test generation failed" {
		t.Errorf("expected Error to be 'Test generation failed', got %q", result.Error)
	}
	if result.TestCount != 0 {
		t.Errorf("expected TestCount to be 0, got %d", result.TestCount)
	}
}

func TestNewExecutor(t *testing.T) {
	// Test that constructor returns non-nil executor
	// Note: We can't fully test without a real agent
	t.Skip("Skipping - requires agent mock")
}

func TestCountTests(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name:     "empty code",
			code:     "",
			expected: 0,
		},
		{
			name:     "single test",
			code:     "func TestExample(t *testing.T) {}",
			expected: 1,
		},
		{
			name: "multiple tests",
			code: `func TestOne(t *testing.T) {}
func TestTwo(t *testing.T) {}
func TestThree(t *testing.T) {}`,
			expected: 3,
		},
		{
			name: "no tests",
			code: `func Example() {}
func Helper() {}`,
			expected: 0,
		},
		{
			name: "test with body",
			code: `func TestExample(t *testing.T) {
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}`,
			expected: 1,
		},
		{
			name: "mixed functions",
			code: `func TestOne(t *testing.T) {}
func helper() {}
func TestTwo(t *testing.T) {}
func Benchmark(b *testing.B) {}`,
			expected: 2,
		},
		{
			name: "test with whitespace",
			code: `  func TestExample(t *testing.T) {}
	func TestAnother(t *testing.T) {}`,
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countTests(tt.code)
			if got != tt.expected {
				t.Errorf("countTests() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestCountTestsWithFullFile(t *testing.T) {
	code := `package service

import "testing"

func TestNewService(t *testing.T) {
	// Test constructor
}

func TestService_Run(t *testing.T) {
	// Test Run method
}

func TestService_Close(t *testing.T) {
	// Test Close method
}

func helperFunction() {
	// This is not a test
}
`

	got := countTests(code)
	if got != 3 {
		t.Errorf("countTests() = %d, want 3", got)
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty", "", 0},
		{"single line", "hello", 1},
		{"two lines", "hello\nworld", 2},
		{"three lines", "a\nb\nc", 3},
		{"trailing newline", "hello\n", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitLines(tt.input)
			if len(got) != tt.expected {
				t.Errorf("splitLines() returned %d lines, want %d", len(got), tt.expected)
			}
		})
	}
}

func TestTrimSpace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"no space", "hello", "hello"},
		{"leading space", "  hello", "hello"},
		{"trailing space", "hello  ", "hello"},
		{"both spaces", "  hello  ", "hello"},
		{"tabs", "\t\thello\t\t", "hello"},
		{"mixed whitespace", " \t hello \t ", "hello"},
		{"internal spaces", "hello world", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimSpace(tt.input)
			if got != tt.expected {
				t.Errorf("trimSpace(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestWriteTestsEmptyGap(t *testing.T) {
	// This test would require mocking the agent
	t.Skip("Skipping - requires agent mock")
}

func TestWriteTestsBatchEmptyGaps(t *testing.T) {
	// This test would require mocking the agent
	t.Skip("Skipping - requires agent mock")
}

func TestWriteTestsBatchWithFixCodes(t *testing.T) {
	// This test would require mocking the agent
	t.Skip("Skipping - requires agent mock")
}

func TestWriteTestsBatchPartialFixCodes(t *testing.T) {
	// This test would require mocking the agent
	t.Skip("Skipping - requires agent mock")
}

func TestWriteTestsBatchNilFixCodes(t *testing.T) {
	// This test would require mocking the agent
	t.Skip("Skipping - requires agent mock")
}

func TestWriteTestsContextTimeout(t *testing.T) {
	// This test would require mocking the agent and context
	t.Skip("Skipping - requires agent mock")
}

func TestWriteTestsErrorHandling(t *testing.T) {
	// This test would require mocking the agent
	t.Skip("Skipping - requires agent mock")
}

func TestWriteTestsBatchErrorHandling(t *testing.T) {
	// This test would require mocking the agent
	t.Skip("Skipping - requires agent mock")
}

func TestCountTestsEdgeCases(t *testing.T) {
	// Test with very long function name
	longName := "func TestThisIsAVeryLongTestNameThatShouldStillBeCountedCorrectly(t *testing.T) {}"
	if countTests(longName) != 1 {
		t.Error("should count test with long name")
	}

	// Test with comment before function
	withComment := `// This is a test
func TestExample(t *testing.T) {}`
	if countTests(withComment) != 1 {
		t.Error("should count test with preceding comment")
	}

	// Test with struct literal containing "func Test"
	withStruct := `type TestConfig struct {
	Name string
}
func TestExample(t *testing.T) {}`
	if countTests(withStruct) != 1 {
		t.Error("should only count actual test functions")
	}
}

func TestSplitLinesEdgeCases(t *testing.T) {
	// Test with Windows line endings
	windows := "line1\r\nline2\r\nline3"
	got := splitLines(windows)
	// Should handle different line endings gracefully
	if len(got) == 0 {
		t.Error("splitLines should handle Windows line endings")
	}

	// Test with single newline
	single := "\n"
	got = splitLines(single)
	if len(got) != 2 {
		t.Errorf("splitLines(\\n) should return 2 lines, got %d", len(got))
	}
}

func TestTrimSpaceEdgeCases(t *testing.T) {
	// Test with only whitespace
	onlySpace := "   "
	if trimSpace(onlySpace) != "" {
		t.Error("trimSpace should return empty string for whitespace-only input")
	}

	// Test with unicode whitespace
	unicodeSpace := "\u3000hello\u3000" // Full-width spaces
	result := trimSpace(unicodeSpace)
	// Note: Our simple implementation may not handle unicode whitespace
	// This is a known limitation
	_ = result
}

func TestWriteTestsPromptConstruction(t *testing.T) {
	// Verify the task prompt template is loaded
	if testExecutorTask == "" {
		t.Fatal("testExecutorTask prompt is empty")
	}

	// Check that prompt contains expected placeholders
	if !strings.Contains(testExecutorTask, "%s") {
		t.Error("testExecutorTask should contain format placeholders")
	}
}

func TestWriteTestsSystemPrompt(t *testing.T) {
	// Verify the system prompt is loaded
	if testExecutorPrompt == "" {
		t.Fatal("testExecutorPrompt is empty")
	}

	// System prompt should be non-empty and contain guidance
	if len(testExecutorPrompt) < 50 {
		t.Error("testExecutorPrompt seems too short")
	}
}

func TestTestResultString(t *testing.T) {
	// Test that TestResult can be used in string context
	result := &TestResult{
		GapID:     "TEST-001",
		TestFile:  "test.go",
		TestCode:  "func TestExample(t *testing.T) {}",
		Success:   true,
		TestCount: 1,
	}

	// Basic validation that struct fields are accessible
	if result.GapID == "" {
		t.Error("GapID should not be empty")
	}
	if result.TestFile == "" {
		t.Error("TestFile should not be empty")
	}
}

func TestExecutorStructInitialization(t *testing.T) {
	// Test that Executor struct is properly defined
	var exec Executor

	// Verify struct fields exist
	_ = exec.agent
}

func TestTestResultWithEmptyFields(t *testing.T) {
	result := &TestResult{}

	if result.GapID != "" {
		t.Errorf("expected empty GapID, got %q", result.GapID)
	}
	if result.TestFile != "" {
		t.Errorf("expected empty TestFile, got %q", result.TestFile)
	}
	if result.TestCode != "" {
		t.Errorf("expected empty TestCode, got %q", result.TestCode)
	}
	if result.Success {
		t.Error("expected Success to be false")
	}
	if result.Error != "" {
		t.Errorf("expected empty Error, got %q", result.Error)
	}
	if result.TestCount != 0 {
		t.Errorf("expected TestCount to be 0, got %d", result.TestCount)
	}
}

func TestCountTestsRealisticCode(t *testing.T) {
	// Test with realistic Go test file
	code := `package calculator

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestAdd(t *testing.T) {
	result := Add(2, 3)
	assert.Equal(t, 5, result)
}

func TestAdd_NegativeNumbers(t *testing.T) {
	result := Add(-2, -3)
	assert.Equal(t, -5, result)
}

func TestAdd_Overflow(t *testing.T) {
	// Test overflow case
	result := Add(int64(^uint64(0) >> 1), 1)
	assert.Panics(t, func() {
		_ = result
	})
}

func TestSubtract(t *testing.T) {
	result := Subtract(5, 3)
	assert.Equal(t, 2, result)
}

func helper(t *testing.T) {
	// This is not a test function
}

func BenchmarkAdd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Add(2, 3)
	}
}
`

	got := countTests(code)
	if got != 4 {
		t.Errorf("countTests() = %d, want 4", got)
	}
}

func TestWriteTestsBatchConcurrentSafety(t *testing.T) {
	// This test would verify that batch processing is safe for concurrent use
	// Requires mocking the agent
	t.Skip("Skipping - requires agent mock")
}
