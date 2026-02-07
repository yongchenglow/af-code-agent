package standards

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/yourorg/github-code-agent/features/analyzer"
	"github.com/yourorg/github-code-agent/pkg/config"
)

// Validator handles standards validation
type Validator struct {
	config *config.Config
	rules  []*Rule
}

// NewValidator creates a new standards validator
func NewValidator(cfg *config.Config) *Validator {
	v := &Validator{
		config: cfg,
		rules:  []*Rule{},
	}

	// Register built-in rules
	v.registerBuiltInRules()

	return v
}

// ValidateStandards checks code against configured standards
func (v *Validator) ValidateStandards(ctx context.Context, files []*analyzer.FileChange) (*ValidationReport, error) {
	report := &ValidationReport{
		Violations:       []*Violation{},
		ViolationsByType: make(map[string]int),
		PassedChecks:     []string{},
		FailedChecks:     []string{},
	}

	// Track which checks were executed
	checksExecuted := make(map[string]bool)

	// Validate each file
	for _, file := range files {
		if file.Status == "removed" || file.Content == "" {
			continue
		}

		// Apply each rule
		for _, rule := range v.rules {
			if !rule.Enabled {
				continue
			}

			checksExecuted[rule.Name] = true

			// Create rule context
			ruleCtx := RuleContext{
				FilePath: file.Filename,
				Content:  file.Content,
				Language: file.Language,
				Config:   v.config,
			}

			// Run validator
			violations, err := rule.Validator(ruleCtx)
			if err != nil {
				// Log error but continue with other rules
				continue
			}

			// Add violations to report
			for _, violation := range violations {
				violation.ID = fmt.Sprintf("%s-%s-%d", file.Filename, rule.ID, violation.Line)
				violation.Type = rule.Type
				violation.Severity = rule.Severity

				report.Violations = append(report.Violations, violation)
				report.ViolationsByType[rule.Type]++

				if !contains(report.FailedChecks, rule.Name) {
					report.FailedChecks = append(report.FailedChecks, rule.Name)
				}
			}

			// Track passed checks
			if len(violations) == 0 && !contains(report.PassedChecks, rule.Name) {
				report.PassedChecks = append(report.PassedChecks, rule.Name)
			}
		}
	}

	report.TotalViolations = len(report.Violations)
	report.Summary = fmt.Sprintf("Found %d violations across %d files", report.TotalViolations, len(files))

	return report, nil
}

// registerBuiltInRules registers the built-in validation rules
func (v *Validator) registerBuiltInRules() {
	// Line length check
	v.rules = append(v.rules, &Rule{
		ID:          "line-length",
		Name:        "Maximum Line Length",
		Description: "Checks that lines don't exceed maximum length",
		Type:        ViolationTypeCoding,
		Severity:    SeverityWarning,
		Enabled:     true,
		Validator:   v.validateLineLength,
	})

	// Function length check
	v.rules = append(v.rules, &Rule{
		ID:          "function-length",
		Name:        "Maximum Function Length",
		Description: "Checks that functions don't exceed maximum length",
		Type:        ViolationTypeCoding,
		Severity:    SeverityWarning,
		Enabled:     true,
		Validator:   v.validateFunctionLength,
	})

	// Naming conventions
	v.rules = append(v.rules, &Rule{
		ID:          "naming-convention",
		Name:        "Naming Conventions",
		Description: "Validates naming conventions for functions, classes, constants",
		Type:        ViolationTypeCoding,
		Severity:    SeverityInfo,
		Enabled:     true,
		Validator:   v.validateNamingConventions,
	})

	// Documentation checks
	v.rules = append(v.rules, &Rule{
		ID:          "missing-docs",
		Name:        "Documentation Requirements",
		Description: "Checks for missing documentation",
		Type:        ViolationTypeDocumentation,
		Severity:    SeverityWarning,
		Enabled:     v.config.Standards.Documentation.RequireDocstrings,
		Validator:   v.validateDocumentation,
	})

	// Hardcoded secrets detection
	v.rules = append(v.rules, &Rule{
		ID:          "hardcoded-secrets",
		Name:        "Hardcoded Secrets Detection",
		Description: "Detects hardcoded secrets, API keys, passwords",
		Type:        ViolationTypeSecurity,
		Severity:    SeverityError,
		Enabled:     v.config.Standards.Security.CheckSecrets,
		Validator:   v.validateNoHardcodedSecrets,
	})
}

