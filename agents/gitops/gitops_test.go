package gitops

import (
	"strings"
	"testing"

	"github.com/yourorg/github-code-agent/agents/fixer"
)

func TestGenerateCommitMessage(t *testing.T) {
	tests := []struct {
		name    string
		patches []*fixer.CodePatch
		want    string
	}{
		{
			name:    "empty patches",
			patches: []*fixer.CodePatch{},
			want:    "🤖 Automated fixes",
		},
		{
			name: "single patch",
			patches: []*fixer.CodePatch{
				{
					IssueID:     "1",
					Description: "Fix memory leak",
				},
			},
			want: "🤖 Auto-fix: Fix memory leak",
		},
		{
			name: "multiple patches",
			patches: []*fixer.CodePatch{
				{IssueID: "1"},
				{IssueID: "2"},
				{IssueID: "3"},
			},
			want: "🤖 Auto-fix: 3 issues resolved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateCommitMessage(tt.patches)
			if got != tt.want {
				t.Errorf("GenerateCommitMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGeneratePRTitle(t *testing.T) {
	tests := []struct {
		name     string
		prNumber int
		fixCount int
		want     string
	}{
		{
			name:     "single fix",
			prNumber: 123,
			fixCount: 1,
			want:     "🤖 Automated fixes for PR #123 (1 issues)",
		},
		{
			name:     "multiple fixes",
			prNumber: 456,
			fixCount: 5,
			want:     "🤖 Automated fixes for PR #456 (5 issues)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GeneratePRTitle(tt.prNumber, tt.fixCount)
			if got != tt.want {
				t.Errorf("GeneratePRTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGeneratePRBody(t *testing.T) {
	patches := []*fixer.CodePatch{
		{
			IssueID:     "1",
			Description: "Fix null pointer",
			FilePath:    "main.go",
			Line:        10,
		},
		{
			IssueID:     "2",
			Description: "Remove unused variable",
			FilePath:    "utils.go",
			Line:        25,
		},
	}

	body := GeneratePRBody(123, patches)

	// Check body contains key information
	if !strings.Contains(body, "PR #123") {
		t.Error("Body should reference original PR number")
	}
	if !strings.Contains(body, "Fix null pointer") {
		t.Error("Body should list first issue")
	}
	if !strings.Contains(body, "Remove unused variable") {
		t.Error("Body should list second issue")
	}
	if !strings.Contains(body, "main.go") {
		t.Error("Body should include file paths")
	}
	if !strings.Contains(body, "line 10") {
		t.Error("Body should include line numbers")
	}
	if !strings.Contains(body, "✅ Syntax correctness") {
		t.Error("Body should mention validation checks")
	}
}

func TestGetSeverityEmoji(t *testing.T) {
	tests := []struct {
		severity string
		want     string
	}{
		{"Critical", "🔴"},
		{"High", "🟠"},
		{"Medium", "🟡"},
		{"Low", "🔵"},
		{"Unknown", "⚪"},
		{"", "⚪"},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			got := getSeverityEmoji(tt.severity)
			if got != tt.want {
				t.Errorf("getSeverityEmoji(%q) = %q, want %q", tt.severity, got, tt.want)
			}
		})
	}
}

func TestGenerateSummaryComment(t *testing.T) {
	tests := []struct {
		name         string
		issueCount   int
		fixPRNum     int
		fixPRURL     string
		mode         OperationMode
		wantContains []string
	}{
		{
			name:       "YOLO mode",
			issueCount: 5,
			mode:       YOLOMode,
			wantContains: []string{
				"Automated Code Review Complete",
				"5 issue(s)",
				"applied fixes directly",
			},
		},
		{
			name:       "Safe mode",
			issueCount: 3,
			fixPRNum:   124,
			fixPRURL:   "https://github.com/org/repo/pull/124",
			mode:       SafeMode,
			wantContains: []string{
				"Automated Code Review Complete",
				"3 issue(s)",
				"PR #124",
				"Review fixes",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSummaryComment(tt.issueCount, tt.fixPRNum, tt.fixPRURL, tt.mode)

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("GenerateSummaryComment() should contain %q, got:\n%s", want, got)
				}
			}
		})
	}
}

func TestFormatCommentBody(t *testing.T) {
	comment := &ReviewComment{
		FilePath: "main.go",
		Line:     10,
		Body:     "Variable is not used",
		Severity: "Medium",
	}

	body := formatCommentBody(comment)

	if !strings.Contains(body, "🟡") {
		t.Error("Body should contain severity emoji")
	}
	if !strings.Contains(body, "Medium") {
		t.Error("Body should contain severity level")
	}
	if !strings.Contains(body, "Variable is not used") {
		t.Error("Body should contain comment body")
	}
	if !strings.Contains(body, "Automated review") {
		t.Error("Body should contain automation note")
	}
}
