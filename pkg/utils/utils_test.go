package utils

import (
	"strings"
	"testing"
)

func TestShouldSkipFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		// Skip extensions
		{"PNG image", "image.png", true},
		{"JPG image", "photo.jpg", true},
		{"JPEG image", "photo.jpeg", true},
		{"GIF image", "animation.gif", true},
		{"SVG image", "icon.svg", true},
		{"ICO file", "favicon.ico", true},
		{"PDF file", "document.pdf", true},
		{"ZIP archive", "archive.zip", true},
		{"TAR archive", "archive.tar", true},
		{"GZ archive", "archive.gz", true},
		{"Lock file", "go.lock", true},
		{"Sum file", "go.sum", true},
		{"Lock JSON", "package-lock.json", true},
		{"Minified JS", "app.min.js", true},
		{"Minified CSS", "style.min.css", true},

		// Skip patterns
		{"Node modules", "node_modules/pkg/index.js", true},
		{"Vendor directory", "vendor/github.com/pkg/lib.go", true},
		{"Git directory", ".git/config", true},
		{"Dist directory", "dist/bundle.js", true},
		{"Build directory", "build/output.exe", true},
		{"Target directory", "target/classes/Main.class", true},

		// Should not skip
		{"Go file", "main.go", false},
		{"Python file", "script.py", false},
		{"JavaScript file", "app.js", false},
		{"TypeScript file", "app.ts", false},
		{"Markdown file", "README.md", false},
		{"YAML file", "config.yaml", false},
		{"JSON file", "config.json", false},

		// Case sensitivity
		{"Uppercase extension", "IMAGE.PNG", true},
		{"Mixed case extension", "Image.Png", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldSkipFile(tt.filename)
			if got != tt.want {
				t.Errorf("ShouldSkipFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestShouldIgnoreFile(t *testing.T) {
	patterns := []string{
		"*.md",
		"docs/**",
		"tests/fixtures/**",
		"vendor/",
		"*.test.js",
	}

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		// Extension patterns
		{"Markdown file", "README.md", true},
		{"Doc file", "docs/guide.md", true},
		{"Test JS file", "app.test.js", true},
		{"Another test JS", "component.test.js", true},

		// Directory patterns
		{"Docs subdirectory", "docs/api/endpoint.md", true},
		{"Test fixtures", "tests/fixtures/data.json", true},
		{"Test fixtures nested", "tests/fixtures/nested/data.xml", true},
		{"Vendor directory", "vendor/lib.go", false}, // Pattern is "vendor/", not "vendor/**"

		// Should not ignore
		{"Go file", "main.go", false},
		{"Src file", "src/main.go", false},
		{"Tests but not fixtures", "tests/main_test.go", false},
		{"Documentation file", "src/doc.go", false},

		// Exact patterns would need exact match
		{"Config file", "config.yaml", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldIgnoreFile(tt.filename, patterns)
			if got != tt.want {
				t.Errorf("ShouldIgnoreFile(%q, patterns) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestShouldIgnoreFileEmptyPatterns(t *testing.T) {
	patterns := []string{}

	filename := "main.go"
	got := ShouldIgnoreFile(filename, patterns)

	if got {
		t.Errorf("ShouldIgnoreFile(%q, empty patterns) = %v, want false", filename, got)
	}
}

func TestShouldIgnoreFileNilPatterns(t *testing.T) {
	var patterns []string

	filename := "main.go"
	got := ShouldIgnoreFile(filename, patterns)

	if got {
		t.Errorf("ShouldIgnoreFile(%q, nil patterns) = %v, want false", filename, got)
	}
}

func TestTruncateContent(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		maxLength int
		want      string
	}{
		{"Empty content", "", 100, ""},
		{"Short content", "Hello", 100, "Hello"},
		{"Exact length", "Hello World", 11, "Hello World"},
		{"Truncate", "Hello World", 5, "Hello\n... (truncated)"},
		{"Truncate at boundary", "Hello World", 6, "Hello \n... (truncated)"},
		{"Zero max length", "Hello", 0, "\n... (truncated)"},
		{"Negative max length", "Hello", -1, "\n... (truncated)"},
		{"Unicode content", "こんにちは", 3, "こ\n... (truncated)"},
		{"Multiline content", "Line 1\nLine 2\nLine 3", 10, "Line 1\nLi\n... (truncated)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateContent(tt.content, tt.maxLength)
			if got != tt.want {
				t.Errorf("TruncateContent(%q, %d) = %q, want %q", tt.content, tt.maxLength, got, tt.want)
			}
		})
	}
}

func TestTruncateContentWithExactBoundary(t *testing.T) {
	content := "Hello World"
	maxLength := len(content)

	got := TruncateContent(content, maxLength)
	if got != content {
		t.Errorf("TruncateContent should not truncate at exact boundary: got %q, want %q", got, content)
	}
}

func TestTruncateContentWithTruncationIndicator(t *testing.T) {
	content := "This is a long string that should be truncated"
	maxLength := 20

	got := TruncateContent(content, maxLength)

	if !strings.HasSuffix(got, "\n... (truncated)") {
		t.Error("TruncateContent should add truncation indicator")
	}

	if len(got) <= maxLength {
		t.Error("Truncated content length should account for truncation indicator")
	}
}

func TestShouldSkipFileEdgeCases(t *testing.T) {
	// Test empty filename
	if ShouldSkipFile("") {
		t.Error("empty filename should not be skipped")
	}

	// Test filename with no extension
	if ShouldSkipFile("README") {
		t.Error("file without extension should not be skipped")
	}

	// Test filename with multiple dots
	if ShouldSkipFile("file.test.go") {
		t.Error("file.test.go should not be skipped")
	}
	if !ShouldSkipFile("file.min.js") {
		t.Error("file.min.js should be skipped")
	}
}

func TestShouldIgnoreFilePatternMatching(t *testing.T) {
	patterns := []string{
		"*.log",
		"temp/**",
		"config.local.yaml",
	}

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"Log file", "app.log", true},
		{"Log in subdir", "logs/app.log", false}, // Pattern is *.log, not **/*.log
		{"Temp file", "temp/cache.txt", true},
		{"Temp nested", "temp/data/file.txt", true},
		{"Exact match", "config.local.yaml", true},
		{"Similar but not exact", "config.local.yml", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldIgnoreFile(tt.filename, patterns)
			if got != tt.want {
				t.Errorf("ShouldIgnoreFile(%q, patterns) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestShouldSkipFilePerformanceFiles(t *testing.T) {
	// Test common performance-critical files that should be skipped
	performanceFiles := []string{
		"bundle.min.js",
		"vendor.js",
		"node_modules/react/index.js",
		"vendor/github.com/gin-gonic/gin/gin.go",
		"dist/bundle.js",
		"build/app.exe",
		"target/classes/App.class",
	}

	for _, filename := range performanceFiles {
		if !ShouldSkipFile(filename) {
			t.Errorf("ShouldSkipFile(%q) should return true for performance-critical file", filename)
		}
	}
}

func TestShouldSkipFileCodeFiles(t *testing.T) {
	// Test common code files that should NOT be skipped
	codeFiles := []string{
		"main.go",
		"app.py",
		"index.js",
		"app.ts",
		"component.tsx",
		"App.java",
		"lib.rs",
		"main.kt",
		"Program.cs",
		"app.rb",
		"index.php",
	}

	for _, filename := range codeFiles {
		if ShouldSkipFile(filename) {
			t.Errorf("ShouldSkipFile(%q) should return false for code file", filename)
		}
	}
}

func TestTruncateContentPreservesContent(t *testing.T) {
	content := "This is important content that should be preserved"
	maxLength := 25

	got := TruncateContent(content, maxLength)

	// Should start with the beginning of content
	if !strings.HasPrefix(got, "This is important conten") {
		t.Errorf("TruncateContent should preserve beginning of content: got %q", got)
	}
}

func TestShouldIgnoreFileDirectoryPatterns(t *testing.T) {
	patterns := []string{
		"node_modules/**",
		"**/testfixtures/**",
	}

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"Node modules direct", "node_modules/pkg/index.js", true},
		{"Node modules nested", "node_modules/pkg/sub/index.js", true},
		{"Test fixtures anywhere", "src/testfixtures/data.json", true},
		{"Test fixtures nested", "src/pkg/testfixtures/data.json", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldIgnoreFile(tt.filename, patterns)
			if got != tt.want {
				t.Errorf("ShouldIgnoreFile(%q, patterns) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestShouldSkipFileSpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"Spaces in name", "my file.png", true},
		{"Special chars", "file@#$%.jpg", true},
		{"Unicode in name", "файл.png", true},
		{"Hyphen in name", "my-file.min.js", true},
		{"Underscore in name", "my_file.lock", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldSkipFile(tt.filename)
			if got != tt.want {
				t.Errorf("ShouldSkipFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestTruncateContentVeryLongContent(t *testing.T) {
	// Create very long content
	content := strings.Repeat("a", 10000)
	maxLength := 100

	got := TruncateContent(content, maxLength)

	if !strings.HasSuffix(got, "\n... (truncated)") {
		t.Error("TruncateContent should add truncation indicator for very long content")
	}

	// Should preserve first maxLength characters
	if len(got) < maxLength {
		t.Errorf("TruncateContent should preserve at least %d characters, got %d", maxLength, len(got))
	}
}

func TestShouldIgnoreFileCaseSensitivity(t *testing.T) {
	patterns := []string{
		"*.MD",
		"DOCS/**",
	}

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"Case mismatch", "README.md", false},
		{"Case match upper", "README.MD", true},
		{"Case mismatch dir", "docs/file.txt", false},
		{"Case match dir", "DOCS/file.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldIgnoreFile(tt.filename, patterns)
			if got != tt.want {
				t.Errorf("ShouldIgnoreFile(%q, patterns) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestShouldSkipFileHiddenFiles(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"Hidden file", ".gitignore", false},
		{"Hidden dir", ".github/workflows/ci.yml", false},
		{"Dotenv", ".env", false},
		{"Hidden in dir", "config/.secret", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldSkipFile(tt.filename)
			if got != tt.want {
				t.Errorf("ShouldSkipFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestTruncateContentEmptyMaxLength(t *testing.T) {
	content := "Hello World"

	got := TruncateContent(content, 0)
	expected := "\n... (truncated)"

	if got != expected {
		t.Errorf("TruncateContent(%q, 0) = %q, want %q", content, got, expected)
	}
}

func TestShouldIgnoreFileMultiplePatterns(t *testing.T) {
	patterns := []string{
		"*.md",
		"*.txt",
		"docs/**",
		"tests/**",
		"vendor/**",
	}

	filename := "docs/tests/README.txt"

	// Should match multiple patterns
	if !ShouldIgnoreFile(filename, patterns) {
		t.Errorf("ShouldIgnoreFile(%q) should match at least one pattern", filename)
	}
}

func TestShouldSkipFileRelativePaths(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"Relative PNG", "./images/photo.png", true},
		{"Relative vendor", "../vendor/lib.go", true},
		{"Relative node_modules", "../../node_modules/pkg.js", true},
		{"Relative Go file", "./src/main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldSkipFile(tt.filename)
			if got != tt.want {
				t.Errorf("ShouldSkipFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestShouldIgnoreFileGlobPatterns(t *testing.T) {
	patterns := []string{
		"**/*.bak",
		"**/*.tmp",
		"temp*",
	}

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"Backup file anywhere", "src/file.bak", true},
		{"Backup file nested", "a/b/c/file.bak", true},
		{"Temp file anywhere", "file.tmp", true},
		{"Temp prefix", "temp123.txt", true},
		{"Temp prefix in path", "src/temp_file.txt", false}, // Pattern is temp*, not **/temp*
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldIgnoreFile(tt.filename, patterns)
			if got != tt.want {
				t.Errorf("ShouldIgnoreFile(%q, patterns) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}
