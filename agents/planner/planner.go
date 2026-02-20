// Package planner provides AI-powered code review planning capabilities.
//
// The planner analyzes code changes and generates comprehensive review plans
// including security issues, bug detection, standards violations, and test gaps.
// It creates prioritized fix plans with dependency tracking for efficient
// automated code review.
package planner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	_ "embed"
	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/Agent-Field/agentfield/sdk/go/ai"
	"github.com/yourorg/github-code-agent/agents/analyzer"
	"github.com/yourorg/github-code-agent/pkg/constants"
	"github.com/yourorg/github-code-agent/pkg/utils"
)

//go:embed prompts/system.md
var plannerSystemPrompt string

//go:embed prompts/task.md
var plannerTaskPrompt string

// ReviewPlan is the comprehensive plan created by the planner
type ReviewPlan struct {
	// Summary of the PR
	Summary string `json:"summary"`

	// Security issues (must fix, highest priority)
	SecurityIssues []*SecurityIssue `json:"security_issues"`

	// Logic bugs (must fix)
	BugIssues []*BugIssue `json:"bug_issues"`

	// Standards violations (should fix, often auto-fixable)
	StandardsViolations []*StandardsViolation `json:"standards_violations"`

	// Test gaps (should add)
	TestGaps []*TestGap `json:"test_gaps"`

	// Fix plan with dependencies
	FixPlan *FixPlan `json:"fix_plan"`

	// Recommendation is one of: APPROVE, FIX, REVIEW_NEEDED
	Recommendation string `json:"recommendation"`
}

// SecurityIssue represents a security vulnerability
type SecurityIssue struct {
	// ID is a unique identifier for the issue
	ID string `json:"id"`
	// FilePath is the path to the file containing the issue
	FilePath string `json:"file_path"`
	// Line is the line number where the issue occurs
	Line int `json:"line"`
	// Type is the category of security issue
	Type string `json:"type"`
	// Severity is one of: Critical, High, Medium, Low
	Severity string `json:"severity"`
	// Title is a brief description of the issue
	Title string `json:"title"`
	// Description provides detailed information about the issue
	Description string `json:"description"`
	// CWE is the Common Weakness Enumeration identifier
	CWE string `json:"cwe"`
	// OWASP is the OWASP Top 10 category
	OWASP string `json:"owasp"`
	// Remediation suggests how to fix the issue
	Remediation string `json:"remediation"`
}

// BugIssue represents a logic bug
type BugIssue struct {
	// ID is a unique identifier for the issue
	ID string `json:"id"`
	// FilePath is the path to the file containing the issue
	FilePath string `json:"file_path"`
	// Line is the line number where the issue occurs
	Line int `json:"line"`
	// Type is the category of bug
	Type string `json:"type"`
	// Severity is one of: Critical, High, Medium, Low
	Severity string `json:"severity"`
	// Title is a brief description of the bug
	Title string `json:"title"`
	// Description provides detailed information about the bug
	Description string `json:"description"`
	// WhyItFails explains why the code fails
	WhyItFails string `json:"why_it_fails"`
	// ExpectedBehavior describes what the code should do
	ExpectedBehavior string `json:"expected_behavior"`
}

// StandardsViolation represents a coding standards violation
type StandardsViolation struct {
	// ID is a unique identifier for the violation
	ID string `json:"id"`
	// FilePath is the path to the file containing the violation
	FilePath string `json:"file_path"`
	// Line is the line number where the violation occurs
	Line int `json:"line"`
	// Rule is the name of the violated rule
	Rule string `json:"rule"`
	// Severity is one of: Critical, High, Medium, Low
	Severity string `json:"severity"`
	// Message describes the violation
	Message string `json:"message"`
	// Why explains why this is a violation
	Why string `json:"why"`
	// Suggestion provides guidance on how to fix it
	Suggestion string `json:"suggestion"`
	// AutoFixable indicates whether the violation can be automatically fixed
	AutoFixable bool `json:"auto_fixable"`
}

// TestGap represents a missing test
type TestGap struct {
	// ID is a unique identifier for the gap
	ID string `json:"id"`
	// Description explains what tests are missing
	Description string `json:"description"`
	// TestFile is the suggested test file path
	TestFile string `json:"test_file"`
	// Framework is the testing framework to use
	Framework string `json:"framework"`
	// TestCount is the estimated number of tests needed
	TestCount int `json:"test_count"`
	// TestCases lists specific test cases to implement
	TestCases []string `json:"test_cases"`
}

// FixPlan contains the execution plan for fixes
type FixPlan struct {
	// ParallelGroups contains groups of fixes that can run in parallel
	ParallelGroups [][]*FixTask `json:"parallel_groups"`

	// SequentialTasks contains fixes that must run in order
	SequentialTasks []*FixTask `json:"sequential_tasks"`
}

