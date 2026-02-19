package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/yourorg/github-code-agent/pkg/config"
)

func createTestContainer() *Container {
	cfg := config.DefaultConfig()
	envCfg := &config.EnvironmentConfig{
		Port:                "8080",
		AgentFieldURL:       "http://localhost:8080",
		AgentFieldToken:     "test-token",
		GitHubAppID:         "123456",
		GitHubPrivateKey:    "test-key",
		GitHubWebhookSecret: "test-secret",
		AIModel:             "test-model",
		AITemperature:       0.2,
		AIMaxTokens:         4000,
		LogLevel:            "info",
	}

	// Create a mock agent (we won't actually connect to anything)
	mockAgent := &agent.Agent{}

	// For testing, we create a container with nil GitHub client
	// The health checks will handle nil gracefully
	container := &Container{
		Config:       cfg,
		EnvConfig:    envCfg,
		Agent:        mockAgent,
		GitHubClient: nil, // Intentionally nil for testing
	}
	return container
}

func TestHealthCheckerNew(t *testing.T) {
	container := createTestContainer()
	hc := NewHealthChecker(container)

	if hc == nil {
		t.Fatal("NewHealthChecker returned nil")
	}

	if hc.container != container {
		t.Error("HealthChecker container not set correctly")
	}

	if hc.IsStarted() {
		t.Error("HealthChecker should not be started initially")
	}
}

func TestHealthCheckerMarkStarted(t *testing.T) {
	container := createTestContainer()
	hc := NewHealthChecker(container)

	if hc.IsStarted() {
		t.Error("Should not be started before MarkStarted")
	}

	hc.MarkStarted()

	if !hc.IsStarted() {
		t.Error("Should be started after MarkStarted")
	}
}

func TestHandleLive(t *testing.T) {
	container := createTestContainer()
	hc := NewHealthChecker(container)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	hc.handleLive(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var response HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response.Status)
	}

	if response.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestHandleReadyWithNilDependencies(t *testing.T) {
	container := createTestContainer()
	hc := NewHealthChecker(container)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	hc.handleReady(w, req)

	resp := w.Result()
	// Should return 503 because GitHub client is nil
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", resp.StatusCode)
	}

	var response HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got '%s'", response.Status)
	}

	// Check that github_api check failed
	if ghCheck, ok := response.Checks["github_api"]; ok {
		if ghCheck.Status {
			t.Error("GitHub API check should fail with nil client")
		}
	} else {
		t.Error("Expected github_api check in response")
	}
}

func TestHandleStarted(t *testing.T) {
	container := createTestContainer()
	hc := NewHealthChecker(container)

	// Test before started
	req := httptest.NewRequest(http.MethodGet, "/health/started", nil)
	w := httptest.NewRecorder()

	hc.handleStarted(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 before started, got %d", resp.StatusCode)
	}

	var response HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "starting" {
		t.Errorf("Expected status 'starting', got '%s'", response.Status)
	}

	// Test after started
	hc.MarkStarted()

	req = httptest.NewRequest(http.MethodGet, "/health/started", nil)
	w = httptest.NewRecorder()

	hc.handleStarted(w, req)

	resp = w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 after started, got %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response.Status)
	}
}

func TestGetOverallStatus(t *testing.T) {
	tests := []struct {
		name     string
		checks   map[string]HealthStatus
		expected string
	}{
		{
			name: "all healthy",
			checks: map[string]HealthStatus{
				"ai_service":     {Status: true, Message: "ok"},
				"github_api":     {Status: true, Message: "ok"},
				"memory_backend": {Status: true, Message: "ok"},
			},
			expected: "ok",
		},
		{
			name: "one unhealthy",
			checks: map[string]HealthStatus{
				"ai_service":     {Status: true, Message: "ok"},
				"github_api":     {Status: false, Message: "error"},
				"memory_backend": {Status: true, Message: "ok"},
			},
			expected: "unhealthy",
		},
		{
			name: "all unhealthy",
			checks: map[string]HealthStatus{
				"ai_service":     {Status: false, Message: "error"},
				"github_api":     {Status: false, Message: "error"},
				"memory_backend": {Status: false, Message: "error"},
			},
			expected: "unhealthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getOverallStatus(tt.checks)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestCheckAIService(t *testing.T) {
	container := createTestContainer()
	hc := NewHealthChecker(container)

	status := hc.checkAIService(context.Background())

	if !status.Status {
		t.Errorf("AI service should be healthy: %s", status.Message)
	}
}

func TestCheckGitHubAPIWithNilClient(t *testing.T) {
	container := createTestContainer()
	hc := NewHealthChecker(container)

	status := hc.checkGitHubAPI(context.Background())

	if status.Status {
		t.Errorf("GitHub API should be unhealthy with nil client: %s", status.Message)
	}

	if status.Message != "GitHub client not initialized" {
		t.Errorf("Expected 'GitHub client not initialized' message, got '%s'", status.Message)
	}
}

func TestCheckMemoryBackend(t *testing.T) {
	container := createTestContainer()
	hc := NewHealthChecker(container)

	status := hc.checkMemoryBackend(context.Background())

	if !status.Status {
		t.Errorf("Memory backend should be healthy: %s", status.Message)
	}
}

func TestPerformHealthChecks(t *testing.T) {
	container := createTestContainer()
	hc := NewHealthChecker(container)

	checks := hc.performHealthChecks(context.Background())

	expectedChecks := []string{"ai_service", "github_api", "memory_backend"}
	for _, checkName := range expectedChecks {
		check, ok := checks[checkName]
		if !ok {
			t.Errorf("Expected check '%s' in results", checkName)
			continue
		}

		// AI and memory should be healthy, GitHub should be unhealthy
		if checkName == "github_api" {
			if check.Status {
				t.Errorf("Check '%s' should be unhealthy with nil client", checkName)
			}
		} else {
			if !check.Status {
				t.Errorf("Check '%s' should be healthy: %s", checkName, check.Message)
			}
		}
	}
}

func TestHealthCheckerConcurrentAccess(t *testing.T) {
	container := createTestContainer()
	hc := NewHealthChecker(container)

	// Start multiple goroutines that call MarkStarted
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			hc.MarkStarted()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should only be marked once
	if !hc.IsStarted() {
		t.Error("HealthChecker should be started")
	}
}

func TestHealthResponseJSON(t *testing.T) {
	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Date(2026, 2, 19, 10, 30, 0, 0, time.UTC),
		Checks: map[string]HealthStatus{
			"ai_service": {Status: true, Message: "ok"},
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var decoded HealthResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if decoded.Status != response.Status {
		t.Errorf("Status mismatch: expected %s, got %s", response.Status, decoded.Status)
	}

	if !decoded.Timestamp.Equal(response.Timestamp) {
		t.Errorf("Timestamp mismatch")
	}

	if len(decoded.Checks) != len(response.Checks) {
		t.Errorf("Checks length mismatch: expected %d, got %d", len(response.Checks), len(decoded.Checks))
	}
}

func TestHealthCheckerNilContainer(t *testing.T) {
	hc := &HealthChecker{
		container:  nil,
		started:    false,
		checkCache: make(map[string]HealthStatus),
	}

	// Should not panic with nil container
	status := hc.checkAIService(context.Background())
	if status.Status {
		t.Error("AI service should be unhealthy with nil container")
	}

	status = hc.checkGitHubAPI(context.Background())
	if status.Status {
		t.Error("GitHub API should be unhealthy with nil container")
	}

	status = hc.checkMemoryBackend(context.Background())
	if status.Status {
		t.Error("Memory backend should be unhealthy with nil container")
	}
}
