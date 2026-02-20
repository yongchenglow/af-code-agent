package planner

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/yourorg/github-code-agent/agents/analyzer"
)

func TestReviewPlanStruct(t *testing.T) {
	plan := &ReviewPlan{
		Summary:             "Test summary",
		SecurityIssues:      []*SecurityIssue{},
		BugIssues:           []*BugIssue{},
		StandardsViolations: []*StandardsViolation{},
		TestGaps:            []*TestGap{},
		FixPlan:             &FixPlan{},
		Recommendation:      "APPROVE",
	}

	if plan.Summary != "Test summary" {
		t.Errorf("expected Summary to be 'Test summary', got %q", plan.Summary)
	}
	if plan.Recommendation != "APPROVE" {
		t.Errorf("expected Recommendation to be 'APPROVE', got %q", plan.Recommendation)
	}
	if len(plan.SecurityIssues) != 0 {
		t.Errorf("expected SecurityIssues to be empty, got %d items", len(plan.SecurityIssues))
	}
}

func TestSecurityIssueStruct(t *testing.T) {
	issue := &SecurityIssue{
		ID:          "SEC-001",
		FilePath:    "pkg/auth/auth.go",
		Line:        42,
		Type:        "security",
		Severity:    "Critical",
		Title:       "SQL Injection Vulnerability",
		Description: "User input is directly concatenated into SQL query",
		CWE:         "CWE-89",
		OWASP:       "A03:2021-Injection",
		Remediation: "Use parameterized queries",
	}

	if issue.ID != "SEC-001" {
		t.Errorf("expected ID to be 'SEC-001', got %q", issue.ID)
	}
	if issue.Line != 42 {
		t.Errorf("expected Line to be 42, got %d", issue.Line)
	}
	if issue.Severity != "Critical" {
		t.Errorf("expected Severity to be 'Critical', got %q", issue.Severity)
	}
}

func TestBugIssueStruct(t *testing.T) {
	issue := &BugIssue{
		ID:               "BUG-001",
		FilePath:         "pkg/handler/handler.go",
		Line:             100,
		Type:             "bug",
		Severity:         "High",
		Title:            "Nil Pointer Dereference",
		Description:      "Variable may be nil when accessed",
		WhyItFails:       "No nil check before dereferencing",
		ExpectedBehavior: "Should check for nil before use",
	}

	if issue.ID != "BUG-001" {
		t.Errorf("expected ID to be 'BUG-001', got %q", issue.ID)
	}
	if issue.WhyItFails != "No nil check before dereferencing" {
		t.Errorf("expected WhyItFails to have correct value, got %q", issue.WhyItFails)
	}
}

func TestStandardsViolationStruct(t *testing.T) {
	violation := &StandardsViolation{
		ID:          "STD-001",
		FilePath:    "pkg/service/service.go",
		Line:        25,
		Rule:        "style",
		Severity:    "Low",
		Message:     "Function name should use CamelCase",
		Why:         "Go naming conventions require CamelCase",
		Suggestion:  "Rename function to use CamelCase",
		AutoFixable: true,
	}

	if violation.Rule != "style" {
		t.Errorf("expected Rule to be 'style', got %q", violation.Rule)
	}
	if !violation.AutoFixable {
		t.Error("expected AutoFixable to be true")
	}
}

func TestTestGapStruct(t *testing.T) {
	gap := &TestGap{
		ID:          "TEST-001",
		Description: "Missing unit tests for error handling",
		TestFile:    "pkg/service/service_test.go",
		Framework:   "testing",
		TestCount:   3,
		TestCases:   []string{"TestErrorCase1", "TestErrorCase2", "TestEdgeCase"},
	}

	if gap.TestCount != 3 {
		t.Errorf("expected TestCount to be 3, got %d", gap.TestCount)
	}
	if len(gap.TestCases) != 3 {
		t.Errorf("expected 3 test cases, got %d", len(gap.TestCases))
	}
}

func TestFixTaskStruct(t *testing.T) {
	task := &FixTask{
		ID:          "FIX-001",
		Type:        "security",
		File:        "pkg/auth/auth.go",
		Line:        42,
		Description: "Fix SQL injection",
		Priority:    "critical",
		DependsOn:   []string{},
		EstTokens:   1000,
	}

	if task.Type != "security" {
		t.Errorf("expected Type to be 'security', got %q", task.Type)
	}
	if task.Priority != "critical" {
		t.Errorf("expected Priority to be 'critical', got %q", task.Priority)
	}
}

