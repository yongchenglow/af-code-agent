package standards

import (
	"fmt"
	"regexp"
	"strings"
)

// GoValidator handles Go-specific validation
type GoValidator struct{}

// NewGoValidator creates a new GoValidator
func NewGoValidator() *GoValidator {
	return &GoValidator{}
}

// CheckFunctionLength checks Go function length
func (g *GoValidator) CheckFunctionLength(ctx RuleContext, maxLength int) []*Violation {
	var violations []*Violation
	lines := strings.Split(ctx.Content, "\n")

	funcPattern := regexp.MustCompile(`^func\s+(\w+)`)
	currentFunc := ""
	funcStartLine := 0
	braceCount := 0
	inFunc := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if funcPattern.MatchString(trimmed) {
			matches := funcPattern.FindStringSubmatch(trimmed)
			currentFunc = matches[1]
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

// CheckDocumentation checks Go documentation requirements
func (g *GoValidator) CheckDocumentation(ctx RuleContext, lines []string) []*Violation {
	var violations []*Violation

	funcPattern := regexp.MustCompile(`^func\s+(\w+)`)
	typePattern := regexp.MustCompile(`^type\s+(\w+)`)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for exported functions
		if funcPattern.MatchString(trimmed) {
			matches := funcPattern.FindStringSubmatch(trimmed)
			funcName := matches[1]

			if g.isExported(funcName) {
				if !g.hasDocComment(lines, i) {
					violations = append(violations, &Violation{
						FilePath:    ctx.FilePath,
						Line:        i + 1,
						Rule:        "missing-docs",
						Message:     fmt.Sprintf("Exported function '%s' is missing documentation", funcName),
						Suggestion:  fmt.Sprintf("Add a comment like: // %s ...", funcName),
						AutoFixable: false,
					})
				}
			}
		}

		// Check for exported types
		if typePattern.MatchString(trimmed) {
			matches := typePattern.FindStringSubmatch(trimmed)
			typeName := matches[1]

			if g.isExported(typeName) {
				if !g.hasDocComment(lines, i) {
					violations = append(violations, &Violation{
						FilePath:    ctx.FilePath,
						Line:        i + 1,
						Rule:        "missing-docs",
						Message:     fmt.Sprintf("Exported type '%s' is missing documentation", typeName),
						Suggestion:  fmt.Sprintf("Add a comment like: // %s ...", typeName),
						AutoFixable: false,
					})
				}
			}
		}
	}

	return violations
}

// isExported checks if a name is exported (starts with capital letter)
func (g *GoValidator) isExported(name string) bool {
	return len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z'
}

// hasDocComment checks if the previous line has a doc comment
func (g *GoValidator) hasDocComment(lines []string, lineIndex int) bool {
	if lineIndex == 0 {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(lines[lineIndex-1]), "//")
}
