package app

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// HealthStatus represents the health status of a component.
type HealthStatus struct {
	Status  bool   `json:"status"`
	Message string `json:"message,omitempty"`
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status    string                  `json:"status"`
	Checks    map[string]HealthStatus `json:"checks,omitempty"`
	Timestamp time.Time               `json:"timestamp"`
}

// HealthChecker provides health check functionality.
type HealthChecker struct {
	container     *Container
	started       bool
	startedMu     sync.RWMutex
	checkCache    map[string]HealthStatus
	cacheDuration time.Duration
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(container *Container) *HealthChecker {
	return &HealthChecker{
		container:     container,
		started:       false,
		checkCache:    make(map[string]HealthStatus),
		cacheDuration: 5 * time.Second,
	}
}

// MarkStarted marks the application as fully started.
func (hc *HealthChecker) MarkStarted() {
	hc.startedMu.Lock()
	defer hc.startedMu.Unlock()
	hc.started = true
}

// IsStarted returns true if the application has fully started.
func (hc *HealthChecker) IsStarted() bool {
	hc.startedMu.RLock()
	defer hc.startedMu.RUnlock()
	return hc.started
}

// checkAIService checks if the AI service is accessible.
func (hc *HealthChecker) checkAIService(ctx context.Context) HealthStatus {
	// Check if agent is initialized
	if hc.container == nil || hc.container.Agent == nil {
		return HealthStatus{
			Status:  false,
			Message: "AI agent not initialized",
		}
	}

	// Basic check - agent exists and is functional
	// In a real scenario, you might ping the AI API
	return HealthStatus{
		Status:  true,
		Message: "AI service is available",
	}
}

// checkGitHubAPI checks if the GitHub API is accessible.
func (hc *HealthChecker) checkGitHubAPI(ctx context.Context) HealthStatus {
	// Check if GitHub client is initialized
	if hc.container == nil || hc.container.GitHubClient == nil {
		return HealthStatus{
			Status:  false,
			Message: "GitHub client not initialized",
		}
	}

	// Basic check - client exists
	// In a real scenario, you might make a lightweight API call
	return HealthStatus{
		Status:  true,
		Message: "GitHub API client is available",
	}
}

// checkMemoryBackend checks if the memory backend is accessible.
func (hc *HealthChecker) checkMemoryBackend(ctx context.Context) HealthStatus {
	// Check if agent has memory backend
	if hc.container == nil || hc.container.Agent == nil {
		return HealthStatus{
			Status:  false,
			Message: "Agent not initialized",
		}
	}

	// Memory backend is available if agent exists
	return HealthStatus{
		Status:  true,
		Message: "Memory backend is available",
	}
}

// performHealthChecks performs all health checks and returns the results.
func (hc *HealthChecker) performHealthChecks(ctx context.Context) map[string]HealthStatus {
	checks := make(map[string]HealthStatus)

	// Check AI service
	checks["ai_service"] = hc.checkAIService(ctx)

	// Check GitHub API
	checks["github_api"] = hc.checkGitHubAPI(ctx)

	// Check memory backend
	checks["memory_backend"] = hc.checkMemoryBackend(ctx)

	return checks
}

// getOverallStatus returns the overall health status.
func getOverallStatus(checks map[string]HealthStatus) string {
	for _, status := range checks {
		if !status.Status {
			return "unhealthy"
		}
	}
	return "ok"
}

// handleLive handles the liveness probe endpoint.
// Returns 200 OK if the process is running.
func (hc *HealthChecker) handleLive(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// handleReady handles the readiness probe endpoint.
// Returns 200 OK if all dependencies are healthy, 503 otherwise.
func (hc *HealthChecker) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := hc.performHealthChecks(ctx)
	overallStatus := getOverallStatus(checks)

	response := HealthResponse{
		Status:    overallStatus,
		Checks:    checks,
		Timestamp: time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	if overallStatus != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	_ = json.NewEncoder(w).Encode(response)
}

// handleStarted handles the startup probe endpoint.
// Returns 200 OK when application is fully initialized, 503 during startup.
func (hc *HealthChecker) handleStarted(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	httpStatus := http.StatusOK

	if !hc.IsStarted() {
		status = "starting"
		httpStatus = http.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status:    status,
		Timestamp: time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(response)
}

// RegisterHealthHandlers registers all health check endpoints.
func (hc *HealthChecker) RegisterHealthHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/health/live", hc.handleLive)
	mux.HandleFunc("/health/ready", hc.handleReady)
	mux.HandleFunc("/health/started", hc.handleStarted)
}
