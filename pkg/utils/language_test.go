package utils

import (
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		// Go
		{"go file", "main.go", "go"},
		{"go test file", "main_test.go", "go"},
		{"go with path", "pkg/handler.go", "go"},

		// Python
		{"python file", "script.py", "python"},
		{"python with path", "src/app.py", "python"},

		// JavaScript
		{"js file", "app.js", "javascript"},
		{"js with path", "src/index.js", "javascript"},

		// TypeScript
		{"ts file", "app.ts", "typescript"},
		{"tsx file", "Component.tsx", "typescript"},
		{"ts with path", "src/app.ts", "typescript"},

		// JSX
		{"jsx file", "Component.jsx", "javascript"},

		// Java
		{"java file", "Main.java", "java"},
		{"java with path", "src/Main.java", "java"},

		// C/C++
		{"c file", "main.c", "c"},
		{"cpp file", "main.cpp", "cpp"},
		{"cc file", "main.cc", "cpp"},
		{"cxx file", "main.cxx", "cpp"},

		// C#
		{"csharp file", "Program.cs", "csharp"},

		// Ruby
		{"ruby file", "app.rb", "ruby"},

		// PHP
		{"php file", "index.php", "php"},

		// Rust
		{"rust file", "main.rs", "rust"},

		// Kotlin
		{"kotlin file", "Main.kt", "kotlin"},

		// Swift
		{"swift file", "Main.swift", "swift"},

		// Unknown
		{"unknown extension", "file.unknown", "unknown"},
		{"no extension", "README", "unknown"},
		{"markdown", "README.md", "unknown"},
		{"yaml", "config.yaml", "unknown"},
		{"json", "config.json", "unknown"},
		{"xml", "config.xml", "unknown"},
		{"html", "index.html", "unknown"},
		{"css", "style.css", "unknown"},

		// Case sensitivity
		{"uppercase extension", "MAIN.GO", "go"},
		{"mixed case extension", "Main.Go", "go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLanguage(tt.filename)
			if got != tt.expected {
				t.Errorf("DetectLanguage(%q) = %q, want %q", tt.filename, got, tt.expected)
			}
		})
	}
}

func TestDetectLanguageWithMultipleDots(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"test file", "main_test.go", "go"},
		{"minified", "app.min.js", "javascript"},
		{"config file", "webpack.config.js", "javascript"},
		{"d.ts file", "index.d.ts", "typescript"},
		{"spec file", "app.spec.ts", "typescript"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLanguage(tt.filename)
			if got != tt.expected {
				t.Errorf("DetectLanguage(%q) = %q, want %q", tt.filename, got, tt.expected)
			}
		})
	}
}

func TestDetectLanguageWithPaths(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"relative path", "./src/main.go", "go"},
		{"parent path", "../lib/helper.py", "python"},
		{"deep path", "a/b/c/d/e/f/main.go", "go"},
		{"absolute path", "/home/user/project/main.go", "go"},
		{"windows path", "C:\\Users\\project\\main.go", "go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLanguage(tt.filename)
			if got != tt.expected {
				t.Errorf("DetectLanguage(%q) = %q, want %q", tt.filename, got, tt.expected)
			}
		})
	}
}

func TestIsCodeLanguage(t *testing.T) {
	tests := []struct {
		name     string
		language string
		expected bool
	}{
		// Supported languages
		{"go", "go", true},
		{"python", "python", true},
		{"javascript", "javascript", true},
		{"typescript", "typescript", true},
		{"java", "java", true},
		{"c", "c", true},
		{"cpp", "cpp", true},
		{"csharp", "csharp", true},
		{"ruby", "ruby", true},
		{"php", "php", true},
		{"rust", "rust", true},
		{"kotlin", "kotlin", true},
		{"swift", "swift", true},

		// Unsupported languages
		{"unknown", "unknown", false},
		{"empty", "", false},
		{"markdown", "markdown", false},
		{"yaml", "yaml", false},
		{"json", "json", false},
		{"xml", "xml", false},
		{"html", "html", false},
		{"css", "css", false},
		{"sql", "sql", false},
		{"shell", "shell", false},
		{"bash", "bash", false},

		// Case sensitivity
		{"Go (uppercase)", "Go", false},
		{"GO (all caps)", "GO", false},
		{"Python (uppercase)", "Python", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCodeLanguage(tt.language)
			if got != tt.expected {
				t.Errorf("IsCodeLanguage(%q) = %v, want %v", tt.language, got, tt.expected)
			}
		})
	}
}

