package utils

import "strings"

// ExtractJSON attempts to extract JSON from text (handles markdown code blocks)
func ExtractJSON(text string) string {
	// Remove markdown code blocks if present
	text = strings.TrimSpace(text)

	// Check for ```json blocks
	if strings.Contains(text, "```json") {
		start := strings.Index(text, "```json") + 7
		end := strings.Index(text[start:], "```")
		if end > 0 {
			return strings.TrimSpace(text[start : start+end])
		}
	}

	// Check for ``` blocks
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		if len(lines) > 2 {
			return strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	// Try to find JSON array first (more specific)
	if idx := strings.Index(text, "["); idx >= 0 {
		return text[idx:]
	}
	// Then try JSON object
	if idx := strings.Index(text, "{"); idx >= 0 {
		return text[idx:]
	}

	return text
}

// ExtractCodeFromResponse extracts code from AI response
func ExtractCodeFromResponse(response string) string {
	// Remove markdown code blocks if present
	code := response

	// Check for ```language blocks
	if strings.Contains(code, "```") {
		parts := strings.Split(code, "```")
		if len(parts) >= 3 {
			// Get the code between first ``` and second ```
			code = parts[1]
			// Remove language identifier if present
			lines := strings.Split(code, "\n")
			if len(lines) > 0 {
				// First line might be language identifier
				firstLine := strings.TrimSpace(lines[0])
				if isLanguageIdentifier(firstLine) {
					code = strings.Join(lines[1:], "\n")
				}
			}
		}
	}

	return strings.TrimSpace(code)
}

// isLanguageIdentifier checks if a string is a common language identifier
func isLanguageIdentifier(s string) bool {
	commonLanguages := map[string]bool{
		"go":         true,
		"python":     true,
		"javascript": true,
		"typescript": true,
		"java":       true,
		"rust":       true,
		"cpp":        true,
		"c":          true,
		"ruby":       true,
		"php":        true,
		"kotlin":     true,
		"swift":      true,
	}
	return commonLanguages[s]
}

// Contains checks if a slice contains an item
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// IsIdentifier checks if a word is a valid identifier
func IsIdentifier(word string) bool {
	if len(word) == 0 {
		return false
	}
	// Simple check - real implementation would be more robust
	for _, ch := range word {
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '_' {
			return false
		}
	}
	return true
}
