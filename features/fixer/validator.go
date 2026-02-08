package fixer

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ValidateFix validates a code fix against multiple criteria
func ValidateFix(ctx context.Context, patch *CodePatch, config *ValidationConfig) (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// 1. Syntax validation
	if config.EnableSyntaxCheck {
		if err := validateSyntax(patch); err != nil {
			result.IsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Syntax error: %v", err))
		}
	}

	// 2. Auto-format if enabled
	if config.AutoFormat {
		formatted, err := autoFormat(patch)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Auto-format warning: %v", err))
		} else if formatted != "" {
			patch.FixedCode = formatted
		}
	}

	// 3. Formatting check
	if config.EnableFormatting {
		if formatErrors := checkFormatting(patch); len(formatErrors) > 0 {
			result.IsValid = false
			result.Errors = append(result.Errors, formatErrors...)
		}
	}

	// 4. Linting
	if config.EnableLinting {
		if lintErrors := runLinters(patch, config.TimeoutSeconds); len(lintErrors) > 0 {
			// Linting errors are warnings, not hard failures
			result.Warnings = append(result.Warnings, lintErrors...)
		}
	}

	// 5. Security scan
	if config.EnableSecurityScan {
		if securityIssues := scanForSecurityIssues(patch); len(securityIssues) > 0 {
			result.IsValid = false
			result.Errors = append(result.Errors, securityIssues...)
		}
	}

	// 6. Verify fix addresses the issue
	if !doesFixAddressIssue(patch) {
		result.IsValid = false
		result.Errors = append(result.Errors, "Fix does not address the original issue")
	}

	return result, nil
}

// validateSyntax checks if the code has valid syntax
func validateSyntax(patch *CodePatch) error {
	switch patch.Language {
	case "go":
		return validateGoSyntax(patch.FixedCode)
	case "python":
		return validatePythonSyntax(patch.FixedCode)
	case "javascript", "typescript":
		return validateJSSyntax(patch.FixedCode)
	default:
		// Skip syntax check for unknown languages
		return nil
	}
}

// validateGoSyntax validates Go code syntax
func validateGoSyntax(code string) error {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "temp.go", code, parser.AllErrors)
	return err
}

// validatePythonSyntax validates Python code syntax using python -m py_compile
func validatePythonSyntax(code string) error {
	// Create temporary file
	tmpfile, err := os.CreateTemp("", "validate-*.py")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		_ = os.Remove(tmpfile.Name())
	}()

	if _, err := tmpfile.Write([]byte(code)); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpfile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Run python syntax check
	cmd := exec.Command("python3", "-m", "py_compile", tmpfile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("syntax error: %s", string(output))
	}

	return nil
}

// validateJSSyntax validates JavaScript/TypeScript syntax using node
func validateJSSyntax(code string) error {
	// Create temporary file
	ext := ".js"
	tmpfile, err := os.CreateTemp("", "validate-*"+ext)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		_ = os.Remove(tmpfile.Name())
	}()

	if _, err := tmpfile.Write([]byte(code)); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpfile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Run node syntax check
	cmd := exec.Command("node", "--check", tmpfile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("syntax error: %s", string(output))
	}

	return nil
}

// runLinters runs language-specific linters on the code
func runLinters(patch *CodePatch, timeoutSeconds int) []string {
	var errors []string
	timeout := time.Duration(timeoutSeconds) * time.Second

	switch patch.Language {
	case "go":
		if lintErrs := runGoLinters(patch, timeout); len(lintErrs) > 0 {
			errors = append(errors, lintErrs...)
		}
	case "python":
		if lintErrs := runPythonLinters(patch, timeout); len(lintErrs) > 0 {
			errors = append(errors, lintErrs...)
		}
	case "javascript", "typescript":
		if lintErrs := runJSLinters(patch, timeout); len(lintErrs) > 0 {
			errors = append(errors, lintErrs...)
		}
	}

	return errors
}

