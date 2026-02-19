package analyzer

// FileChange represents a changed file in a pull request
type FileChange struct {
	Filename  string
	Status    string // added, modified, deleted
	Additions int
	Deletions int
	Changes   int
	Patch     string
	Content   string
	Language  string
	BlobURL   string
}

// CodeAST represents parsed code structure
type CodeAST struct {
	Language   string
	Functions  []Function
	Classes    []Class
	Imports    []string
	Complexity int
}

// Function represents a function in code
type Function struct {
	Name       string
	LineStart  int
	LineEnd    int
	Complexity int
	Parameters []string
	ReturnType string
}

// Class represents a class in code
type Class struct {
	Name      string
	LineStart int
	LineEnd   int
	Methods   []Function
}

// Metrics contains code quality metrics
type Metrics struct {
	LinesOfCode          int
	CyclomaticComplexity int
	MaintainabilityIndex float64
	CommentRatio         float64
	TestCoverage         float64
}

// AnalysisResult contains the complete analysis of a PR
type AnalysisResult struct {
	Files   []*FileChange
	Metrics *Metrics
	Summary string
}
