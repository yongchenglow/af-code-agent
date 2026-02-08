package standards

import (
	"fmt"
	"regexp"
	"strings"
)

// PythonValidator handles Python-specific validation
type PythonValidator struct{}

// NewPythonValidator creates a new PythonValidator
func NewPythonValidator() *PythonValidator {
	return &PythonValidator{}
}

// CheckFunctionLength checks Python function length
func (p *PythonValidator) CheckFunctionLength(ctx RuleContext, maxLength int) []*Violation {
	var violations []*Violation
	lines := strings.Split(ctx.Content, "\n")

	funcPattern := regexp.MustCompile(`^\s*def\s+(\w+)`)
	currentFunc := ""
	funcStartLine := 0
	inFunc := false
	baseIndent := 0

	for i, line := range lines {
		if funcPattern.MatchString(line) {
			matches := funcPattern.FindStringSubmatch(line)
			currentFunc = matches[1]
			funcStartLine = i + 1
			baseIndent = p.getIndentLevel(line)
			inFunc = true
			continue
		}

		if inFunc {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}

			indent := p.getIndentLevel(line)
			if indent <= baseIndent && trimmed != "" {
				funcLength := i - funcStartLine
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

// CheckDocumentation checks Python documentation requirements
func (p *PythonValidator) CheckDocumentation(ctx RuleContext, lines []string) []*Violation {
	var violations []*Violation

	funcPattern := regexp.MustCompile(`^\s*def\s+(\w+)`)
	classPattern := regexp.MustCompile(`^\s*class\s+(\w+)`)

	for i, line := range lines {
		if funcPattern.MatchString(line) || classPattern.MatchString(line) {
			if !p.hasDocstring(lines, i) {
				matches := funcPattern.FindStringSubmatch(line)
				if len(matches) == 0 {
					matches = classPattern.FindStringSubmatch(line)
				}
				name := matches[1]

				violations = append(violations, &Violation{
					FilePath:    ctx.FilePath,
					Line:        i + 1,
					Rule:        "missing-docs",
					Message:     fmt.Sprintf("Function/class '%s' is missing docstring", name),
					Suggestion:  "Add a docstring describing the purpose and parameters",
					AutoFixable: false,
				})
			}
		}
	}

	return violations
}

// getIndentLevel returns the indentation level of a line
func (p *PythonValidator) getIndentLevel(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

// hasDocstring checks if the next non-empty line contains a docstring
func (p *PythonValidator) hasDocstring(lines []string, lineIndex int) bool {
	for j := lineIndex + 1; j < len(lines) && j < lineIndex+3; j++ {
		nextLine := strings.TrimSpace(lines[j])
		if nextLine == "" {
			continue
		}
		if strings.HasPrefix(nextLine, `"""`) || strings.HasPrefix(nextLine, `'''`) {
			return true
		}
		break
	}
	return false
}
