package validator

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidator_ValidateContentType(t *testing.T) {
	validator := NewValidator(ValidatorConfig{})

	tests := []struct {
		name        string
		contentType string
		wantErr     bool
	}{
		{
			name:        "valid json content type",
			contentType: "application/json",
			wantErr:     false,
		},
		{
			name:        "valid json with charset",
			contentType: "application/json; charset=utf-8",
			wantErr:     false,
		},
		{
			name:        "invalid content type",
			contentType: "text/plain",
			wantErr:     true,
		},
		{
			name:        "missing content type",
			contentType: "",
			wantErr:     true,
		},
		{
			name:        "html content type",
			contentType: "text/html",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte(`{}`)))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			err := validator.validateContentType(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateContentType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidatePayloadSize(t *testing.T) {
	validator := NewValidator(ValidatorConfig{
		MaxPayloadSize: 1024, // 1KB for testing
	})

	tests := []struct {
		name    string
		size    int
		wantErr bool
	}{
		{
			name:    "small payload",
			size:    100,
			wantErr: false,
		},
		{
			name:    "exact limit",
			size:    1024,
			wantErr: false,
		},
		{
			name:    "exceeds limit",
			size:    1025,
			wantErr: true,
		},
		{
			name:    "large payload",
			size:    10240,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := make([]byte, tt.size)
			err := validator.validatePayloadSize(body)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePayloadSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateEventType(t *testing.T) {
	validator := NewValidator(ValidatorConfig{
		AllowedEvents: []string{"pull_request", "push", "issue_comment"},
	})

	tests := []struct {
		name      string
		eventType string
		wantErr   bool
	}{
		{
			name:      "valid pull_request event",
			eventType: "pull_request",
			wantErr:   false,
		},
		{
			name:      "valid push event",
			eventType: "push",
			wantErr:   false,
		},
		{
			name:      "valid issue_comment event",
			eventType: "issue_comment",
			wantErr:   false,
		},
		{
			name:      "invalid event type",
			eventType: "unknown_event",
			wantErr:   true,
		},
		{
			name:      "empty event type",
			eventType: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateEventType(tt.eventType)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEventType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateJSON(t *testing.T) {
	validator := NewValidator(ValidatorConfig{})

	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "valid json object",
			body:    `{"key": "value"}`,
			wantErr: false,
		},
		{
			name:    "valid json array",
			body:    `[1, 2, 3]`,
			wantErr: false,
		},
		{
			name:    "invalid json",
			body:    `{key: "value"}`,
			wantErr: true,
		},
		{
			name:    "empty body",
			body:    ``,
			wantErr: true,
		},
		{
			name:    "just whitespace",
			body:    `   `,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateJSON([]byte(tt.body))
			if (err != nil) != tt.wantErr {
				t.Errorf("validateJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateInjection(t *testing.T) {
	validator := NewValidator(ValidatorConfig{})

	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "normal payload",
			body:    `{"action": "opened", "number": 123}`,
			wantErr: false,
		},
		{
			name:    "SQL injection OR",
			body:    `{"action": "' OR '1'='1"}`,
			wantErr: true,
		},
		{
			name:    "SQL comment injection",
			body:    `{"action": "test"; --}`,
			wantErr: false, // Only flag at end of line
		},
		{
			name:    "DROP TABLE injection",
			body:    `{"action": "test"; DROP TABLE users}`,
			wantErr: true,
		},
		{
			name:    "NoSQL where injection",
			body:    `{"$where": "this.value > 0"}`,
			wantErr: true,
		},
		{
			name:    "NoSQL ne injection",
			body:    `{"password": {"$ne": null}}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateInjection([]byte(tt.body))
			if (err != nil) != tt.wantErr {
				t.Errorf("validateInjection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidatePRNumber(t *testing.T) {
	validator := NewValidator(ValidatorConfig{})

	tests := []struct {
		name    string
		prNum   int
		wantErr bool
	}{
		{
			name:    "valid PR number",
			prNum:   123,
			wantErr: false,
		},
		{
			name:    "PR number 1",
			prNum:   1,
			wantErr: false,
		},
		{
			name:    "zero PR number",
			prNum:   0,
			wantErr: true,
		},
		{
			name:    "negative PR number",
			prNum:   -5,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePRNumber(tt.prNum)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePRNumber() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_SanitizeFilePath(t *testing.T) {
	validator := NewValidator(ValidatorConfig{})

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name:    "normal path",
			path:    "src/main.go",
			want:    "src/main.go",
			wantErr: false,
		},
		{
			name:    "nested path",
			path:    "pkg/validator/validator.go",
			want:    "pkg/validator/validator.go",
			wantErr: false,
		},
		{
			name:    "path traversal",
			path:    "../etc/passwd",
			want:    "",
			wantErr: true,
		},
		{
			name:    "path traversal in middle",
			path:    "src/../../../etc/passwd",
			want:    "",
			wantErr: true,
		},
		{
			name:    "absolute path",
			path:    "/etc/passwd",
			want:    "",
			wantErr: true,
		},
		{
			name:    "windows path traversal",
			path:    "..\\..\\windows\\system32",
			want:    "",
			wantErr: true,
		},
		{
			name:    "path with null byte",
			path:    "src/main.go\x00.jpg",
			want:    "src/main.go.jpg",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validator.SanitizeFilePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("SanitizeFilePath() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("SanitizeFilePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidator_ValidateRepository(t *testing.T) {
	validator := NewValidator(ValidatorConfig{})

	tests := []struct {
		name    string
		owner   string
		repo    string
		wantErr bool
	}{
		{
			name:    "valid repository",
			owner:   "myorg",
			repo:    "my-repo",
			wantErr: false,
		},
		{
			name:    "repository with underscores",
			owner:   "my_org",
			repo:    "my_repo",
			wantErr: false,
		},
		{
			name:    "repository with dots",
			owner:   "org.name",
			repo:    "repo.name",
			wantErr: false,
		},
		{
			name:    "empty owner",
			owner:   "",
			repo:    "my-repo",
			wantErr: true,
		},
		{
			name:    "empty repo",
			owner:   "myorg",
			repo:    "",
			wantErr: true,
		},
		{
			name:    "invalid owner with special chars",
			owner:   "my@org!",
			repo:    "my-repo",
			wantErr: true,
		},
		{
			name:    "name too long",
			owner:   "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789",
			repo:    "my-repo",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateRepository(tt.owner, tt.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRepository() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ValidateRequest(t *testing.T) {
	validator := NewValidator(ValidatorConfig{
		AllowedEvents:  []string{"pull_request", "push"},
		MaxPayloadSize: 1024,
	})

	tests := []struct {
		name        string
		contentType string
		eventType   string
		body        string
		wantErr     bool
	}{
		{
			name:        "valid request",
			contentType: "application/json",
			eventType:   "pull_request",
			body:        `{"action": "opened", "number": 123}`,
			wantErr:     false,
		},
		{
			name:        "invalid content type",
			contentType: "text/plain",
			eventType:   "pull_request",
			body:        `{"action": "opened"}`,
			wantErr:     true,
		},
		{
			name:        "invalid event type",
			contentType: "application/json",
			eventType:   "unknown",
			body:        `{"action": "opened"}`,
			wantErr:     true,
		},
		{
			name:        "invalid JSON",
			contentType: "application/json",
			eventType:   "pull_request",
			body:        `{action: "opened"}`,
			wantErr:     true,
		},
		{
			name:        "payload too large",
			contentType: "application/json",
			eventType:   "pull_request",
			body:        string(make([]byte, 2000)),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", tt.contentType)
			req.Header.Set("X-GitHub-Event", tt.eventType)

			err := validator.ValidateRequest(req, []byte(tt.body), tt.eventType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	limiter := NewRateLimiter(RateLimiterConfig{
		MaxRequests: 3,
		Window:      time.Second,
	})

	repo := "test/repo"

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		if !limiter.Allow(repo) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 4th request should be denied
	if limiter.Allow(repo) {
		t.Error("Request 4 should be denied")
	}

	// Different repo should be allowed
	if !limiter.Allow("other/repo") {
		t.Error("Different repo should be allowed")
	}
}

func TestRateLimiter_GetRemaining(t *testing.T) {
	limiter := NewRateLimiter(RateLimiterConfig{
		MaxRequests: 5,
		Window:      time.Second,
	})

	repo := "test/repo"

	// Initially should have 5 remaining
	if remaining := limiter.GetRemaining(repo); remaining != 5 {
		t.Errorf("GetRemaining() = %d, want 5", remaining)
	}

	// Make 2 requests
	limiter.Allow(repo)
	limiter.Allow(repo)

	// Should have 3 remaining
	if remaining := limiter.GetRemaining(repo); remaining != 3 {
		t.Errorf("GetRemaining() after 2 requests = %d, want 3", remaining)
	}
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	limiter := NewRateLimiter(RateLimiterConfig{
		MaxRequests:     2,
		Window:          100 * time.Millisecond,
		CleanupInterval: 50 * time.Millisecond,
	})

	repo := "test/repo"

	// Use up all requests
	limiter.Allow(repo)
	limiter.Allow(repo)

	// Should be denied
	if limiter.Allow(repo) {
		t.Error("Should be denied after limit reached")
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	if !limiter.Allow(repo) {
		t.Error("Should be allowed after window expires")
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	limiter := NewRateLimiter(RateLimiterConfig{
		MaxRequests: 100,
		Window:      time.Second,
	})

	repo := "test/repo"
	done := make(chan bool, 10)

	// Start 10 goroutines making requests
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				limiter.Allow(repo)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to finish
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have used all 100 requests
	if remaining := limiter.GetRemaining(repo); remaining != 0 {
		t.Errorf("GetRemaining() = %d, want 0", remaining)
	}
}
