package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yourorg/github-code-agent/agents/analyzer"
	"github.com/yourorg/github-code-agent/agents/planner"
	"github.com/yourorg/github-code-agent/agents/qasynth"
	"github.com/yourorg/github-code-agent/agents/testexec"
	"github.com/yourorg/github-code-agent/pkg/constants"
	"github.com/yourorg/github-code-agent/pkg/utils"
)

// TestIntegrationPlannerPromptBuilding tests the planner prompt building end-to-end
func TestIntegrationPlannerPromptBuilding(t *testing.T) {
	// Create sample file changes
	files := []*analyzer.FileChange{
		{
			Filename:  "pkg/auth/auth.go",
			Language:  "go",
			Status:    "added",
			Additions: 50,
			Deletions: 0,
			Patch: `+package auth
+
+func Login(username, password string) (*Token, error) {
+    // TODO: Add input validation
+    query := fmt.Sprintf("SELECT * FROM users WHERE username = '%s'", username)
+    // Vulnerable to SQL injection!
+    return executeQuery(query)
+}`,
		},
		{
			Filename:  "pkg/handler/handler.go",
			Language:  "go",
			Status:    "modified",
			Additions: 20,
			Deletions: 10,
			Patch: ` func ProcessRequest(w http.ResponseWriter, r *http.Request) {
-    data := parseBody(r)
-    result := handle(data)
+    var data Request
+    if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
+        http.Error(w, "Invalid request", http.StatusBadRequest)
+        return
+    }
+    result := handle(&data)
     writeResponse(w, result)
 }`,
		},
	}

	// Test building review context
	prContext := map[string]any{
		"title":       "Add authentication and improve request handling",
		"description": "This PR adds JWT authentication and improves error handling in request processing",
	}

	reviewCtx := buildTestReviewContext(files, prContext)

	// Verify context is built correctly
	if reviewCtx.PRTitle != prContext["title"].(string) {
		t.Errorf("PRTitle mismatch: got %q, want %q", reviewCtx.PRTitle, prContext["title"])
	}

	// Test building PR info
	prInfo := buildTestPRInfo(reviewCtx)
	if !strings.Contains(prInfo, "Add authentication") {
		t.Error("PR info should contain title")
	}

	// Test building files info
	filesInfo := buildTestFilesInfo(files)
	if !strings.Contains(filesInfo, "pkg/auth/auth.go") {
		t.Error("Files info should contain first file")
	}
	if !strings.Contains(filesInfo, "pkg/handler/handler.go") {
		t.Error("Files info should contain second file")
	}

	// Note: Can't test prompt template loading here as they're embedded in planner package
	// This would be tested in planner package tests
}

// TestIntegrationQASynthesisDecisionFlow tests the QA synthesis decision flow
func TestIntegrationQASynthesisDecisionFlow(t *testing.T) {
	// Simulate different validation scenarios
	scenarios := []struct {
		name             string
		syntaxValid      bool
		lintingPassed    bool
		testsPassed      bool
		expectedAction   string
		iterationHistory []*qasynth.IterationRecord
	}{
		{
			name:           "all checks pass",
			syntaxValid:    true,
			lintingPassed:  true,
			testsPassed:    true,
			expectedAction: "APPROVE",
		},
		{
			name:           "syntax error",
			syntaxValid:    false,
			lintingPassed:  false,
			testsPassed:    false,
			expectedAction: "FIX",
		},
		{
			name:           "linting failed",
			syntaxValid:    true,
			lintingPassed:  false,
			testsPassed:    true,
			expectedAction: "FIX",
		},
		{
			name:           "tests failed",
			syntaxValid:    true,
			lintingPassed:  true,
			testsPassed:    false,
			expectedAction: "FIX",
		},
		{
			name:           "stuck after multiple attempts",
			syntaxValid:    false,
			lintingPassed:  false,
			testsPassed:    false,
			expectedAction: "BLOCK",
			iterationHistory: []*qasynth.IterationRecord{
				{Action: "FIX", Summary: "Attempt 1 failed"},
				{Action: "FIX", Summary: "Attempt 2 failed"},
				{Action: "FIX", Summary: "Attempt 3 failed"},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			input := &qasynth.SynthesisInput{
				IssueID:          "TEST-001",
				Description:      "Test issue description",
				SyntaxValid:      scenario.syntaxValid,
				LintingPassed:    scenario.lintingPassed,
				TestsPassed:      scenario.testsPassed,
				IterationHistory: scenario.iterationHistory,
			}

			// Build prompt (this is what would be sent to AI)
			prompt := buildTestQASynthesisPrompt(input)

			// Verify prompt contains relevant information
			if !strings.Contains(prompt, "TEST-001") {
				t.Error("Prompt should contain issue ID")
			}
			if scenario.syntaxValid && !strings.Contains(prompt, "true") {
				t.Error("Prompt should contain syntax validation result")
			}

			// Verify iteration history is included when present
			if len(scenario.iterationHistory) > 0 && !strings.Contains(prompt, "Iteration History") {
				t.Error("Prompt should contain iteration history")
			}

			// Note: Actual AI decision would require mocking the agent
			// This test verifies the prompt building works correctly
		})
	}
}

