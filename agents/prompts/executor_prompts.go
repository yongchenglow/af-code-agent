package prompts

import (
	"fmt"
	"strings"
)

// SecurityExecutorPrompts contains security executor prompt templates
type SecurityExecutorPrompts struct {
	SystemPrompt string
	TaskPrompt   func(issue *SecurityIssue) string
}

// SecurityIssue represents a security vulnerability
type SecurityIssue struct {
	ID           string
	FilePath     string
	Line         int
	Type         string
	Severity     string
	Title        string
	Description  string
	CWE          string
	OWASP        string
	Remediation  string
}

// NewSecurityExecutorPrompts creates security executor prompts
func NewSecurityExecutorPrompts() *SecurityExecutorPrompts {
	return &SecurityExecutorPrompts{
		SystemPrompt: systemSecurityExecutorPrompt,
		TaskPrompt:   buildSecurityExecutorTaskPrompt,
	}
}

const systemSecurityExecutorPrompt = `You are a Security Engineer specializing in application security and OWASP Top 10 vulnerabilities. Your task is to fix security vulnerabilities in code.

## Your Expertise
- SQL Injection and NoSQL Injection
- Cross-Site Scripting (XSS)
- Authentication and Session Management flaws
- Insecure Direct Object References (IDOR)
- Security Misconfigurations
- Sensitive Data Exposure
- Missing Access Controls
- Cross-Site Request Forgery (CSRF)
- Components with Known Vulnerabilities
- Insufficient Logging and Monitoring
- Hardcoded Secrets, API Keys, Passwords
- Cryptographic Failures

## Fix Requirements
1. **Eliminate the vulnerability completely** - No partial fixes
2. **Don't introduce new security issues** - Verify the fix is secure
3. **Follow secure coding best practices** - Use parameterized queries, proper encoding, etc.
4. **Include input validation** - Validate all user inputs
5. **Minimal changes** - Only change what's necessary to fix the vulnerability

## Security-Specific Validation
After generating fix, verify:
- [ ] No user input reaches sensitive operations without validation
- [ ] Secrets are not hardcoded
- [ ] Authentication/authorization is enforced
- [ ] Error messages don't leak sensitive information

Output ONLY the fixed code.`

func buildSecurityExecutorTaskPrompt(issue *SecurityIssue) string {
	var b strings.Builder

	b.WriteString("## Security Fix Task\n\n")
	b.WriteString(fmt.Sprintf("**Vulnerability Type**: %s\n", issue.Type))
	b.WriteString(fmt.Sprintf("**Severity**: %s\n", issue.Severity))
	b.WriteString(fmt.Sprintf("**CWE**: %s\n", issue.CWE))
	b.WriteString(fmt.Sprintf("**OWASP**: %s\n", issue.OWASP))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("**File**: %s:%d\n", issue.FilePath, issue.Line))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("**Issue**: %s\n", issue.Description))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("**Remediation**: %s\n", issue.Remediation))
	b.WriteString("\n")
	b.WriteString("## Your Task\n\n")
	b.WriteString("Fix this security vulnerability. Your fix MUST eliminate the vulnerability completely.\n\n")
	b.WriteString("Output ONLY the fixed code.\n")

	return b.String()
}

// BugFixExecutorPrompts contains bug fix executor prompt templates
type BugFixExecutorPrompts struct {
	SystemPrompt string
	TaskPrompt   func(issue *BugIssue) string
}

// BugIssue represents a logic bug
type BugIssue struct {
	ID               string
	FilePath         string
	Line             int
	Type             string
	Severity         string
	Title            string
	Description      string
	WhyItFails       string
	ExpectedBehavior string
}

// NewBugFixExecutorPrompts creates bug fix executor prompts
func NewBugFixExecutorPrompts() *BugFixExecutorPrompts {
	return &BugFixExecutorPrompts{
		SystemPrompt: systemBugFixExecutorPrompt,
		TaskPrompt:   buildBugFixExecutorTaskPrompt,
	}
}

const systemBugFixExecutorPrompt = `You are a senior Developer fixing logic bugs in production code. Your fixes are minimal, correct, and don't introduce new bugs.

## Principles
1. **Understand the intent** - What was the code supposed to do?
2. **Find the root cause** - Why does it fail?
3. **Minimal fix** - Change only what's necessary
4. **Preserve functionality** - Don't break working code
5. **Handle edge cases** - Consider nil, empty, max values

## Think Before Fixing
1. What is the INTENDED behavior?
2. What is the ACTUAL behavior?
3. What is the ROOT CAUSE?
4. What is the MINIMAL fix?

Output ONLY the fixed code.`