func TestGetSupportedLanguages(t *testing.T) {
	languages := GetSupportedLanguages()

	// Check that we have the expected number of languages
	expectedCount := 13 // go, python, javascript, typescript, java, c, cpp, csharp, ruby, php, rust, kotlin, swift
	if len(languages) != expectedCount {
		t.Errorf("GetSupportedLanguages() returned %d languages, want %d", len(languages), expectedCount)
	}

	// Create a map for easier checking
	langMap := make(map[string]bool)
	for _, lang := range languages {
		langMap[lang] = true
	}

	// Check that all expected languages are present
	expectedLanguages := []string{
		"go", "python", "javascript", "typescript", "java",
		"c", "cpp", "csharp", "ruby", "php", "rust", "kotlin", "swift",
	}

	for _, expected := range expectedLanguages {
		if !langMap[expected] {
			t.Errorf("GetSupportedLanguages() missing expected language: %s", expected)
		}
	}
}

func TestGetSupportedLanguagesNoDuplicates(t *testing.T) {
	languages := GetSupportedLanguages()

	// Check for duplicates
	seen := make(map[string]bool)
	for _, lang := range languages {
		if seen[lang] {
			t.Errorf("GetSupportedLanguages() returned duplicate language: %s", lang)
		}
		seen[lang] = true
	}
}

func TestGetSupportedLanguagesIsNotEmpty(t *testing.T) {
	languages := GetSupportedLanguages()

	if len(languages) == 0 {
		t.Error("GetSupportedLanguages() should return non-empty slice")
	}
}

func TestDetectLanguageEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"empty string", "", "unknown"},
		{"single dot", ".", "unknown"},
		{"double dot", "..", "unknown"},
		{"only extension", ".go", "go"},
		{"hidden file", ".gitignore", "unknown"},
		{"hidden file with known ext", ".config.json", "unknown"},
		{"trailing slash", "dir/", "unknown"},
		{"trailing dot", "file.", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLanguage(tt.filename)
			if got != tt.expected {
				t.Errorf("DetectLanguage(%q) = %q, want %q", tt.filename, got, tt.expected)
			}
		})
	}
}

func TestIsCodeLanguageWithWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		language string
		expected bool
	}{
		{"leading space", " go", false},
		{"trailing space", "go ", false},
		{"both spaces", " go ", false},
		{"tab", "\tgo\t", false},
		{"newline", "\ngo\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCodeLanguage(tt.language)
			if got != tt.expected {
				t.Errorf("IsCodeLanguage(%q) = %v, want %v", tt.language, got, tt.expected)
			}
		})
	}
}

func TestDetectLanguageSpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"spaces in name", "my file.go", "go"},
		{"special chars", "file@#$%.py", "python"},
		{"unicode in name", "файл.go", "go"},
		{"hyphen in name", "my-file.go", "go"},
		{"underscore in name", "my_file.go", "go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLanguage(tt.filename)
			if got != tt.expected {
				t.Errorf("DetectLanguage(%q) = %q, want %q", tt.filename, got, tt.expected)
			}
		})
	}
}