// FixTask represents a single fix task
type FixTask struct {
	// ID is a unique identifier for the task
	ID string `json:"id"`
	// Type is one of: security, bug, standards, test
	Type string `json:"type"`
	// File is the path to the file to fix
	File string `json:"file"`
	// Line is the line number to fix
	Line int `json:"line"`
	// Description explains what needs to be fixed
	Description string `json:"description"`
	// Priority is one of: critical, high, medium, low
	Priority string `json:"priority"`
	// DependsOn lists task IDs that must complete first
	DependsOn []string `json:"depends_on"`
	// EstTokens is the estimated token count for the fix
	EstTokens int `json:"est_tokens"`
}

// Planner handles AI-powered code review planning
type Planner struct {
	agent *agent.Agent
}

// NewPlanner creates a new code review planner
func NewPlanner(a *agent.Agent) *Planner {
	return &Planner{
		agent: a,
	}
}

// PlanReview performs comprehensive code review and creates a fix plan
func (p *Planner) PlanReview(ctx context.Context, files []*analyzer.FileChange, prContext map[string]any) (*ReviewPlan, error) {
	// Filter out files that shouldn't be reviewed
	reviewableFiles := filterReviewableFiles(files)

	if len(reviewableFiles) == 0 {
		return &ReviewPlan{
			Summary:             "No reviewable files found",
			SecurityIssues:      []*SecurityIssue{},
			BugIssues:           []*BugIssue{},
			StandardsViolations: []*StandardsViolation{},
			TestGaps:            []*TestGap{},
			FixPlan:             &FixPlan{},
			Recommendation:      "APPROVE",
		}, nil
	}

	// Build review context
	reviewCtx := buildReviewContext(reviewableFiles, prContext)

	// Build prompt using template
	prInfo := buildPRInfo(reviewCtx)
	filesInfo := buildFilesInfo(reviewCtx.Files)
	taskPrompt := fmt.Sprintf(plannerTaskPrompt, prInfo, filesInfo)

	// Create context with timeout
	aiCtx, cancel := context.WithTimeout(ctx, constants.DefaultAITimeout)
	defer cancel()

	// Use AgentField's built-in AI method
	response, err := p.agent.AI(aiCtx, taskPrompt,
		ai.WithSystem(plannerSystemPrompt),
		ai.WithTemperature(constants.DefaultAITemperature),
		ai.WithMaxTokens(constants.ReviewAIMaxTokens))

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("AI review planning timeout: request exceeded 10 minute limit: %w", err)
		}
		return nil, fmt.Errorf("AI review planning failed: %w", err)
	}

	// Parse AI response into structured review plan
	plan, err := parseReviewPlan(response.Text())
	if err != nil {
		return nil, fmt.Errorf("failed to parse review plan: %w", err)
	}

	return plan, nil
}

// parseReviewPlan parses the AI response into a ReviewPlan
func parseReviewPlan(text string) (*ReviewPlan, error) {
	// Try to extract JSON from the response
	jsonText := utils.ExtractJSON(text)

	var response struct {
		Issues         []*RawIssue `json:"issues"`
		Summary        string      `json:"summary"`
		Recommendation string      `json:"recommendation"`
	}

	if err := json.Unmarshal([]byte(jsonText), &response); err != nil {
		// If JSON parsing fails, create empty plan
		return &ReviewPlan{
			Summary:             "Unable to parse AI response",
			SecurityIssues:      []*SecurityIssue{},
			BugIssues:           []*BugIssue{},
			StandardsViolations: []*StandardsViolation{},
			TestGaps:            []*TestGap{},
			FixPlan:             &FixPlan{},
			Recommendation:      "REVIEW_NEEDED",
		}, nil
	}

	// Categorize issues into plan
	plan := categorizeIssues(response.Issues)
	plan.Summary = response.Summary
	plan.Recommendation = response.Recommendation

	// Generate fix plan
	plan.FixPlan = generateFixPlan(plan)

	return plan, nil
}

// RawIssue is the raw issue format from AI response
type RawIssue struct {
	ID          string `json:"id"`
	FilePath    string `json:"file_path"`
	Line        int    `json:"line"`
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
	CWE         string `json:"cwe"`
	OWASP       string `json:"owasp"`
}

