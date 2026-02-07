package webhook

import (
	"testing"

	"github.com/yourorg/github-code-agent/pkg/config"
)

func TestValidateSignature(t *testing.T) {
	secret := "test-secret"
	payload := []byte(`{"test":"data"}`)

	tests := []struct {
		name      string
		signature string
		wantErr   bool
	}{
		{
			name:      "missing sha256 prefix",
			signature: "invalid",
			wantErr:   true,
		},
		{
			name:      "invalid format",
			signature: "sha256=",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSignature(payload, tt.signature, secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSignature() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestShouldProcessEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    string
		action   string
		triggers []string
		want     bool
	}{
		{
			name:     "pull request opened",
			event:    "pull_request",
			action:   "opened",
			triggers: []string{"pull_request.opened"},
			want:     true,
		},
		{
			name:     "pull request closed not configured",
			event:    "pull_request",
			action:   "closed",
			triggers: []string{"pull_request.opened"},
			want:     false,
		},
		{
			name:     "check suite completed",
			event:    "check_suite",
			action:   "completed",
			triggers: []string{"check_suite.completed"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldProcessEvent(tt.event, tt.action, tt.triggers)
			if got != tt.want {
				t.Errorf("ShouldProcessEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandlerDebounce(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Webhooks.DebounceSeconds = 1

	handler := NewHandler(cfg, "test-secret")

	event := &WebhookEvent{
		Type:   "pull_request",
		Action: "synchronize",
		PullRequest: &PullRequestPayload{
			Number: 123,
		},
	}

	// First call should not debounce
	if handler.shouldDebounce(event) {
		t.Error("First event should not be debounced")
	}

	// Second call within timeout should debounce
	if !handler.shouldDebounce(event) {
		t.Error("Second event within timeout should be debounced")
	}
}
