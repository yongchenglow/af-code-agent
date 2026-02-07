package deepseek

import (
	"context"
	"fmt"
)

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a request to the chat API
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// ChatResponse represents a response from the chat API
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message Message `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Client wraps the DeepSeek API client
type Client struct {
	apiKey  string
	baseURL string
	model   string
	temperature float64
	maxTokens int
}

// NewClient creates a new DeepSeek client
func NewClient(apiKey, baseURL, model string, temperature float64, maxTokens int) *Client {
	return &Client{
		apiKey:      apiKey,
		baseURL:     baseURL,
		model:       model,
		temperature: temperature,
		maxTokens:   maxTokens,
	}
}

// Chat sends a chat request to DeepSeek
func (c *Client) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// TODO: Implement actual API call to DeepSeek
	// For now, this is a placeholder that will be implemented with the actual HTTP client
	// when we integrate with DeepSeek API in later phases

	return nil, fmt.Errorf("DeepSeek integration not yet implemented - Phase 2")
}

// GenerateReview generates a code review using DeepSeek
func (c *Client) GenerateReview(ctx context.Context, code, language string) (string, error) {
	req := &ChatRequest{
		Model: c.model,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an expert code reviewer focusing on security, performance, and best practices.",
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("Review this %s code:\n\n%s", language, code),
			},
		},
		Temperature: c.temperature,
		MaxTokens:   c.maxTokens,
	}

	resp, err := c.Chat(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from DeepSeek")
	}

	return resp.Choices[0].Message.Content, nil
}

// GenerateFix generates a code fix using DeepSeek
func (c *Client) GenerateFix(ctx context.Context, issue, code string, previousErrors []string) (string, error) {
	prompt := fmt.Sprintf("Generate a minimal fix for this issue:\n\n%s\n\nCode:\n%s", issue, code)

	if len(previousErrors) > 0 {
		prompt += "\n\nPrevious fix attempts had these issues:\n"
		for _, err := range previousErrors {
			prompt += fmt.Sprintf("- %s\n", err)
		}
		prompt += "\nPlease generate a fix that avoids these problems."
	}

	req := &ChatRequest{
		Model: c.model,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an expert code fixer. Generate minimal, targeted fixes that pass all validation checks (syntax, linting, formatting).",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.1, // Lower temperature for deterministic fixes
		MaxTokens:   c.maxTokens,
	}

	resp, err := c.Chat(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from DeepSeek")
	}

	return resp.Choices[0].Message.Content, nil
}
