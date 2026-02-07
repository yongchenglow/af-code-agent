package mocks

import (
	"context"
	"encoding/json"
)

// MockAIResponse provides canned AI responses for testing
type MockAIResponse struct {
	Text  string
	Error error
}

// MockAIClient provides a mock AI client for testing
type MockAIClient struct {
	Responses map[string]*MockAIResponse
	Calls     []string
}

// NewMockAIClient creates a new mock AI client
func NewMockAIClient() *MockAIClient {
	return &MockAIClient{
		Responses: map[string]*MockAIResponse{
			"review": {
				Text: `{
					"issues": [
						{
							"severity": "High",
							"line": 10,
							"file_path": "test.go",
							"title": "Potential nil pointer dereference",
							"description": "Variable could be nil before access",
							"suggestion": "Add nil check before accessing"
						}
					]
				}`,
			},
			"security": {
				Text: `{
					"security_issues": [
						{
							"severity": "Critical",
							"line": 5,
							"file_path": "auth.go",
							"title": "Hardcoded credentials",
							"description": "API key is hardcoded in source",
							"suggestion": "Use environment variables"
						}
					],
					"count": 1
				}`,
			},
			"fix": {
				Text: `if user != nil {
	user.Name = "test"
}`,
			},
		},
		Calls: []string{},
	}
}

// Call mocks an AI call
func (m *MockAIClient) Call(ctx context.Context, promptType string, prompt string) (string, error) {
	m.Calls = append(m.Calls, promptType)

	if response, ok := m.Responses[promptType]; ok {
		if response.Error != nil {
			return "", response.Error
		}
		return response.Text, nil
	}

	// Default response
	return `{"status": "ok"}`, nil
}

// SetResponse sets a custom response for a prompt type
func (m *MockAIClient) SetResponse(promptType string, text string, err error) {
	m.Responses[promptType] = &MockAIResponse{
		Text:  text,
		Error: err,
	}
}

// GetCallCount returns the number of times the AI was called
func (m *MockAIClient) GetCallCount() int {
	return len(m.Calls)
}

// ResetCalls clears the call history
func (m *MockAIClient) ResetCalls() {
	m.Calls = []string{}
}

// MockReviewIssue represents a review issue for testing
type MockReviewIssue struct {
	Severity    string `json:"severity"`
	Line        int    `json:"line"`
	FilePath    string `json:"file_path"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

// MockReviewResponse represents a review response
type MockReviewResponse struct {
	Issues []MockReviewIssue `json:"issues"`
}

// ParseReviewResponse parses a review response
func ParseReviewResponse(text string) (*MockReviewResponse, error) {
	var response MockReviewResponse
	err := json.Unmarshal([]byte(text), &response)
	return &response, err
}
