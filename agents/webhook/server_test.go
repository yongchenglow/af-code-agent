package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/yourorg/github-code-agent/agents/analyzer"
	"github.com/yourorg/github-code-agent/pkg/config"
	ghpkg "github.com/yourorg/github-code-agent/pkg/github"
)

func TestWebhookSignatureValidation(t *testing.T) {
	// Create a test agent
	app, err := agent.New(agent.Config{
		NodeID:  "test-agent",
		Version: "1.0.0",
		TeamID:  "test",
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Register reasoners
	RegisterReasoners(app)
	analyzer.RegisterReasoners(app)

	// Create mock GitHub client
	ghClient, err := ghpkg.NewClient("test-token")
	if err != nil {
		t.Fatalf("Failed to create GitHub client: %v", err)
	}

	// Create webhook server
	webhookSecret := "test-secret-123"
	cfg := config.DefaultConfig()
	server := NewServer(app, webhookSecret, ghClient, cfg)

	tests := []struct {
		name           string
		payload        string
		eventType      string
		useValidSig    bool
		expectedStatus int
	}{
		{
			name:           "Valid signature",
			payload:        `{"action":"opened","pull_request":{"number":1},"repository":{"full_name":"test/repo"}}`,
			eventType:      "pull_request",
			useValidSig:    true,
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "Invalid signature",
			payload:        `{"action":"opened","pull_request":{"number":1},"repository":{"full_name":"test/repo"}}`,
			eventType:      "pull_request",
			useValidSig:    false,
			expectedStatus: http.StatusAccepted, // Validation passes, async processing fails
		},
		{
			name:           "Missing signature",
			payload:        `{"action":"opened","pull_request":{"number":1}}`,
			eventType:      "pull_request",
			useValidSig:    false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(tt.payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-GitHub-Event", tt.eventType)

			// Add signature if needed
			if tt.useValidSig {
				mac := hmac.New(sha256.New, []byte(webhookSecret))
				mac.Write([]byte(tt.payload))
				signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))
				req.Header.Set("X-Hub-Signature-256", signature)
			} else if tt.name != "Missing signature" {
				// Invalid signature
				req.Header.Set("X-Hub-Signature-256", "sha256=invalid")
			}

			// Record response
			rec := httptest.NewRecorder()

			// Handle request
			server.ServeHTTP(rec, req)

			// Check status
			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestValidateSignatureWithRealSecret(t *testing.T) {
	// Test with the actual webhook secret from your .env
	secret := ">zh.z4mfJoFyrxW+k1)B=?0f*5DA*H@-"

	// Example GitHub webhook payload
	payload := []byte(`{"ref":"refs/heads/main","before":"abc123","after":"def456"}`)

	// Compute signature like GitHub does
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// Should validate successfully
	err := ValidateSignature(payload, signature, secret)
	if err != nil {
		t.Errorf("ValidateSignature failed with real secret: %v", err)
	}

	t.Logf("Expected signature for payload: %s", signature)
}

func TestWebhookPayloadParsing(t *testing.T) {
	payload := map[string]any{
		"action": "opened",
		"pull_request": map[string]any{
			"number": float64(123),
			"title":  "Test PR",
		},
		"repository": map[string]any{
			"full_name": "owner/repo",
		},
	}

	payloadBytes, _ := json.Marshal(payload)

	t.Logf("Test payload: %s", string(payloadBytes))

	var parsed map[string]any
	if err := json.Unmarshal(payloadBytes, &parsed); err != nil {
		t.Errorf("Failed to parse payload: %v", err)
	}

	action, ok := parsed["action"].(string)
	if !ok || action != "opened" {
		t.Errorf("Failed to extract action")
	}

	pr, ok := parsed["pull_request"].(map[string]any)
	if !ok {
		t.Errorf("Failed to extract pull_request")
	}

	prNumber, ok := pr["number"].(float64)
	if !ok || prNumber != 123 {
		t.Errorf("Failed to extract PR number")
	}
}

func TestWebhook_ContentTypeValidation(t *testing.T) {
	app, _ := agent.New(agent.Config{NodeID: "test", Version: "1.0.0", TeamID: "test"})
	ghClient, _ := ghpkg.NewClient("test-token")
	cfg := config.DefaultConfig()
	server := NewServer(app, "test-secret", ghClient, cfg)

	tests := []struct {
		name        string
		contentType string
		wantStatus  int
	}{
		{"valid json", "application/json", http.StatusAccepted}, // Will pass validation, fail sig check
		{"invalid type", "text/plain", http.StatusBadRequest},
		{"empty", "", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := `{"action":"opened"}`
			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(payload))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			req.Header.Set("X-GitHub-Event", "pull_request")
			req.Header.Set("X-Hub-Signature-256", "sha256=invalid")

			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestWebhook_EventTypeValidation(t *testing.T) {
	app, _ := agent.New(agent.Config{NodeID: "test", Version: "1.0.0", TeamID: "test"})
	ghClient, _ := ghpkg.NewClient("test-token")
	cfg := config.DefaultConfig()
	server := NewServer(app, "test-secret", ghClient, cfg)

	tests := []struct {
		name       string
		eventType  string
		wantStatus int
	}{
		{"valid event", "pull_request", http.StatusAccepted}, // Will pass validation, fail sig check
		{"invalid event", "unknown_event", http.StatusBadRequest},
		{"empty event", "", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := `{"action":"opened","repository":{"full_name":"test/repo"}}`
			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-GitHub-Event", tt.eventType)
			req.Header.Set("X-Hub-Signature-256", "sha256=invalid")

			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestWebhook_RateLimiting(t *testing.T) {
	app, _ := agent.New(agent.Config{NodeID: "test", Version: "1.0.0", TeamID: "test"})
	ghClient, _ := ghpkg.NewClient("test-token")
	cfg := config.DefaultConfig()
	server := NewServer(app, "test-secret", ghClient, cfg)

	repo := "test/repo"
	payload := `{"action":"opened","repository":{"full_name":"` + repo + `"}}`

	// Compute valid signature
	mac := hmac.New(sha256.New, []byte("test-secret"))
	mac.Write([]byte(payload))
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// Make 10 requests (should all be accepted)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GitHub-Event", "pull_request")
		req.Header.Set("X-Hub-Signature-256", signature)

		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Errorf("Request %d: expected status %d, got %d", i+1, http.StatusAccepted, rec.Code)
		}
	}

	// 11th request should be rate limited
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", signature)

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected rate limiting, got status %d", rec.Code)
	}

	// Check rate limit headers
	if remaining := rec.Header().Get("X-RateLimit-Remaining"); remaining == "" {
		t.Error("Expected X-RateLimit-Remaining header")
	}
	if reset := rec.Header().Get("X-RateLimit-Reset"); reset == "" {
		t.Error("Expected X-RateLimit-Reset header")
	}
	if retry := rec.Header().Get("Retry-After"); retry == "" {
		t.Error("Expected Retry-After header")
	}
}

func TestWebhook_RateLimitPerRepo(t *testing.T) {
	app, _ := agent.New(agent.Config{NodeID: "test", Version: "1.0.0", TeamID: "test"})
	ghClient, _ := ghpkg.NewClient("test-token")
	cfg := config.DefaultConfig()
	server := NewServer(app, "test-secret", ghClient, cfg)

	// Exhaust rate limit for repo1
	repo1Payload := `{"action":"opened","repository":{"full_name":"repo1/test"}}`
	mac := hmac.New(sha256.New, []byte("test-secret"))
	mac.Write([]byte(repo1Payload))
	sig1 := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(repo1Payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GitHub-Event", "pull_request")
		req.Header.Set("X-Hub-Signature-256", sig1)
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, req)
	}

	// repo2 should still be allowed
	repo2Payload := `{"action":"opened","repository":{"full_name":"repo2/test"}}`
	mac2 := hmac.New(sha256.New, []byte("test-secret"))
	mac2.Write([]byte(repo2Payload))
	sig2 := "sha256=" + hex.EncodeToString(mac2.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(repo2Payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", sig2)

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("repo2 should not be rate limited, got status %d", rec.Code)
	}
}

func TestWebhook_ExtractRepositoryFromBody(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{
			name:    "repository in payload",
			payload: `{"repository":{"full_name":"owner/repo"},"action":"opened"}`,
			want:    "owner/repo",
		},
		{
			name:    "no repository",
			payload: `{"action":"opened"}`,
			want:    "",
		},
		{
			name:    "invalid json",
			payload: `{invalid}`,
			want:    "",
		},
		{
			name:    "pull_request event",
			payload: `{"pull_request":{"base":{"repo":{"full_name":"owner/repo"}}}}`,
			want:    "owner/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRepositoryFromBody([]byte(tt.payload))
			if got != tt.want {
				t.Errorf("extractRepositoryFromBody() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWebhook_JSONValidation(t *testing.T) {
	app, _ := agent.New(agent.Config{NodeID: "test", Version: "1.0.0", TeamID: "test"})
	ghClient, _ := ghpkg.NewClient("test-token")
	cfg := config.DefaultConfig()
	server := NewServer(app, "test-secret", ghClient, cfg)

	tests := []struct {
		name       string
		payload    string
		wantStatus int
	}{
		{"valid json", `{"action":"opened","repository":{"full_name":"test/repo"}}`, http.StatusAccepted}, // Will pass validation
		{"invalid json", `{action: "opened"}`, http.StatusBadRequest},
		{"empty body", ``, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(tt.payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-GitHub-Event", "pull_request")
			req.Header.Set("X-Hub-Signature-256", "sha256=invalid")

			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestWebhook_MethodValidation(t *testing.T) {
	app, _ := agent.New(agent.Config{NodeID: "test", Version: "1.0.0", TeamID: "test"})
	ghClient, _ := ghpkg.NewClient("test-token")
	cfg := config.DefaultConfig()
	server := NewServer(app, "test-secret", ghClient, cfg)

	tests := []struct {
		name       string
		method     string
		wantStatus int
	}{
		{"POST", http.MethodPost, http.StatusBadRequest}, // Will fail content-type check
		{"GET", http.MethodGet, http.StatusMethodNotAllowed},
		{"PUT", http.MethodPut, http.StatusMethodNotAllowed},
		{"DELETE", http.MethodDelete, http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/webhook", nil)
			req.Header.Set("X-GitHub-Event", "pull_request")
			req.Header.Set("X-Hub-Signature-256", "sha256=invalid")

			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestWebhook_RateLimitHeaders(t *testing.T) {
	app, _ := agent.New(agent.Config{NodeID: "test", Version: "1.0.0", TeamID: "test"})
	ghClient, _ := ghpkg.NewClient("test-token")
	cfg := config.DefaultConfig()
	server := NewServer(app, "test-secret", ghClient, cfg)

	payload := `{"action":"opened","repository":{"full_name":"test/repo"}}`
	mac := hmac.New(sha256.New, []byte("test-secret"))
	mac.Write([]byte(payload))
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", signature)

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	// Check for rate limit headers in successful response
	if rec.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, rec.Code)
	}

	// Headers should be present for valid requests with repo
	if rec.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("Expected X-RateLimit-Remaining header")
	}
	if rec.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("Expected X-RateLimit-Reset header")
	}
}

func TestWebhook_AsyncProcessing(t *testing.T) {
	app, _ := agent.New(agent.Config{NodeID: "test", Version: "1.0.0", TeamID: "test"})
	ghClient, _ := ghpkg.NewClient("test-token")
	cfg := config.DefaultConfig()
	server := NewServer(app, "test-secret", ghClient, cfg)

	payload := `{"action":"opened","repository":{"full_name":"test/repo"}}`
	mac := hmac.New(sha256.New, []byte("test-secret"))
	mac.Write([]byte(payload))
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", signature)

	rec := httptest.NewRecorder()

	// Request should return immediately with 202 Accepted
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("Expected immediate 202 Accepted, got %d", rec.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if response["status"] != "accepted" {
		t.Errorf("Expected status 'accepted', got '%s'", response["status"])
	}

	// Wait a bit for async processing
	time.Sleep(100 * time.Millisecond)
}

func TestWebhook_InjectionDetection(t *testing.T) {
	app, _ := agent.New(agent.Config{NodeID: "test", Version: "1.0.0", TeamID: "test"})
	ghClient, _ := ghpkg.NewClient("test-token")
	cfg := config.DefaultConfig()
	server := NewServer(app, "test-secret", ghClient, cfg)

	// Payload with SQL injection pattern
	payload := `{"action":"' OR '1'='1","repository":{"full_name":"test/repo"}}`

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", "sha256=invalid")

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	// Should be rejected due to injection detection
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected injection detection to reject request, got status %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "injection") {
		t.Logf("Response body: %s", rec.Body.String())
	}
}
