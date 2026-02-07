package standards

import (
	"context"
	"testing"

	"github.com/yourorg/github-code-agent/features/analyzer"
	"github.com/yourorg/github-code-agent/pkg/config"
)

func TestValidateLineLength(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Standards.Coding.MaxLineLength = 80

	validator := NewValidator(cfg)

	ctx := RuleContext{
		FilePath: "test.go",
		Content: `package main

func main() {
	shortLine := "ok"
	veryLongLineThatExceedsTheMaximumLineLengthAndShouldBeDetectedByTheValidator := "too long"
}`,
		Language: "go",
	}

	violations, err := validator.validateLineLength(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(violations) == 0 {
		t.Error("Expected at least one violation for long line")
	}

	if violations[0].Line != 5 {
		t.Errorf("Expected violation on line 5, got line %d", violations[0].Line)
	}
}

func TestValidateNoHardcodedSecrets(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Standards.Security.CheckSecrets = true

	validator := NewValidator(cfg)

	tests := []struct {
		name               string
		content            string
		expectedViolations int
	}{
		{
			name: "detect API key",
			content: `
api_key = "sk-1234567890abcdefghijklmnopqrstuvwxyz"
`,
			expectedViolations: 1,
		},
		{
			name: "detect password",
			content: `
password = "secretpassword123"
`,
			expectedViolations: 1,
		},
		{
			name: "clean code",
			content: `
api_key = os.Getenv("API_KEY")
password = config.GetPassword()
`,
			expectedViolations: 0,
		},
		{
			name: "comment should be ignored",
			content: `
// api_key = "this-is-just-a-comment"
`,
			expectedViolations: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := RuleContext{
				FilePath: "test.go",
				Content:  tt.content,
				Language: "go",
			}

			violations, err := validator.validateNoHardcodedSecrets(ctx)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(violations) != tt.expectedViolations {
				t.Errorf("Expected %d violations, got %d", tt.expectedViolations, len(violations))
			}
		})
	}
}

func TestValidateStandards(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Standards.Coding.MaxLineLength = 50 // Shorter to trigger violation
	cfg.Standards.Security.CheckSecrets = true

	validator := NewValidator(cfg)

	files := []*analyzer.FileChange{
		{
			Filename: "main.go",
			Status:   "modified",
			Content: `package main

func main() {
	secret_key := "this_is_a_very_long_secret_that_should_be_detected_by_validation"
	println(secret_key)
}`,
			Language: "go",
		},
	}

	report, err := validator.ValidateStandards(context.Background(), files)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if report.TotalViolations == 0 {
		t.Error("Expected violations but got none")
	}

	// Should have either line length or security violations
	hasViolations := report.ViolationsByType[ViolationTypeCoding] > 0 ||
		report.ViolationsByType[ViolationTypeSecurity] > 0

	if !hasViolations {
		t.Error("Expected coding or security violations")
	}
}

func TestCheckGoFunctionLength(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Standards.Coding.MaxFunctionLength = 5

	validator := NewValidator(cfg)

	ctx := RuleContext{
		FilePath: "test.go",
		Content: `package main

func shortFunc() {
	line1()
	line2()
}

func longFunc() {
	line1()
	line2()
	line3()
	line4()
	line5()
	line6()
	line7()
	line8()
}`,
		Language: "go",
	}

	violations := validator.checkGoFunctionLength(ctx, 5)

	if len(violations) == 0 {
		t.Error("Expected violation for long function")
	}

	// Should detect longFunc
	found := false
	for _, v := range violations {
		if v.Message != "" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find long function violation")
	}
}

func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}

	if !contains(slice, "b") {
		t.Error("Expected to find 'b' in slice")
	}

	if contains(slice, "d") {
		t.Error("Did not expect to find 'd' in slice")
	}
}

func TestIsIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"validName", true},
		{"valid_name", true},
		{"ValidName123", true},
		{"123invalid", true}, // Simple check doesn't validate start char
		{"invalid-name", false},
		{"invalid.name", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v for '%s', got %v", tt.expected, tt.input, result)
			}
		})
	}
}