func TestFixPlanStruct(t *testing.T) {
	plan := &FixPlan{
		ParallelGroups: [][]*FixTask{
			{
				{ID: "FIX-001", Type: "security"},
				{ID: "FIX-002", Type: "security"},
			},
		},
		SequentialTasks: []*FixTask{
			{ID: "FIX-003", Type: "test"},
		},
	}

	if len(plan.ParallelGroups) != 1 {
		t.Errorf("expected 1 parallel group, got %d", len(plan.ParallelGroups))
	}
	if len(plan.ParallelGroups[0]) != 2 {
		t.Errorf("expected 2 tasks in first group, got %d", len(plan.ParallelGroups[0]))
	}
	if len(plan.SequentialTasks) != 1 {
		t.Errorf("expected 1 sequential task, got %d", len(plan.SequentialTasks))
	}
}

func TestNewPlanner(t *testing.T) {
	// Create a nil agent for testing (will panic if used, but constructor should work)
	var nilAgent interface{} = (*analyzer.FileChange)(nil)
	_ = nilAgent

	// Test that constructor returns non-nil planner
	// Note: We can't fully test without a real agent, but we can test the constructor
	// This is a placeholder - in real tests you'd mock the agent
	t.Skip("Skipping - requires agent mock")
}

func TestFilterReviewableFiles(t *testing.T) {
	files := []*analyzer.FileChange{
		{Filename: "pkg/main.go", Language: "go", Status: "added", Additions: 10, Deletions: 0},
		{Filename: "docs/README.md", Language: "markdown", Status: "modified", Additions: 5, Deletions: 2},
		{Filename: "vendor/lib.go", Language: "go", Status: "added", Additions: 100, Deletions: 0},
		{Filename: "node_modules/pkg.js", Language: "javascript", Status: "added", Additions: 50, Deletions: 0},
		{Filename: "pkg/auth.go", Language: "go", Status: "removed", Additions: 0, Deletions: 100},
		{Filename: "image.png", Language: "binary", Status: "added", Additions: 0, Deletions: 0},
		{Filename: "pkg/service.go", Language: "go", Status: "modified", Additions: 20, Deletions: 5},
	}

	reviewable := filterReviewableFiles(files)

	// Should skip: vendor/, node_modules/, removed files, binary files
	// Should include: pkg/main.go, docs/README.md, pkg/service.go
	if len(reviewable) != 3 {
		t.Errorf("expected 3 reviewable files, got %d", len(reviewable))
	}

	// Verify specific files are included
	includedFiles := make(map[string]bool)
	for _, f := range reviewable {
		includedFiles[f.Filename] = true
	}

	if !includedFiles["pkg/main.go"] {
		t.Error("expected pkg/main.go to be reviewable")
	}
	if !includedFiles["pkg/service.go"] {
		t.Error("expected pkg/service.go to be reviewable")
	}
	if !includedFiles["docs/README.md"] {
		t.Error("expected docs/README.md to be reviewable")
	}

	// Verify specific files are excluded
	if includedFiles["vendor/lib.go"] {
		t.Error("expected vendor/lib.go to be filtered out")
	}
	if includedFiles["node_modules/pkg.js"] {
		t.Error("expected node_modules/pkg.js to be filtered out")
	}
	if includedFiles["pkg/auth.go"] {
		t.Error("expected removed file pkg/auth.go to be filtered out")
	}
	if includedFiles["image.png"] {
		t.Error("expected image.png to be filtered out")
	}
}

func TestBuildReviewContext(t *testing.T) {
	files := []*analyzer.FileChange{
		{Filename: "pkg/main.go", Language: "go"},
	}

	prContext := map[string]any{
		"title":       "Fix bug in authentication",
		"description": "This PR fixes a critical bug",
	}

	ctx := buildReviewContext(files, prContext)

	if ctx.PRTitle != "Fix bug in authentication" {
		t.Errorf("expected PRTitle to be 'Fix bug in authentication', got %q", ctx.PRTitle)
	}
	if ctx.PRDescription != "This PR fixes a critical bug" {
		t.Errorf("expected PRDescription to be 'This PR fixes a critical bug', got %q", ctx.PRDescription)
	}
	if len(ctx.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(ctx.Files))
	}
}

