package reviewer

import (
	"sort"

	"github.com/yourorg/github-code-agent/features/standards"
)

// PrioritizedIssues contains issues sorted by priority
type PrioritizedIssues struct {
	Critical []*Issue
	High     []*Issue
	Medium   []*Issue
	Low      []*Issue
	All      []*Issue
}

// PrioritizeIssues sorts and prioritizes issues by severity and category
func PrioritizeIssues(reviewIssues []*Issue, violations []*standards.Violation, securityIssues []*SecurityIssue) *PrioritizedIssues {
	result := &PrioritizedIssues{
		Critical: []*Issue{},
		High:     []*Issue{},
		Medium:   []*Issue{},
		Low:      []*Issue{},
		All:      []*Issue{},
	}

	// Convert all issues to common Issue type
	allIssues := make([]*Issue, 0)

	// Add review issues
	allIssues = append(allIssues, reviewIssues...)

	// Convert violations to issues
	for _, v := range violations {
		issue := violationToIssue(v)
		allIssues = append(allIssues, issue)
	}

	// Convert security issues to issues
	for _, s := range securityIssues {
		issue := securityIssueToIssue(s)
		allIssues = append(allIssues, issue)
	}

	// Deduplicate issues (same file, line, and similar message)
	allIssues = deduplicateIssues(allIssues)

	// Sort by severity and category
	sort.Slice(allIssues, func(i, j int) bool {
		return comparePriority(allIssues[i], allIssues[j])
	})

	// Group by severity
	for _, issue := range allIssues {
		result.All = append(result.All, issue)

		switch issue.Severity {
		case SeverityCritical:
			result.Critical = append(result.Critical, issue)
		case SeverityHigh:
			result.High = append(result.High, issue)
		case SeverityMedium:
			result.Medium = append(result.Medium, issue)
		case SeverityLow:
			result.Low = append(result.Low, issue)
		}
	}

	return result
}

// violationToIssue converts a standards violation to an Issue
func violationToIssue(v *standards.Violation) *Issue {
	// Map violation severity to issue severity
	severity := SeverityLow
	switch v.Severity {
	case standards.SeverityError:
		severity = SeverityHigh
	case standards.SeverityWarning:
		severity = SeverityMedium
	case standards.SeverityInfo:
		severity = SeverityLow
	}

	// Map violation type to category
	category := CategoryStyle
	if v.Type == standards.ViolationTypeSecurity {
		category = CategorySecurity
		severity = SeverityCritical
	}

	return &Issue{
		ID:          v.ID,
		FilePath:    v.FilePath,
		Line:        v.Line,
		Column:      v.Column,
		Severity:    severity,
		Category:    category,
		Title:       v.Message,
		Description: v.Message,
		Suggestion:  v.Suggestion,
	}
}

// securityIssueToIssue converts a SecurityIssue to an Issue
func securityIssueToIssue(s *SecurityIssue) *Issue {
	return &Issue{
		ID:          s.ID,
		FilePath:    s.FilePath,
		Line:        s.Line,
		Severity:    s.Severity,
		Category:    CategorySecurity,
		Title:       s.Title,
		Description: s.Description + "\n\nRemediation: " + s.Remediation,
		Suggestion:  s.Remediation,
	}
}

// deduplicateIssues removes duplicate issues
func deduplicateIssues(issues []*Issue) []*Issue {
	seen := make(map[string]bool)
	result := make([]*Issue, 0, len(issues))

	for _, issue := range issues {
		// Create key from file, line, and first 50 chars of title
		title := issue.Title
		if len(title) > 50 {
			title = title[:50]
		}
		key := issue.FilePath + ":" + string(rune(issue.Line)) + ":" + title

		if !seen[key] {
			seen[key] = true
			result = append(result, issue)
		}
	}

	return result
}

// comparePriority compares two issues for sorting (higher priority first)
func comparePriority(a, b *Issue) bool {
	// First, compare by severity
	severityPriority := map[string]int{
		SeverityCritical: 4,
		SeverityHigh:     3,
		SeverityMedium:   2,
		SeverityLow:      1,
	}

	aPriority := severityPriority[a.Severity]
	bPriority := severityPriority[b.Severity]

	if aPriority != bPriority {
		return aPriority > bPriority
	}

	// Then by category (security first)
	categoryPriority := map[string]int{
		CategorySecurity:        5,
		CategoryBug:             4,
		CategoryPerformance:     3,
		CategoryMaintainability: 2,
		CategoryStyle:           1,
	}

	aCatPriority := categoryPriority[a.Category]
	bCatPriority := categoryPriority[b.Category]

	if aCatPriority != bCatPriority {
		return aCatPriority > bCatPriority
	}

	// Finally by file path and line number
	if a.FilePath != b.FilePath {
		return a.FilePath < b.FilePath
	}

	return a.Line < b.Line
}

// FilterByThreshold filters issues based on severity threshold
func FilterByThreshold(issues []*Issue, threshold string) []*Issue {
	severityLevel := map[string]int{
		SeverityCritical: 4,
		SeverityHigh:     3,
		SeverityMedium:   2,
		SeverityLow:      1,
	}

	thresholdLevel := severityLevel[threshold]
	if thresholdLevel == 0 {
		return issues // Invalid threshold, return all
	}

	filtered := make([]*Issue, 0)
	for _, issue := range issues {
		if severityLevel[issue.Severity] >= thresholdLevel {
			filtered = append(filtered, issue)
		}
	}

	return filtered
}