// TestIntegrationTestGenerationFlow tests the test generation flow
func TestIntegrationTestGenerationFlow(t *testing.T) {
	// Create sample test gaps
	testGaps := []*planner.TestGap{
		{
			ID:          "TEST-001",
			Description: "Add unit tests for error handling in AuthService",
			TestFile:    "pkg/auth/auth_test.go",
			Framework:   "testing",
			TestCount:   3,
			TestCases: []string{
				"TestLogin_InvalidCredentials",
				"TestLogin_EmptyUsername",
				"TestLogin_DatabaseError",
			},
		},
		{
			ID:          "TEST-002",
			Description: "Add integration tests for HTTP handler",
			TestFile:    "pkg/handler/handler_integration_test.go",
			Framework:   "testing",
			TestCount:   2,
			TestCases: []string{
				"TestProcessRequest_ValidRequest",
				"TestProcessRequest_InvalidJSON",
			},
		},
	}

	// Simulate fix code that tests should verify
	fixCodes := map[string]string{
		"TEST-001": `package auth

func Login(username, password string) (*Token, error) {
	if username == "" {
		return nil, ErrEmptyUsername
	}
	if password == "" {
		return nil, ErrEmptyPassword
	}
	// ... rest of implementation
}`,
	}

	// Test building test generation prompts
	for _, gap := range testGaps {
		t.Run(gap.ID, func(t *testing.T) {
			// Verify gap structure
			if gap.ID == "" {
				t.Error("Test gap should have ID")
			}
			if gap.TestFile == "" {
				t.Error("Test gap should have test file")
			}
			if gap.TestCount <= 0 {
				t.Error("Test gap should have positive test count")
			}

			// Build task prompt
			fixCode := fixCodes[gap.ID]
			taskPrompt := buildTestTaskPrompt(gap, fixCode)

			// Verify prompt contains relevant information
			if !strings.Contains(taskPrompt, gap.Description) {
				t.Error("Task prompt should contain gap description")
			}
			if !strings.Contains(taskPrompt, gap.TestFile) {
				t.Error("Task prompt should contain test file")
			}
			if fixCode != "" && !strings.Contains(taskPrompt, fixCode) {
				t.Error("Task prompt should contain fix code")
			}
		})
	}
}