// validateLineLength checks line length
func (v *Validator) validateLineLength(ctx RuleContext) ([]*Violation, error) {
	maxLength := v.config.Standards.Coding.MaxLineLength
	if maxLength == 0 {
		return nil, nil
	}

	var violations []*Violation
	lines := strings.Split(ctx.Content, "\n")

	for i, line := range lines {
		if len(line) > maxLength {
			violations = append(violations, &Violation{
				FilePath:    ctx.FilePath,
				Line:        i + 1,
				Rule:        "line-length",
				Message:     fmt.Sprintf("Line exceeds maximum length of %d characters (current: %d)", maxLength, len(line)),
				Suggestion:  "Consider breaking this line into multiple lines",
				AutoFixable: true,
			})
		}
	}

	return violations, nil
}

// validateFunctionLength checks function length
func (v *Validator) validateFunctionLength(ctx RuleContext) ([]*Violation, error) {
	maxLength := v.config.Standards.Coding.MaxFunctionLength
	if maxLength == 0 {
		return nil, nil
	}

	var violations []*Violation

	// Simple heuristic for function detection (language-specific parsers would be better)
	switch ctx.Language {
	case "go":
		violations = v.checkGoFunctionLength(ctx, maxLength)
	case "python":
		violations = v.checkPythonFunctionLength(ctx, maxLength)
	case "javascript", "typescript":
		violations = v.checkJSFunctionLength(ctx, maxLength)
	}

	return violations, nil
}

// validateNamingConventions checks naming conventions
func (v *Validator) validateNamingConventions(ctx RuleContext) ([]*Violation, error) {
	var violations []*Violation

	// This is a simplified check - real implementation would use AST parsing
	lines := strings.Split(ctx.Content, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Check for common patterns that violate naming conventions
		// This is language-agnostic but could be enhanced per language

		// Example: Check for single-letter variable names (except i, j, k in loops)
		if strings.Contains(line, "var ") || strings.Contains(line, "let ") {
			// Extract variable name and check
			// This is simplified - real implementation would parse properly
		}

		// Check for overly long names (> 50 chars)
		words := strings.Fields(line)
		for _, word := range words {
			if len(word) > 50 && isIdentifier(word) {
				violations = append(violations, &Violation{
					FilePath:    ctx.FilePath,
					Line:        i + 1,
					Rule:        "naming-convention",
					Message:     fmt.Sprintf("Identifier name is too long: %s", word),
					Suggestion:  "Consider using a shorter, more concise name",
					AutoFixable: false,
				})
			}
		}
	}

	return violations, nil
}

// validateDocumentation checks for missing documentation
func (v *Validator) validateDocumentation(ctx RuleContext) ([]*Violation, error) {
	if !v.config.Standards.Documentation.RequireDocstrings {
		return nil, nil
	}

	var violations []*Violation
	lines := strings.Split(ctx.Content, "\n")

	switch ctx.Language {
	case "go":
		violations = v.checkGoDocumentation(ctx, lines)
	case "python":
		violations = v.checkPythonDocumentation(ctx, lines)
	}

	return violations, nil
}

