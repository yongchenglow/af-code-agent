package planner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/Agent-Field/agentfield/sdk/go/ai"
	"github.com/yourorg/github-code-agent/agents/analyzer"
	"github.com/yourorg/github-code-agent/agents/prompts"
	"github.com/yourorg/github-code-agent/pkg/constants"
	"github.com/yourorg/github-code-agent/pkg/utils"
)

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

	// Recommendation
	Recommendation string `json:"recommendation"` // APPROVE, FIX, REVIEW_NEEDED
}

// SecurityIssue represents a security vulnerability
type SecurityIssue struct {
	ID          string `json:"id"`
	FilePath    string `json:"file_path"`
	Line        int    `json:"line"`
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CWE         string `json:"cwe"`
	OWASP       string `json:"owasp"`
	Remediation string `json:"remediation"`
}

// BugIssue represents a logic bug
type BugIssue struct {
	ID               string `json:"id"`
	FilePath         string `json:"file_path"`
	Line             int    `json:"line"`
	Type             string `json:"type"`
	Severity         string `json:"severity"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	WhyItFails       string `json:"why_it_fails"`
	ExpectedBehavior string `json:"expected_behavior"`
}

// StandardsViolation represents a coding standards violation
type StandardsViolation struct {
	ID          string `json:"id"`
	FilePath    string `json:"file_path"`
	Line        int    `json:"line"`
	Rule        string `json:"rule"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Why         string `json:"why"`
	Suggestion  string `json:"suggestion"`
	AutoFixable bool   `json:"auto_fixable"`
}

// TestGap represents a missing test
type TestGap struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	TestFile    string   `json:"test_file"`
	Framework   string   `json:"framework"`
	TestCount   int      `json:"test_count"`
	TestCases   []string `json:"test_cases"`
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
	ID          string   `json:"id"`
	Type        string   `json:"type"` // security, bug, standards, test
	File        string   `json:"file"`
	Line        int      `json:"line"`
	Description string   `json:"description"`
	Priority    string   `json:"priority"` // critical, high, medium, low
	DependsOn   []string `json:"depends_on"`
	EstTokens   int      `json:"est_tokens"`
}

// Planner handles AI-powered code review planning
type Planner struct {
	agent  *agent.Agent
	prompts *prompts.ReviewerPrompts
}

// NewPlanner creates a new code review planner
func NewPlanner(a *agent.Agent) *Planner {
	return &Planner{
		agent:  a,
		prompts: prompts.NewReviewerPrompts(),
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
	reviewCtx := prompts.ReviewContext{
		Files:     reviewableFiles,
		PRContext: prContext,
	}

	// Build prompt using template system
	taskPrompt := p.prompts.TaskPrompt(reviewCtx)

	// Create context with timeout
	aiCtx, cancel := context.WithTimeout(ctx, constants.DefaultAITimeout)
	defer cancel()

	// Use AgentField's built-in AI method
	response, err := p.agent.AI(aiCtx, taskPrompt,
		ai.WithSystem(p.prompts.SystemPrompt),
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