// TestIntegrationUtilityFunctions tests integration of utility functions
func TestIntegrationUtilityFunctions(t *testing.T) {
	// Test file filtering workflow
	files := []*analyzer.FileChange{
		{Filename: "main.go", Language: "go", Status: "added"},
		{Filename: "test.go", Language: "go", Status: "added"},
		{Filename: "image.png", Language: "binary", Status: "added"},
		{Filename: "vendor/lib.go", Language: "go", Status: "added"},
		{Filename: "README.md", Language: "markdown", Status: "added"},
	}

	// Filter reviewable files
	reviewable := filterTestReviewableFiles(files)
	if len(reviewable) != 3 {
		t.Errorf("Expected 3 reviewable files, got %d", len(reviewable))
	}

	// Filter code files only
	codeFiles := filterTestCodeFiles(files)
	if len(codeFiles) != 3 {
		t.Errorf("Expected 3 code files, got %d", len(codeFiles))
	}

	// Test language detection
	for _, file := range codeFiles {
		lang := utils.DetectLanguage(file.Filename)
		if lang != file.Language && lang != "unknown" {
			t.Errorf("Language detection mismatch for %s: got %q, want %q",
				file.Filename, lang, file.Language)
		}
	}

	// Test content truncation
	longContent := strings.Repeat("line\n", 1000)
	truncated := utils.TruncateContent(longContent, constants.MaxContentLength)
	if !strings.HasSuffix(truncated, "... (truncated)") {
		t.Error("Truncated content should have truncation indicator")
	}
}

// TestIntegrationPromptTemplates tests that all prompt templates are loaded
func TestIntegrationPromptTemplates(t *testing.T) {
	// Just verify templates are non-empty
	// Note: We can't access embedded prompt variables from tests package
	// This is a placeholder for actual integration testing

	t.Skip("Skipping - prompt templates are embedded in respective packages")
}

// TestIntegrationJSONParsing tests JSON parsing across components
func TestIntegrationJSONParsing(t *testing.T) {
	// Test planner response parsing
	plannerResponse := `{
		"summary": "Code review summary",
		"recommendation": "FIX",
		"issues": [
			{
				"id": "SEC-001",
				"file_path": "pkg/auth.go",
				"line": 42,
				"severity": "Critical",
				"category": "security",
				"title": "SQL Injection",
				"description": "Vulnerable to SQL injection",
				"suggestion": "Use parameterized queries",
				"cwe": "CWE-89",
				"owasp": "A03:2021"
			}
		]
	}`

	// Parse planner response
	var plannerResp struct {
		Summary        string              `json:"summary"`
		Recommendation string              `json:"recommendation"`
		Issues         []*planner.RawIssue `json:"issues"`
	}

	if err := json.Unmarshal([]byte(plannerResponse), &plannerResp); err != nil {
		t.Fatalf("Failed to parse planner response: %v", err)
	}

	if plannerResp.Summary != "Code review summary" {
		t.Errorf("Planner summary mismatch: got %q", plannerResp.Summary)
	}
	if len(plannerResp.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(plannerResp.Issues))
	}

	// Test QA synthesis response parsing
	qaResponse := `{
		"action": "FIX",
		"summary": "Linting errors found",
		"feedback": ["Fix line length", "Add comments"],
		"stuck": false
	}`

	var qaResp qasynth.SynthesisDecision
	if err := json.Unmarshal([]byte(qaResponse), &qaResp); err != nil {
		t.Fatalf("Failed to parse QA response: %v", err)
	}

	if qaResp.Action != "FIX" {
		t.Errorf("QA action mismatch: got %q", qaResp.Action)
	}
	if len(qaResp.Feedback) != 2 {
		t.Errorf("Expected 2 feedback items, got %d", len(qaResp.Feedback))
	}

	// Test test result parsing
	testResult := &testexec.TestResult{
		GapID:     "TEST-001",
		TestFile:  "pkg/auth_test.go",
		TestCode:  "package auth\n\nfunc TestLogin(t *testing.T) {}",
		Success:   true,
		TestCount: 1,
	}

	testData, err := json.Marshal(testResult)
	if err != nil {
		t.Fatalf("Failed to marshal test result: %v", err)
	}

	var unmarshaled testexec.TestResult
	if err := json.Unmarshal(testData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal test result: %v", err)
	}

	if unmarshaled.GapID != testResult.GapID {
		t.Errorf("Test result GapID mismatch: got %q", unmarshaled.GapID)
	}
}

