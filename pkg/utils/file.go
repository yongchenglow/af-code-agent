package utils

import (
	"strings"
)

// skipExtensions are file extensions that should be skipped during review
var skipExtensions = []string{
	".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico",
	".pdf", ".zip", ".tar", ".gz",
	".lock", ".sum", "lock.json",
	".min.js", ".min.css",
}

// skipPatterns are path patterns that should be skipped during review
var skipPatterns = []string{
	"node_modules/",
	"vendor/",
	".git/",
	"dist/",
	"build/",
	"target/",
}

// ShouldSkipFile determines if a file should be skipped during review
func ShouldSkipFile(filename string) bool {
	lowerFilename := strings.ToLower(filename)

	// Check extensions
	for _, ext := range skipExtensions {
		if strings.HasSuffix(lowerFilename, ext) {
			return true
		}
	}

	// Check patterns
	for _, pattern := range skipPatterns {
		if strings.Contains(filename, pattern) {
			return true
		}
	}

	return false
}

// ShouldIgnoreFile checks if a file should be ignored based on patterns
func ShouldIgnoreFile(filename string, ignorePatterns []string) bool {
	for _, pattern := range ignorePatterns {
		// Simple pattern matching (can be enhanced with glob patterns)
		if strings.HasSuffix(pattern, "**") {
			// Directory pattern
			prefix := strings.TrimSuffix(pattern, "**")
			if strings.HasPrefix(filename, prefix) {
				return true
			}
		} else if strings.HasPrefix(pattern, "*.") {
			// Extension pattern
			ext := strings.TrimPrefix(pattern, "*")
			if strings.HasSuffix(filename, ext) {
				return true
			}
		} else if filename == pattern {
			// Exact match
			return true
		}
	}
	return false
}

// TruncateContent truncates content to maxLength with truncation indicator
func TruncateContent(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}
	return content[:maxLength] + "\n... (truncated)"
}