// categorizeIssues categorizes raw issues into the plan structure
func categorizeIssues(issues []*RawIssue) *ReviewPlan {
	plan := &ReviewPlan{
		SecurityIssues:      []*SecurityIssue{},
		BugIssues:           []*BugIssue{},
		StandardsViolations: []*StandardsViolation{},
		TestGaps:            []*TestGap{},
	}

	for _, issue := range issues {
		// Assign ID if missing
		if issue.ID == "" {
			issue.ID = fmt.Sprintf("%s-%d", issue.FilePath, issue.Line)
		}

		// Categorize based on category field
		switch issue.Category {
		case "security":
			plan.SecurityIssues = append(plan.SecurityIssues, &SecurityIssue{
				ID:          issue.ID,
				FilePath:    issue.FilePath,
				Line:        issue.Line,
				Type:        issue.Category,
				Severity:    issue.Severity,
				Title:       issue.Title,
				Description: issue.Description,
				CWE:         issue.CWE,
				OWASP:       issue.OWASP,
				Remediation: issue.Suggestion,
			})
		case "bug":
			plan.BugIssues = append(plan.BugIssues, &BugIssue{
				ID:               issue.ID,
				FilePath:         issue.FilePath,
				Line:             issue.Line,
				Type:             issue.Category,
				Severity:         issue.Severity,
				Title:            issue.Title,
				Description:      issue.Description,
				WhyItFails:       issue.Suggestion,
				ExpectedBehavior: "",
			})
		case "style", "maintainability":
			plan.StandardsViolations = append(plan.StandardsViolations, &StandardsViolation{
				ID:          issue.ID,
				FilePath:    issue.FilePath,
				Line:        issue.Line,
				Rule:        issue.Category,
				Severity:    issue.Severity,
				Message:     issue.Title,
				Why:         issue.Description,
				Suggestion:  issue.Suggestion,
				AutoFixable: false,
			})
		}
	}

	return plan
}

// generateFixPlan creates an execution plan from the review plan
func generateFixPlan(plan *ReviewPlan) *FixPlan {
	fixPlan := &FixPlan{
		ParallelGroups:  [][]*FixTask{},
		SequentialTasks: []*FixTask{},
	}

	// Group security fixes (highest priority, can run in parallel)
	if len(plan.SecurityIssues) > 0 {
		securityGroup := make([]*FixTask, 0, len(plan.SecurityIssues))
		for _, issue := range plan.SecurityIssues {
			securityGroup = append(securityGroup, &FixTask{
				ID:          "sec-" + issue.ID,
				Type:        "security",
				File:        issue.FilePath,
				Line:        issue.Line,
				Description: issue.Description,
				Priority:    "critical",
				EstTokens:   1000,
			})
		}
		fixPlan.ParallelGroups = append(fixPlan.ParallelGroups, securityGroup)
	}

	// Group bug fixes (high priority, can run in parallel)
	if len(plan.BugIssues) > 0 {
		bugGroup := make([]*FixTask, 0, len(plan.BugIssues))
		for _, issue := range plan.BugIssues {
			bugGroup = append(bugGroup, &FixTask{
				ID:          "bug-" + issue.ID,
				Type:        "bug",
				File:        issue.FilePath,
				Line:        issue.Line,
				Description: issue.Description,
				Priority:    "high",
				EstTokens:   1500,
			})
		}
		fixPlan.ParallelGroups = append(fixPlan.ParallelGroups, bugGroup)
	}

	// Group standards fixes (medium priority, can run in parallel)
	if len(plan.StandardsViolations) > 0 {
		standardsGroup := make([]*FixTask, 0, len(plan.StandardsViolations))
		for _, violation := range plan.StandardsViolations {
			standardsGroup = append(standardsGroup, &FixTask{
				ID:          "std-" + violation.ID,
				Type:        "standards",
				File:        violation.FilePath,
				Line:        violation.Line,
				Description: violation.Message,
				Priority:    "medium",
				EstTokens:   500,
			})
		}
		fixPlan.ParallelGroups = append(fixPlan.ParallelGroups, standardsGroup)
	}

	// Test tasks run after fixes (sequential)
	if len(plan.TestGaps) > 0 {
		for _, gap := range plan.TestGaps {
			fixPlan.SequentialTasks = append(fixPlan.SequentialTasks, &FixTask{
				ID:          "test-" + gap.ID,
				Type:        "test",
				File:        gap.TestFile,
				Description: gap.Description,
				Priority:    "medium",
				EstTokens:   1500,
			})
		}
	}

	return fixPlan
}

// ReviewContext provides dynamic context to prompts
type ReviewContext struct {
	PRTitle       string
	PRDescription string
	Files         []*analyzer.FileChange
}

// buildReviewContext builds the review context from files and PR context
func buildReviewContext(files []*analyzer.FileChange, prContext map[string]any) ReviewContext {
	ctx := ReviewContext{
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

// buildPRInfo builds the PR info section of the prompt
func buildPRInfo(ctx ReviewContext) string {
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

// buildFilesInfo builds the files info section of the prompt
func buildFilesInfo(files []*analyzer.FileChange) string {
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

// filterReviewableFiles filters out files that shouldn't be reviewed
func filterReviewableFiles(files []*analyzer.FileChange) []*analyzer.FileChange {
	var reviewable []*analyzer.FileChange

	for _, file := range files {
		// Skip deleted files
		if file.Status == constants.FileStatusRemoved {
			continue
		}

		// Skip binary files and common non-code files
		if utils.ShouldSkipFile(file.Filename) {
			continue
		}

		reviewable = append(reviewable, file)
	}

	return reviewable
}
