package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/yourorg/github-code-agent/features/analyzer"
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

	// Create webhook server
	webhookSecret := "test-secret-123"
	server := NewServer(app, webhookSecret)

	tests := []struct {
		name           string
		payload        string
		eventType      string
		useValidSig    bool
		expectedStatus int
	}{
		{
			name:           "Valid signature",
			payload:        `{"action":"opened","pull_request":{"number":1}}`,
			eventType:      "pull_request",
			useValidSig:    true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid signature",
			payload:        `{"action":"opened","pull_request":{"number":1}}`,
			eventType:      "pull_request",
			useValidSig:    false,
			expectedStatus: http.StatusUnauthorized,
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