// runGoLinters runs golangci-lint or go vet
func runGoLinters(patch *CodePatch, timeout time.Duration) []string {
	var errors []string

	// Create temp file
	tmpfile, err := os.CreateTemp("", "lint-*.go")
	if err != nil {
		return []string{fmt.Sprintf("failed to create temp file: %v", err)}
	}
	defer func() {
		_ = os.Remove(tmpfile.Name())
	}()

	if _, err := tmpfile.Write([]byte(patch.FixedCode)); err != nil {
		return []string{fmt.Sprintf("failed to write temp file: %v", err)}
	}
	if err := tmpfile.Close(); err != nil {
		return []string{fmt.Sprintf("failed to close temp file: %v", err)}
	}

	// Try go vet first (more commonly available)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "vet", tmpfile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			errors = append(errors, fmt.Sprintf("go vet: %s", string(output)))
		}
	}

	return errors
}

// runPythonLinters runs pylint or flake8
func runPythonLinters(patch *CodePatch, timeout time.Duration) []string {
	var errors []string

	// Create temp file
	tmpfile, err := os.CreateTemp("", "lint-*.py")
	if err != nil {
		return []string{fmt.Sprintf("failed to create temp file: %v", err)}
	}
	defer func() {
		_ = os.Remove(tmpfile.Name())
	}()

	if _, err := tmpfile.Write([]byte(patch.FixedCode)); err != nil {
		return []string{fmt.Sprintf("failed to write temp file: %v", err)}
	}
	if err := tmpfile.Close(); err != nil {
		return []string{fmt.Sprintf("failed to close temp file: %v", err)}
	}

	// Try flake8 (more lenient than pylint)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "flake8", "--select=E,W,F", tmpfile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			errors = append(errors, fmt.Sprintf("flake8: %s", string(output)))
		}
	}

	return errors
}

// runJSLinters runs eslint
func runJSLinters(patch *CodePatch, timeout time.Duration) []string {
	var errors []string

	// Create temp file
	ext := ".js"
	if patch.Language == "typescript" {
		ext = ".ts"
	}
	tmpfile, err := os.CreateTemp("", "lint-*"+ext)
	if err != nil {
		return []string{fmt.Sprintf("failed to create temp file: %v", err)}
	}
	defer func() {
		_ = os.Remove(tmpfile.Name())
	}()

	if _, err := tmpfile.Write([]byte(patch.FixedCode)); err != nil {
		return []string{fmt.Sprintf("failed to write temp file: %v", err)}
	}
	if err := tmpfile.Close(); err != nil {
		return []string{fmt.Sprintf("failed to close temp file: %v", err)}
	}

	// Try eslint
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "eslint", tmpfile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			errors = append(errors, fmt.Sprintf("eslint: %s", string(output)))
		}
	}

	return errors
}

// autoFormat automatically formats code
func autoFormat(patch *CodePatch) (string, error) {
	switch patch.Language {
	case "go":
		return formatGoCode(patch.FixedCode)
	case "python":
		return formatPythonCode(patch.FixedCode)
	case "javascript", "typescript":
		return formatJSCode(patch.FixedCode)
	default:
		return "", nil
	}
}

// formatGoCode formats Go code using gofmt
func formatGoCode(code string) (string, error) {
	tmpfile, err := os.CreateTemp("", "format-*.go")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = os.Remove(tmpfile.Name())
	}()

	if _, err := tmpfile.Write([]byte(code)); err != nil {
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", err
	}

	cmd := exec.Command("gofmt", tmpfile.Name())
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// formatPythonCode formats Python code using black
func formatPythonCode(code string) (string, error) {
	tmpfile, err := os.CreateTemp("", "format-*.py")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = os.Remove(tmpfile.Name())
	}()

	if _, err := tmpfile.Write([]byte(code)); err != nil {
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", err
	}

	cmd := exec.Command("black", "--quiet", tmpfile.Name())
	if err := cmd.Run(); err != nil {
		return "", err
	}

	// Read formatted file
	formatted, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return "", err
	}

	return string(formatted), nil
}

