package analyzer

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v57/github"
	ghclient "github.com/yourorg/github-code-agent/pkg/github"
)

// Analyzer handles code analysis
type Analyzer struct {
	githubClient *github.Client
}

// NewAnalyzer creates a new code analyzer
func NewAnalyzer(githubClient *github.Client) *Analyzer {
	return &Analyzer{
		githubClient: githubClient,
	}
}

// AnalyzePR analyzes a pull request and returns all changed files
func (a *Analyzer) AnalyzePR(ctx context.Context, owner, repo string, prNumber int) (*AnalysisResult, error) {
	// Fetch PR files
	files, err := ghclient.GetPRFiles(ctx, a.githubClient, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR files: %w", err)
	}

	// Convert to our FileChange type
	fileChanges := make([]*FileChange, 0, len(files))
	totalLOC := 0

	for _, file := range files {
		fc := &FileChange{
			Filename:  file.Filename,
			Status:    file.Status,
			Additions: file.Additions,
			Deletions: file.Deletions,
			Changes:   file.Changes,
			Patch:     file.Patch,
			BlobURL:   file.BlobURL,
			Language:  detectLanguage(file.Filename),
		}

		// Fetch file content for analysis
		content, err := a.fetchFileContent(ctx, owner, repo, file.Filename, prNumber)
		if err == nil {
			fc.Content = content
		}

		fileChanges = append(fileChanges, fc)
		totalLOC += file.Additions
	}

	// Calculate basic metrics
	metrics := &Metrics{
		LinesOfCode: totalLOC,
	}

	// Generate summary
	summary := fmt.Sprintf("Analyzed %d files with %d lines of code changed", len(fileChanges), totalLOC)

	return &AnalysisResult{
		Files:   fileChanges,
		Metrics: metrics,
		Summary: summary,
	}, nil
}

// fetchFileContent fetches the content of a file from GitHub
func (a *Analyzer) fetchFileContent(ctx context.Context, owner, repo, path string, prNumber int) (string, error) {
	// Get PR to find head ref
	pr, _, err := a.githubClient.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return "", err
	}

	// Get file content from head ref
	opts := &github.RepositoryContentGetOptions{
		Ref: pr.Head.GetSHA(),
	}

	fileContent, _, _, err := a.githubClient.Repositories.GetContents(ctx, owner, repo, path, opts)
	if err != nil {
		return "", err
	}

	if fileContent == nil {
		return "", fmt.Errorf("file content is nil")
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return "", err
	}

	return content, nil
}

// ParseCodeStructure parses code into AST (placeholder for Phase 2)
func ParseCodeStructure(content, language string) (*CodeAST, error) {
	// TODO: Implement AST parsing for different languages
	// For now, return a basic structure
	return &CodeAST{
		Language:   language,
		Functions:  []Function{},
		Classes:    []Class{},
		Imports:    []string{},
		Complexity: 0,
	}, nil
}

// CalculateComplexity calculates code complexity metrics (placeholder for Phase 2)
func CalculateComplexity(content string) (*Metrics, error) {
	// TODO: Implement actual complexity calculation
	// For now, return basic metrics
	lines := strings.Split(content, "\n")
	return &Metrics{
		LinesOfCode:          len(lines),
		CyclomaticComplexity: 0,
		MaintainabilityIndex: 0,
		CommentRatio:         0,
		TestCoverage:         0,
	}, nil
}

// detectLanguage detects the programming language from filename
func detectLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	languageMap := map[string]string{
		".go":   "go",
		".py":   "python",
		".js":   "javascript",
		".ts":   "typescript",
		".jsx":  "javascript",
		".tsx":  "typescript",
		".java": "java",
		".c":    "c",
		".cpp":  "cpp",
		".cs":   "csharp",
		".rb":   "ruby",
		".php":  "php",
		".rs":   "rust",
		".kt":   "kotlin",
		".swift": "swift",
	}

	if lang, ok := languageMap[ext]; ok {
		return lang
	}

	return "unknown"
}

// ShouldIgnoreFile checks if a file should be ignored based on patterns
func ShouldIgnoreFile(filename string, ignorePatterns []string) bool {
	for _, pattern := range ignorePatterns {
		// Simple pattern matching (can be enhanced with glob patterns)
		if strings.HasSuffix(pattern, "**") {
			// Directory pattern
			prefix := strings.TrimSuffix(pattern, "**")
			if strings.HasPrefix(filename, prefix) {
				return true
			}
		} else if strings.HasPrefix(pattern, "*.") {
			// Extension pattern
			ext := strings.TrimPrefix(pattern, "*")
			if strings.HasSuffix(filename, ext) {
				return true
			}
		} else if filename == pattern {
			// Exact match
			return true
		}
	}
	return false
}