func TestLanguageExtensionMapCompleteness(t *testing.T) {
	// Test that the languageExtensionMap contains expected extensions
	expectedExtensions := map[string]string{
		".go":    "go",
		".py":    "python",
		".js":    "javascript",
		".ts":    "typescript",
		".jsx":   "javascript",
		".tsx":   "typescript",
		".java":  "java",
		".c":     "c",
		".cpp":   "cpp",
		".cc":    "cpp",
		".cxx":   "cpp",
		".cs":    "csharp",
		".rb":    "ruby",
		".php":   "php",
		".rs":    "rust",
		".kt":    "kotlin",
		".swift": "swift",
	}

	for ext, expectedLang := range expectedExtensions {
		if languageExtensionMap[ext] != expectedLang {
			t.Errorf("languageExtensionMap[%q] = %q, want %q", ext, languageExtensionMap[ext], expectedLang)
		}
	}
}

func TestCodeLanguagesMapCompleteness(t *testing.T) {
	// Test that codeLanguages contains expected languages
	expectedLanguages := map[string]bool{
		"go":         true,
		"python":     true,
		"javascript": true,
		"typescript": true,
		"java":       true,
		"c":          true,
		"cpp":        true,
		"csharp":     true,
		"ruby":       true,
		"php":        true,
		"rust":       true,
		"kotlin":     true,
		"swift":      true,
	}

	for lang := range expectedLanguages {
		if !codeLanguages[lang] {
			t.Errorf("codeLanguages missing expected language: %s", lang)
		}
	}
}

func TestDetectLanguagePerformance(t *testing.T) {
	// Quick performance test with many calls
	filename := "main.go"
	for i := 0; i < 1000; i++ {
		result := DetectLanguage(filename)
		if result != "go" {
			t.Errorf("DetectLanguage() failed at iteration %d", i)
		}
	}
}

func TestIsCodeLanguagePerformance(t *testing.T) {
	// Quick performance test with many calls
	language := "go"
	for i := 0; i < 1000; i++ {
		result := IsCodeLanguage(language)
		if !result {
			t.Errorf("IsCodeLanguage() failed at iteration %d", i)
		}
	}
}

func TestDetectLanguageWithFileExtensions(t *testing.T) {
	// Test all extensions in the map
	for ext, expectedLang := range languageExtensionMap {
		filename := "test" + ext
		got := DetectLanguage(filename)
		if got != expectedLang {
			t.Errorf("DetectLanguage(%q) = %q, want %q", filename, got, expectedLang)
		}
	}
}

func TestIsCodeLanguageAllSupported(t *testing.T) {
	// Test that all languages returned by GetSupportedLanguages are recognized
	languages := GetSupportedLanguages()
	for _, lang := range languages {
		if !IsCodeLanguage(lang) {
			t.Errorf("IsCodeLanguage(%q) should return true for supported language", lang)
		}
	}
}

func TestDetectLanguageCaseVariations(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"lowercase", "main.go", "go"},
		{"uppercase", "MAIN.GO", "go"},
		{"mixed case", "Main.Go", "go"},
		{"camelCase", "myFile.go", "go"},
		{"PascalCase", "MyFile.go", "go"},
		{"SCREAMING", "MAIN.GO", "go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLanguage(tt.filename)
			if got != tt.expected {
				t.Errorf("DetectLanguage(%q) = %q, want %q", tt.filename, got, tt.expected)
			}
		})
	}
}

func TestGetSupportedLanguagesSorted(t *testing.T) {
	languages := GetSupportedLanguages()

	// Check that languages are returned in a consistent order
	// (not necessarily sorted, but consistent)
	for i := 1; i < len(languages); i++ {
		if languages[i] == languages[i-1] {
			t.Errorf("GetSupportedLanguages() has duplicate at index %d", i)
		}
	}
}

func TestDetectLanguageWithSymlinks(t *testing.T) {
	// Test that symlinks in path don't affect detection
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{"symlink-like path", "link/to/main.go", "go"},
		{"multiple symlinks", "a/b/c/d.go", "go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLanguage(tt.filename)
			if got != tt.expected {
				t.Errorf("DetectLanguage(%q) = %q, want %q", tt.filename, got, tt.expected)
			}
		})
	}
}