func TestBuildReviewContextMissingFields(t *testing.T) {
	files := []*analyzer.FileChange{
		{Filename: "pkg/main.go", Language: "go"},
	}

	// Test with missing fields
	prContext := map[string]any{}

	ctx := buildReviewContext(files, prContext)

	if ctx.PRTitle != "" {
		t.Errorf("expected empty PRTitle, got %q", ctx.PRTitle)
	}
	if ctx.PRDescription != "" {
		t.Errorf("expected empty PRDescription, got %q", ctx.PRDescription)
	}
}

func TestBuildPRInfo(t *testing.T) {
	ctx := ReviewContext{
		PRTitle:       "Test PR",
		PRDescription: "Test description",
	}

	info := buildPRInfo(ctx)

	if !strings.Contains(info, "**PR Title**: Test PR") {
		t.Error("expected PR title in output")
	}
	if !strings.Contains(info, "**PR Description**: Test description") {
		t.Error("expected PR description in output")
	}
}

func TestBuildPRInfoEmpty(t *testing.T) {
	ctx := ReviewContext{
		PRTitle:       "",
		PRDescription: "",
	}

	info := buildPRInfo(ctx)

	if info != "" {
		t.Errorf("expected empty output for empty context, got %q", info)
	}
}

func TestBuildFilesInfo(t *testing.T) {
	files := []*analyzer.FileChange{
		{Filename: "pkg/main.go", Language: "go", Additions: 10, Deletions: 5, Patch: "+fmt.Println()\n-fmt.Print()"},
		{Filename: "pkg/auth.go", Language: "go", Additions: 20, Deletions: 10},
	}

	info := buildFilesInfo(files)

	if !strings.Contains(info, "### File: pkg/main.go (go)") {
		t.Error("expected first file in output")
	}
	if !strings.Contains(info, "Changes: +10 -5") {
		t.Error("expected changes count in output")
	}
	if !strings.Contains(info, "```diff") {
		t.Error("expected diff block in output")
	}
}

func TestBuildFilesInfoTruncation(t *testing.T) {
	// Create 25 files to test truncation
	files := make([]*analyzer.FileChange, 25)
	for i := 0; i < 25; i++ {
		files[i] = &analyzer.FileChange{
			Filename:  "pkg/file.go",
			Language:  "go",
			Additions: 10,
			Deletions: 5,
		}
	}

	info := buildFilesInfo(files)

	// Should mention truncation
	if !strings.Contains(info, "... and 5 more files") {
		t.Error("expected truncation message in output")
	}
}