// validateNoHardcodedSecrets checks for hardcoded secrets
func (v *Validator) validateNoHardcodedSecrets(ctx RuleContext) ([]*Violation, error) {
	if !v.config.Standards.Security.CheckSecrets {
		return nil, nil
	}

	var violations []*Violation
	lines := strings.Split(ctx.Content, "\n")

	// Common secret patterns
	secretPatterns := []struct {
		pattern *regexp.Regexp
		message string
	}{
		{regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[:=]\s*["']([^"']{20,})["']`), "Possible hardcoded API key"},
		{regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*["']([^"']+)["']`), "Possible hardcoded password"},
		{regexp.MustCompile(`(?i)(secret|token)\s*[:=]\s*["']([^"']{20,})["']`), "Possible hardcoded secret/token"},
		{regexp.MustCompile(`(?i)aws[_-]?access[_-]?key[_-]?id\s*[:=]\s*["']([A-Z0-9]{20})["']`), "Possible AWS access key"},
		{regexp.MustCompile(`(?i)aws[_-]?secret[_-]?access[_-]?key\s*[:=]\s*["']([A-Za-z0-9/+=]{40})["']`), "Possible AWS secret key"},
		{regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`), "Possible GitHub personal access token"},
		{regexp.MustCompile(`sk-[a-zA-Z0-9]{48}`), "Possible OpenAI API key"},
	}

	for i, line := range lines {
		// Skip comments (basic check)
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}

		for _, sp := range secretPatterns {
			if sp.pattern.MatchString(line) {
				violations = append(violations, &Violation{
					FilePath:    ctx.FilePath,
					Line:        i + 1,
					Rule:        "hardcoded-secrets",
					Message:     sp.message + " detected",
					Suggestion:  "Use environment variables or a secrets management system",
					AutoFixable: false,
				})
			}
		}
	}

	return violations, nil
}

// Language-specific helper functions

func (v *Validator) checkGoFunctionLength(ctx RuleContext, maxLength int) []*Violation {
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

func (v *Validator) checkPythonFunctionLength(ctx RuleContext, maxLength int) []*Violation {
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
			baseIndent = len(line) - len(strings.TrimLeft(line, " \t"))
			inFunc = true
			continue
		}

		if inFunc {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}

			indent := len(line) - len(strings.TrimLeft(line, " \t"))
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

func (v *Validator) checkJSFunctionLength(ctx RuleContext, maxLength int) []*Violation {
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

func (v *Validator) checkGoDocumentation(ctx RuleContext, lines []string) []*Violation {
	var violations []*Violation

	funcPattern := regexp.MustCompile(`^func\s+(\w+)`)
	typePattern := regexp.MustCompile(`^type\s+(\w+)`)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for exported functions and types (start with capital letter)
		if funcPattern.MatchString(trimmed) {
			matches := funcPattern.FindStringSubmatch(trimmed)
			funcName := matches[1]

			// If exported (starts with capital), check for doc comment
			if len(funcName) > 0 && funcName[0] >= 'A' && funcName[0] <= 'Z' {
				// Check previous line for comment
				if i == 0 || !strings.HasPrefix(strings.TrimSpace(lines[i-1]), "//") {
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

		if typePattern.MatchString(trimmed) {
			matches := typePattern.FindStringSubmatch(trimmed)
			typeName := matches[1]

			if len(typeName) > 0 && typeName[0] >= 'A' && typeName[0] <= 'Z' {
				if i == 0 || !strings.HasPrefix(strings.TrimSpace(lines[i-1]), "//") {
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

func (v *Validator) checkPythonDocumentation(ctx RuleContext, lines []string) []*Violation {
	var violations []*Violation

	funcPattern := regexp.MustCompile(`^\s*def\s+(\w+)`)
	classPattern := regexp.MustCompile(`^\s*class\s+(\w+)`)

	for i, line := range lines {
		if funcPattern.MatchString(line) || classPattern.MatchString(line) {
			// Check next non-empty line for docstring
			hasDocstring := false
			for j := i + 1; j < len(lines) && j < i+3; j++ {
				nextLine := strings.TrimSpace(lines[j])
				if nextLine == "" {
					continue
				}
				if strings.HasPrefix(nextLine, `"""`) || strings.HasPrefix(nextLine, `'''`) {
					hasDocstring = true
					break
				}
				break
			}

			if !hasDocstring {
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

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func isIdentifier(word string) bool {
	if len(word) == 0 {
		return false
	}
	// Simple check - real implementation would be more robust
	for _, ch := range word {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
			return false
		}
	}
	return true
}