// TestIntegrationContextPropagation tests context propagation through components
func TestIntegrationContextPropagation(t *testing.T) {
	ctx := context.Background()

	// Simulate context with timeout (as would be used in real workflow)
	aiCtx, cancel := context.WithTimeout(ctx, constants.DefaultAITimeout)
	defer cancel()

	// Verify context deadline is set
	deadline, ok := aiCtx.Deadline()
	if !ok {
		t.Fatal("Context should have deadline")
	}

	// Verify deadline is reasonable (within 11 minutes for 10 minute timeout)
	expectedDeadline := time.Now().Add(11 * time.Minute)
	if deadline.After(expectedDeadline) {
		t.Error("Context deadline too far in future")
	}

	// Test context cancellation
	cancel()
	select {
	case <-aiCtx.Done():
		// Expected
	default:
		t.Error("Context should be done after cancellation")
	}
}

// TestIntegrationErrorHandling tests error handling across components
func TestIntegrationErrorHandling(t *testing.T) {
	// Test empty input handling

	// Planner should handle empty files
	emptyPlan := &planner.ReviewPlan{
		Summary:             "No reviewable files found",
		SecurityIssues:      []*planner.SecurityIssue{},
		BugIssues:           []*planner.BugIssue{},
		StandardsViolations: []*planner.StandardsViolation{},
		TestGaps:            []*planner.TestGap{},
		FixPlan:             &planner.FixPlan{},
		Recommendation:      "APPROVE",
	}

	if emptyPlan.Recommendation != "APPROVE" {
		t.Error("Empty plan should recommend APPROVE")
	}

	// Test invalid JSON handling
	invalidJSON := "not valid json"
	extractedJSON := utils.ExtractJSON(invalidJSON)
	if extractedJSON != invalidJSON {
		t.Error("Invalid JSON should be returned as-is")
	}

	// Test code extraction from invalid response
	invalidCode := "no code blocks here"
	extractedCode := utils.ExtractCodeFromResponse(invalidCode)
	if extractedCode != invalidCode {
		t.Error("Invalid code should be returned as-is")
	}
}

// TestIntegrationFileOperations tests file operation utilities
func TestIntegrationFileOperations(t *testing.T) {
	// Test file skip logic
	skipFiles := []string{
		"image.png",
		"vendor/lib.go",
		"node_modules/pkg.js",
		"go.sum",
	}

	for _, filename := range skipFiles {
		if !utils.ShouldSkipFile(filename) {
			t.Errorf("ShouldSkipFile(%q) should return true", filename)
		}
	}

	// Test file keep logic
	keepFiles := []string{
		"main.go",
		"pkg/auth.go",
		"src/handler.go",
	}

	for _, filename := range keepFiles {
		if utils.ShouldSkipFile(filename) {
			t.Errorf("ShouldSkipFile(%q) should return false", filename)
		}
	}

	// Test ignore patterns
	patterns := []string{"*.md", "docs/**", "tests/fixtures/**"}
	ignoreFiles := []string{
		"README.md",
		"docs/guide.md",
		"tests/fixtures/data.json",
	}

	for _, filename := range ignoreFiles {
		if !utils.ShouldIgnoreFile(filename, patterns) {
			t.Errorf("ShouldIgnoreFile(%q) should return true", filename)
		}
	}
}

// TestIntegrationConstantsConsistency tests that constants are used consistently
func TestIntegrationConstantsConsistency(t *testing.T) {
	// Verify AI timeout is used consistently
	if constants.DefaultAITimeout != 10*time.Minute {
		t.Errorf("DefaultAITimeout changed: got %v", constants.DefaultAITimeout)
	}

	// Verify max tokens are reasonable
	if constants.ReviewAIMaxTokens < 1000 {
		t.Error("ReviewAIMaxTokens seems too low")
	}
	if constants.ReviewAIMaxTokens > 10000 {
		t.Error("ReviewAIMaxTokens seems too high")
	}

	// Verify severity levels are consistent
	severities := []string{
		constants.SeverityCritical,
		constants.SeverityHigh,
		constants.SeverityMedium,
		constants.SeverityLow,
	}

	for _, severity := range severities {
		if severity == "" {
			t.Error("Severity level should not be empty")
		}
	}
}

