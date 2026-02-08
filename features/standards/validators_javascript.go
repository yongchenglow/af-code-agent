package standards

import (
	"fmt"
	"regexp"
	"strings"
)

// JSValidator handles JavaScript/TypeScript-specific validation
type JSValidator struct{}

// NewJSValidator creates a new JSValidator
func NewJSValidator() *JSValidator {
	return &JSValidator{}
}

// CheckFunctionLength checks JavaScript/TypeScript function length
func (j *JSValidator) CheckFunctionLength(ctx RuleContext, maxLength int) []*Violation {
	var violations []*Violation
	lines := strings.Split(ctx.Content, "\n")

	funcPattern := regexp.MustCompile(`function\s+(\w+)|const\s+(\w+)\s*=\s*\(.*\)\s*=>|(\w+)\s*\(.*\)\s*{`)
	currentFunc := ""
	funcStartLine := 0
	braceCount := 0
	inFunc := false

	for i, line := range lines {
		if funcPattern.MatchString(line) {
			matches := funcPattern.FindStringSubmatch(line)
			for _, match := range matches[1:] {
				if match != "" {
					currentFunc = match
					break
				}
			}
			funcStartLine = i + 1
			braceCount = strings.Count(line, "{") - strings.Count(line, "}")
			inFunc = true
			continue
		}

		if inFunc {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount == 0 {
				funcLength := i - funcStartLine + 1
				if funcLength > maxLength {
					violations = append(violations, &Violation{
						FilePath:    ctx.FilePath,
						Line:        funcStartLine,
						Rule:        "function-length",
						Message:     fmt.Sprintf("Function '%s' exceeds maximum length of %d lines (current: %d)", currentFunc, maxLength, funcLength),
						Suggestion:  "Consider breaking this function into smaller functions",
						AutoFixable: false,
					})
				}
				inFunc = false
			}
		}
	}

	return violations
}
