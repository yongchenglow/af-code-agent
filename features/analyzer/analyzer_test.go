package analyzer

import (
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"main.go", "go"},
		{"script.py", "python"},
		{"app.js", "javascript"},
		{"component.tsx", "typescript"},
		{"README.md", "unknown"},
		{"test.java", "java"},
		{"program.rs", "rust"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := detectLanguage(tt.filename)
			if got != tt.want {
				t.Errorf("detectLanguage(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestShouldIgnoreFile(t *testing.T) {
	patterns := []string{
		"*.md",
		"docs/**",
		"tests/fixtures/**",
	}

	tests := []struct {
		filename string
		want     bool
	}{
		{"README.md", true},
		{"docs/guide.txt", true},
		{"tests/fixtures/data.json", true},
		{"src/main.go", false},
		{"config.yaml", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := ShouldIgnoreFile(tt.filename, patterns)
			if got != tt.want {
				t.Errorf("ShouldIgnoreFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestCalculateComplexity(t *testing.T) {
	code := `package main

import "fmt"

func main() {
    fmt.Println("Hello")
}
`
	metrics, err := CalculateComplexity(code)
	if err != nil {
		t.Fatalf("CalculateComplexity() error = %v", err)
	}

	if metrics.LinesOfCode == 0 {
		t.Error("Expected non-zero lines of code")
	}
}

func TestParseCodeStructure(t *testing.T) {
	code := `func example() {}`

	ast, err := ParseCodeStructure(code, "go")
	if err != nil {
		t.Fatalf("ParseCodeStructure() error = %v", err)
	}

	if ast.Language != "go" {
		t.Errorf("Expected language 'go', got %q", ast.Language)
	}
}
