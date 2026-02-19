package reviewer

import (
	"testing"

	"github.com/yourorg/github-code-agent/agents/standards"
)

func TestPrioritizeIssues(t *testing.T) {
	reviewIssues := []*Issue{
		{
			FilePath: "main.go",
			Line:     10,
			Severity: SeverityHigh,
			Category: CategoryBug,
			Title:    "Potential nil pointer",
		},
		{
			FilePath: "util.go",
			Line:     20,
			Severity: SeverityLow,
			Category: CategoryStyle,
			Title:    "Line too long",
		},
	}

	violations := []*standards.Violation{
		{
			FilePath: "config.go",
			Line:     15,
			Severity: standards.SeverityError,
			Type:     standards.ViolationTypeSecurity,
			Message:  "Hardcoded secret detected",
		},
	}

	securityIssues := []*SecurityIssue{
		{
			FilePath: "auth.go",
			Line:     5,
			Severity: SeverityCritical,
			Title:    "SQL injection vulnerability",
		},
	}

	result := PrioritizeIssues(reviewIssues, violations, securityIssues)

	// Check totals
	if len(result.All) != 4 {
		t.Errorf("Expected 4 total issues, got %d", len(result.All))
	}

	// Check critical issues (should include security)
	if len(result.Critical) != 2 {
		t.Errorf("Expected 2 critical issues, got %d", len(result.Critical))
	}

	// Check sorting - critical should be first
	if result.All[0].Severity != SeverityCritical {
		t.Errorf("First issue should be Critical, got %s", result.All[0].Severity)
	}
}

func TestComparePriority(t *testing.T) {
	tests := []struct {
		name     string
		issueA   *Issue
		issueB   *Issue
		expected bool // true if A has higher priority
	}{
		{
			name: "critical before high",
			issueA: &Issue{
				Severity: SeverityCritical,
				Category: CategoryBug,
			},
			issueB: &Issue{
				Severity: SeverityHigh,
				Category: CategoryBug,
			},
			expected: true,
		},
		{
			name: "security before bug (same severity)",
			issueA: &Issue{
				Severity: SeverityHigh,
				Category: CategorySecurity,
			},
			issueB: &Issue{
				Severity: SeverityHigh,
				Category: CategoryBug,
			},
			expected: true,
		},
		{
			name: "earlier file first (same severity and category)",
			issueA: &Issue{
				FilePath: "a.go",
				Line:     10,
				Severity: SeverityMedium,
				Category: CategoryBug,
			},
			issueB: &Issue{
				FilePath: "b.go",
				Line:     10,
				Severity: SeverityMedium,
				Category: CategoryBug,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := comparePriority(tt.issueA, tt.issueB)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFilterByThreshold(t *testing.T) {
	issues := []*Issue{
		{Severity: SeverityCritical},
		{Severity: SeverityHigh},
		{Severity: SeverityMedium},
		{Severity: SeverityLow},
	}

	tests := []struct {
		threshold string
		expected  int
	}{
		{SeverityCritical, 1},
		{SeverityHigh, 2},
		{SeverityMedium, 3},
		{SeverityLow, 4},
		{"invalid", 4}, // Should return all if invalid
	}

	for _, tt := range tests {
		t.Run(tt.threshold, func(t *testing.T) {
			result := FilterByThreshold(issues, tt.threshold)
			if len(result) != tt.expected {
				t.Errorf("Expected %d issues for threshold %s, got %d",
					tt.expected, tt.threshold, len(result))
			}
		})
	}
}

func TestDeduplicateIssues(t *testing.T) {
	issues := []*Issue{
		{
			FilePath: "main.go",
			Line:     10,
			Title:    "Duplicate issue",
		},
		{
			FilePath: "main.go",
			Line:     10,
			Title:    "Duplicate issue",
		},
		{
			FilePath: "main.go",
			Line:     20,
			Title:    "Different line",
		},
	}

	result := deduplicateIssues(issues)

	if len(result) != 2 {
		t.Errorf("Expected 2 unique issues after deduplication, got %d", len(result))
	}
}