func buildBugFixExecutorTaskPrompt(issue *BugIssue) string {
	var b strings.Builder

	b.WriteString("## Bug Fix Task\n\n")
	b.WriteString(fmt.Sprintf("**Bug Type**: %s\n", issue.Type))
	b.WriteString(fmt.Sprintf("**Severity**: %s\n", issue.Severity))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("**File**: %s:%d\n", issue.FilePath, issue.Line))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("**What's Wrong**: %s\n", issue.Description))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("**Why It Fails**: %s\n", issue.WhyItFails))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("**Expected Behavior**: %s\n", issue.ExpectedBehavior))
	b.WriteString("\n")
	b.WriteString("## Your Task\n\n")
	b.WriteString("Fix this bug. Your fix MUST correct the logic error without introducing new bugs.\n\n")
	b.WriteString("Output ONLY the fixed code.\n")

	return b.String()
}

// StandardsExecutorPrompts contains standards executor prompt templates
type StandardsExecutorPrompts struct {
	SystemPrompt string
	TaskPrompt   func(violation *StandardsViolation) string
}

// StandardsViolation represents a coding standards violation
type StandardsViolation struct {
	ID          string
	FilePath    string
	Line        int
	Rule        string
	Severity    string
	Message     string
	Why         string
	Suggestion  string
	AutoFixable bool
}

// NewStandardsExecutorPrompts creates standards executor prompts
func NewStandardsExecutorPrompts() *StandardsExecutorPrompts {
	return &StandardsExecutorPrompts{
		SystemPrompt: systemStandardsExecutorPrompt,
		TaskPrompt:   buildStandardsExecutorTaskPrompt,
	}
}

const systemStandardsExecutorPrompt = `You are a Code Quality Engineer fixing coding standards violations. Your fixes are minimal and follow project conventions.

## Fix Requirements
1. **Be minimal** - Change ONLY what's needed to fix the violation
2. **Follow conventions** - Match existing code style
3. **Preserve functionality** - Don't change logic, only style
4. **Use proper tools** - Apply standard formatting rules

Output ONLY the fixed code.`

func buildStandardsExecutorTaskPrompt(violation *StandardsViolation) string {
	var b strings.Builder

	b.WriteString("## Standards Fix Task\n\n")
	b.WriteString(fmt.Sprintf("**Rule**: %s\n", violation.Rule))
	b.WriteString(fmt.Sprintf("**Severity**: %s\n", violation.Severity))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("**File**: %s:%d\n", violation.FilePath, violation.Line))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("**Issue**: %s\n", violation.Message))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("**Why It Matters**: %s\n", violation.Why))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("**Suggested Fix**: %s\n", violation.Suggestion))
	b.WriteString("\n")
	b.WriteString("## Your Task\n\n")
	b.WriteString("Fix this standards violation. Be minimal - change ONLY what's needed.\n\n")
	b.WriteString("Output ONLY the fixed code.\n")

	return b.String()
}

// TestExecutorPrompts contains test executor prompt templates
type TestExecutorPrompts struct {
	SystemPrompt string
	TaskPrompt   func(gap *TestGap, fixCode string) string
}

// TestGap represents a missing test
type TestGap struct {
	ID          string
	Description string
	TestFile    string
	Framework   string
	TestCount   int
	TestCases   []string
}

// NewTestExecutorPrompts creates test executor prompts
func NewTestExecutorPrompts() *TestExecutorPrompts {
	return &TestExecutorPrompts{
		SystemPrompt: systemTestExecutorPrompt,
		TaskPrompt:   buildTestExecutorTaskPrompt,
	}
}

const systemTestExecutorPrompt = `You are a Test Engineer writing comprehensive tests for fixed code. Your tests are clear, thorough, and follow best practices.

## Test Guidelines
1. **One test per scenario** - Don't combine multiple tests
2. **Descriptive names** - TestFunction_Scenario_ExpectedResult
3. **Table-driven where applicable** - Go best practice
4. **Edge cases** - Empty input, max values, nil cases
5. **Verify the fix** - Test the specific issue that was fixed

## Test Structure
Use the Arrange-Act-Assert pattern:
- Arrange: Set up test data and mocks
- Act: Call the function under test
- Assert: Verify the expected outcome

Output the complete test file.`

func buildTestExecutorTaskPrompt(gap *TestGap, fixCode string) string {
	var b strings.Builder

	b.WriteString("## Test Writing Task\n\n")
	b.WriteString(fmt.Sprintf("**What Was Fixed**: %s\n", gap.Description))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("**Test File**: %s\n", gap.TestFile))
	b.WriteString(fmt.Sprintf("**Test Framework**: %s\n", gap.Framework))
	b.WriteString(fmt.Sprintf("**Tests to Write**: %d\n", gap.TestCount))
	b.WriteString("\n")

	if len(gap.TestCases) > 0 {
		b.WriteString("**Test Cases**:\n")
		for i, tc := range gap.TestCases {
			b.WriteString(fmt.Sprintf("%d. %s\n", i+1, tc))
		}
		b.WriteString("\n")
	}

	if fixCode != "" {
		b.WriteString("**Fixed Code**:\n")
		b.WriteString("```go\n")
		b.WriteString(fixCode)
		b.WriteString("\n```\n\n")
	}

	b.WriteString("## Your Task\n\n")
	b.WriteString("Write tests that verify the fix works correctly. Include edge cases and error conditions.\n\n")
	b.WriteString("Output the complete test file.\n")

	return b.String()
}