// formatJSCode formats JS/TS code using prettier
func formatJSCode(code string) (string, error) {
	tmpfile, err := os.CreateTemp("", "format-*.js")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = os.Remove(tmpfile.Name())
	}()

	if _, err := tmpfile.Write([]byte(code)); err != nil {
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", err
	}

	cmd := exec.Command("prettier", "--write", tmpfile.Name())
	if err := cmd.Run(); err != nil {
		return "", err
	}

	// Read formatted file
	formatted, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return "", err
	}

	return string(formatted), nil
}

// checkFormatting checks if code follows formatting standards
func checkFormatting(patch *CodePatch) []string {
	var errors []string

	switch patch.Language {
	case "go":
		if !isGoFormatted(patch.FixedCode) {
			errors = append(errors, "Code is not properly formatted (gofmt)")
		}
	case "python":
		if !isPythonFormatted(patch.FixedCode) {
			errors = append(errors, "Code is not properly formatted (black)")
		}
	case "javascript", "typescript":
		if !isJSFormatted(patch.FixedCode) {
			errors = append(errors, "Code is not properly formatted (prettier)")
		}
	}

	return errors
}

// isGoFormatted checks if Go code is properly formatted
func isGoFormatted(code string) bool {
	formatted, err := formatGoCode(code)
	if err != nil {
		return false
	}
	return formatted == code
}

// isPythonFormatted checks if Python code is properly formatted
func isPythonFormatted(code string) bool {
	formatted, err := formatPythonCode(code)
	if err != nil {
		return false
	}
	return strings.TrimSpace(formatted) == strings.TrimSpace(code)
}

// isJSFormatted checks if JS code is properly formatted
func isJSFormatted(code string) bool {
	formatted, err := formatJSCode(code)
	if err != nil {
		return false
	}
	return strings.TrimSpace(formatted) == strings.TrimSpace(code)
}

// scanForSecurityIssues scans the fix for new security vulnerabilities
func scanForSecurityIssues(patch *CodePatch) []string {
	var issues []string

	// Common security patterns to check
	securityPatterns := map[string]*regexp.Regexp{
		"Hardcoded password":     regexp.MustCompile(`(?i)(password|passwd|pwd)\s*=\s*["'][^"']+["']`),
		"Hardcoded API key":      regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*=\s*["'][^"']+["']`),
		"SQL concatenation":      regexp.MustCompile(`(?i)SELECT.*\+.*FROM|INSERT.*\+.*VALUES`),
		"Eval usage":             regexp.MustCompile(`\beval\s*\(`),
		"Unsafe deserialization": regexp.MustCompile(`(?i)pickle\.loads|yaml\.load\(`),
	}

	for name, pattern := range securityPatterns {
		if pattern.MatchString(patch.FixedCode) {
			issues = append(issues, fmt.Sprintf("Potential security issue: %s", name))
		}
	}

	return issues
}

// doesFixAddressIssue verifies the fix actually addresses the original issue
func doesFixAddressIssue(patch *CodePatch) bool {
	// Basic check: ensure the fixed code is different from original
	if patch.FixedCode == patch.OriginalCode {
		return false
	}

	// Ensure the fix is not empty
	if strings.TrimSpace(patch.FixedCode) == "" {
		return false
	}

	// Additional heuristics can be added here
	// For now, we assume if the code changed meaningfully, it addresses the issue
	return true
}

// WritePatchToFile writes a patch to a temporary file for validation
func WritePatchToFile(patch *CodePatch) (string, error) {
	ext := filepath.Ext(patch.FilePath)
	if ext == "" {
		// Determine extension from language
		switch patch.Language {
		case "go":
			ext = ".go"
		case "python":
			ext = ".py"
		case "javascript":
			ext = ".js"
		case "typescript":
			ext = ".ts"
		default:
			ext = ".txt"
		}
	}

	tmpfile, err := os.CreateTemp("", "patch-*"+ext)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := tmpfile.Write([]byte(patch.FixedCode)); err != nil {
		_ = tmpfile.Close()
		_ = os.Remove(tmpfile.Name())
		return "", fmt.Errorf("failed to write patch: %w", err)
	}

	if err := tmpfile.Close(); err != nil {
		_ = os.Remove(tmpfile.Name())
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return tmpfile.Name(), nil
}