func TestParseReviewPlanValidJSON(t *testing.T) {
	response := `{
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

	plan, err := parseReviewPlan(response)

	if err != nil {
		t.Fatalf("parseReviewPlan failed: %v", err)
	}

	if plan.Summary != "Code review summary" {
		t.Errorf("expected summary 'Code review summary', got %q", plan.Summary)
	}
	if plan.Recommendation != "FIX" {
		t.Errorf("expected recommendation 'FIX', got %q", plan.Recommendation)
	}
	if len(plan.SecurityIssues) != 1 {
		t.Errorf("expected 1 security issue, got %d", len(plan.SecurityIssues))
	}
	if plan.SecurityIssues[0].Title != "SQL Injection" {
		t.Errorf("expected security issue title 'SQL Injection', got %q", plan.SecurityIssues[0].Title)
	}
}

func TestParseReviewPlanInvalidJSON(t *testing.T) {
	response := `This is not valid JSON`

	plan, err := parseReviewPlan(response)

	if err != nil {
		t.Fatalf("parseReviewPlan should not return error for invalid JSON: %v", err)
	}

	if plan.Summary != "Unable to parse AI response" {
		t.Errorf("expected fallback summary, got %q", plan.Summary)
	}
	if plan.Recommendation != "REVIEW_NEEDED" {
		t.Errorf("expected fallback recommendation 'REVIEW_NEEDED', got %q", plan.Recommendation)
	}
}

func TestParseReviewPlanEmptyIssues(t *testing.T) {
	response := `{
		"summary": "No issues found",
		"recommendation": "APPROVE",
		"issues": []
	}`

	plan, err := parseReviewPlan(response)

	if err != nil {
		t.Fatalf("parseReviewPlan failed: %v", err)
	}

	if len(plan.SecurityIssues) != 0 {
		t.Errorf("expected 0 security issues, got %d", len(plan.SecurityIssues))
	}
	if len(plan.BugIssues) != 0 {
		t.Errorf("expected 0 bug issues, got %d", len(plan.BugIssues))
	}
}

func TestCategorizeIssues(t *testing.T) {
	issues := []*RawIssue{
		{
			ID:          "SEC-001",
			Category:    "security",
			FilePath:    "pkg/auth.go",
			Line:        42,
			Severity:    "Critical",
			Title:       "SQL Injection",
			Description: "Vulnerable",
			Suggestion:  "Fix it",
			CWE:         "CWE-89",
			OWASP:       "A03:2021",
		},
		{
			ID:          "BUG-001",
			Category:    "bug",
			FilePath:    "pkg/handler.go",
			Line:        100,
			Severity:    "High",
			Title:       "Nil pointer",
			Description: "May panic",
			Suggestion:  "Add nil check",
		},
		{
			ID:          "STD-001",
			Category:    "style",
			FilePath:    "pkg/main.go",
			Line:        10,
			Severity:    "Low",
			Title:       "Bad naming",
			Description: "Not idiomatic",
			Suggestion:  "Rename",
		},
		{
			ID:          "STD-002",
			Category:    "maintainability",
			FilePath:    "pkg/main.go",
			Line:        20,
			Severity:    "Medium",
			Title:       "Long function",
			Description: "Too complex",
			Suggestion:  "Refactor",
		},
	}

	plan := categorizeIssues(issues)

	if len(plan.SecurityIssues) != 1 {
		t.Errorf("expected 1 security issue, got %d", len(plan.SecurityIssues))
	}
	if len(plan.BugIssues) != 1 {
		t.Errorf("expected 1 bug issue, got %d", len(plan.BugIssues))
	}
	if len(plan.StandardsViolations) != 2 {
		t.Errorf("expected 2 standards violations, got %d", len(plan.StandardsViolations))
	}
}

func TestCategorizeIssuesEmptyInput(t *testing.T) {
	issues := []*RawIssue{}

	plan := categorizeIssues(issues)

	if plan.SecurityIssues == nil {
		t.Error("expected SecurityIssues to be initialized")
	}
	if plan.BugIssues == nil {
		t.Error("expected BugIssues to be initialized")
	}
	if plan.StandardsViolations == nil {
		t.Error("expected StandardsViolations to be initialized")
	}
	if plan.TestGaps == nil {
		t.Error("expected TestGaps to be initialized")
	}
}

func TestGenerateFixPlan(t *testing.T) {
	plan := &ReviewPlan{
		SecurityIssues: []*SecurityIssue{
			{ID: "SEC-001", FilePath: "pkg/auth.go", Line: 42, Description: "SQL injection"},
		},
		BugIssues: []*BugIssue{
			{ID: "BUG-001", FilePath: "pkg/handler.go", Line: 100, Description: "Nil pointer"},
		},
		StandardsViolations: []*StandardsViolation{
			{ID: "STD-001", FilePath: "pkg/main.go", Line: 10, Message: "Bad naming"},
		},
		TestGaps: []*TestGap{
			{ID: "TEST-001", Description: "Missing tests", TestFile: "pkg/main_test.go"},
		},
	}

	fixPlan := generateFixPlan(plan)

	if len(fixPlan.ParallelGroups) != 3 {
		t.Errorf("expected 3 parallel groups, got %d", len(fixPlan.ParallelGroups))
	}

	// First group should be security fixes
	if len(fixPlan.ParallelGroups[0]) != 1 {
		t.Errorf("expected 1 security fix in first group, got %d", len(fixPlan.ParallelGroups[0]))
	}
	if fixPlan.ParallelGroups[0][0].Type != "security" {
		t.Errorf("expected first group to be security fixes, got %q", fixPlan.ParallelGroups[0][0].Type)
	}
	if fixPlan.ParallelGroups[0][0].Priority != "critical" {
		t.Errorf("expected security fixes to have critical priority, got %q", fixPlan.ParallelGroups[0][0].Priority)
	}

	// Sequential tasks should be test generation
	if len(fixPlan.SequentialTasks) != 1 {
		t.Errorf("expected 1 sequential task, got %d", len(fixPlan.SequentialTasks))
	}
	if fixPlan.SequentialTasks[0].Type != "test" {
		t.Errorf("expected sequential task to be test type, got %q", fixPlan.SequentialTasks[0].Type)
	}
}

func TestGenerateFixPlanEmptyPlan(t *testing.T) {
	plan := &ReviewPlan{
		SecurityIssues:      []*SecurityIssue{},
		BugIssues:           []*BugIssue{},
		StandardsViolations: []*StandardsViolation{},
		TestGaps:            []*TestGap{},
	}

	fixPlan := generateFixPlan(plan)

	if fixPlan == nil {
		t.Fatal("expected non-nil fix plan")
	}
	if len(fixPlan.ParallelGroups) != 0 {
		t.Errorf("expected 0 parallel groups, got %d", len(fixPlan.ParallelGroups))
	}
	if len(fixPlan.SequentialTasks) != 0 {
		t.Errorf("expected 0 sequential tasks, got %d", len(fixPlan.SequentialTasks))
	}
}

func TestRawIssueJSONMarshaling(t *testing.T) {
	issue := &RawIssue{
		ID:          "TEST-001",
		FilePath:    "pkg/test.go",
		Line:        50,
		Severity:    "Medium",
		Category:    "bug",
		Title:       "Test Issue",
		Description: "Test Description",
		Suggestion:  "Test Suggestion",
		CWE:         "CWE-000",
		OWASP:       "A00:0000",
	}

	data, err := json.Marshal(issue)
	if err != nil {
		t.Fatalf("failed to marshal RawIssue: %v", err)
	}

	var unmarshaled RawIssue
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal RawIssue: %v", err)
	}

	if unmarshaled.ID != issue.ID {
		t.Errorf("expected ID %q, got %q", issue.ID, unmarshaled.ID)
	}
	if unmarshaled.Line != issue.Line {
		t.Errorf("expected Line %d, got %d", issue.Line, unmarshaled.Line)
	}
}

func TestReviewPlanJSONMarshaling(t *testing.T) {
	plan := &ReviewPlan{
		Summary:        "Test Summary",
		SecurityIssues: []*SecurityIssue{},
		BugIssues:      []*BugIssue{},
		StandardsViolations: []*StandardsViolation{
			{ID: "STD-001", FilePath: "test.go", Rule: "style"},
		},
		TestGaps:       []*TestGap{},
		FixPlan:        &FixPlan{},
		Recommendation: "APPROVE",
	}

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("failed to marshal ReviewPlan: %v", err)
	}

	if !strings.Contains(string(data), "Test Summary") {
		t.Error("expected summary in JSON output")
	}
	if !strings.Contains(string(data), "APPROVE") {
		t.Error("expected recommendation in JSON output")
	}
}

func TestPlanReviewEmptyFiles(t *testing.T) {
	// This test would require mocking the agent
	// For now, we skip it
	t.Skip("Skipping - requires agent mock")
}

func TestPlanReviewContextTimeout(t *testing.T) {
	// This test would require mocking the agent and context
	// For now, we skip it
	t.Skip("Skipping - requires agent mock and context setup")
}

func TestBuildFilesInfoLargeFileList(t *testing.T) {
	// Test with exactly 20 files (boundary)
	files := make([]*analyzer.FileChange, 20)
	for i := 0; i < 20; i++ {
		files[i] = &analyzer.FileChange{
			Filename:  "pkg/file.go",
			Language:  "go",
			Additions: 10,
			Deletions: 5,
		}
	}

	info := buildFilesInfo(files)

	// Should not have truncation message
	if strings.Contains(info, "... and") {
		t.Error("should not truncate at exactly 20 files")
	}

	// Test with 21 files (should truncate)
	files = append(files, &analyzer.FileChange{
		Filename:  "pkg/file21.go",
		Language:  "go",
		Additions: 10,
		Deletions: 5,
	})

	info = buildFilesInfo(files)

	if !strings.Contains(info, "... and 1 more files") {
		t.Error("should truncate at 21 files")
	}
}