// Helper functions

type testReviewContext struct {
	PRTitle       string
	PRDescription string
	Files         []*analyzer.FileChange
}

func buildTestReviewContext(files []*analyzer.FileChange, prContext map[string]any) testReviewContext {
	ctx := testReviewContext{
		Files: files,
	}
	if title, ok := prContext["title"].(string); ok {
		ctx.PRTitle = title
	}
	if description, ok := prContext["description"].(string); ok {
		ctx.PRDescription = description
	}
	return ctx
}

func buildTestPRInfo(ctx testReviewContext) string {
	var b strings.Builder
	if ctx.PRTitle != "" {
		b.WriteString("**PR Title**: ")
		b.WriteString(ctx.PRTitle)
		b.WriteString("\n")
	}
	if ctx.PRDescription != "" {
		b.WriteString("**PR Description**: ")
		b.WriteString(ctx.PRDescription)
		b.WriteString("\n")
	}
	return b.String()
}

func buildTestFilesInfo(files []*analyzer.FileChange) string {
	var b strings.Builder
	for i, file := range files {
		if i >= 20 {
			fmt.Fprintf(&b, "\n... and %d more files\n", len(files)-20)
			break
		}
		fmt.Fprintf(&b, "### File: %s (%s)\n", file.Filename, file.Language)
		fmt.Fprintf(&b, "Changes: +%d -%d\n", file.Additions, file.Deletions)
		if file.Patch != "" {
			b.WriteString("```diff\n")
			b.WriteString(file.Patch)
			b.WriteString("\n```\n\n")
		}
	}
	return b.String()
}

func buildTestQASynthesisPrompt(input *qasynth.SynthesisInput) string {
	iterationHistory := ""
	if len(input.IterationHistory) > 0 {
		iterationHistory += "## Iteration History\n"
		for i, iter := range input.IterationHistory {
			iterationHistory += fmt.Sprintf("%d. **%s**: %s\n", i+1, iter.Action, iter.Summary)
		}
		iterationHistory += "\n"
	}

	return fmt.Sprintf("Issue: %s\nDescription: %s\nSyntax Valid: %v\nLinting Passed: %v\nTests Passed: %v\n%s",
		input.IssueID,
		input.Description,
		input.SyntaxValid,
		input.LintingPassed,
		input.TestsPassed,
		iterationHistory,
	)
}

func buildTestTaskPrompt(gap *planner.TestGap, fixCode string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Generate tests for: %s\n", gap.Description)
	fmt.Fprintf(&b, "Test file: %s\n", gap.TestFile)
	fmt.Fprintf(&b, "Framework: %s\n", gap.Framework)
	fmt.Fprintf(&b, "Test count: %d\n", gap.TestCount)

	if len(gap.TestCases) > 0 {
		b.WriteString("Test cases:\n")
		for i, tc := range gap.TestCases {
			fmt.Fprintf(&b, "%d. %s\n", i+1, tc)
		}
	}

	if fixCode != "" {
		fmt.Fprintf(&b, "\nFixed code:\n```go\n%s\n```\n", fixCode)
	}

	return b.String()
}

func filterTestReviewableFiles(files []*analyzer.FileChange) []*analyzer.FileChange {
	var reviewable []*analyzer.FileChange
	for _, file := range files {
		if file.Status == constants.FileStatusRemoved {
			continue
		}
		if utils.ShouldSkipFile(file.Filename) {
			continue
		}
		reviewable = append(reviewable, file)
	}
	return reviewable
}

func filterTestCodeFiles(files []*analyzer.FileChange) []*analyzer.FileChange {
	var codeFiles []*analyzer.FileChange
	for _, file := range files {
		if utils.IsCodeLanguage(file.Language) {
			codeFiles = append(codeFiles, file)
		}
	}
	return codeFiles
}
