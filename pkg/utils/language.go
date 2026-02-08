package utils

import (
	"path/filepath"
	"strings"
)

// languageExtensionMap maps file extensions to programming languages
var languageExtensionMap = map[string]string{
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

// codeLanguages defines which languages are programming languages
var codeLanguages = map[string]bool{
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

// DetectLanguage detects the programming language from filename
func DetectLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	if lang, ok := languageExtensionMap[ext]; ok {
		return lang
	}

	return "unknown"
}

// IsCodeLanguage checks if the language is a programming language
func IsCodeLanguage(lang string) bool {
	return codeLanguages[lang]
}

// GetSupportedLanguages returns a list of supported programming languages
func GetSupportedLanguages() []string {
	languages := make([]string, 0, len(codeLanguages))
	for lang := range codeLanguages {
		languages = append(languages, lang)
	}
	return languages
}
