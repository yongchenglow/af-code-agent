package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name:   "default config",
			config: DefaultConfig(),
		},
		{
			name: "json format",
			config: Config{
				Level:  LevelInfo,
				Format: FormatJSON,
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "text format",
			config: Config{
				Level:  LevelDebug,
				Format: FormatText,
				Output: &bytes.Buffer{},
			},
		},
		{
			name: "error level only",
			config: Config{
				Level:  LevelError,
				Format: FormatText,
				Output: &bytes.Buffer{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.config)
			if l == nil {
				t.Fatal("expected logger, got nil")
			}
			if l.Logger == nil {
				t.Fatal("expected slog.Logger, got nil")
			}
		})
	}
}

func TestLoggerLevels(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := Config{
		Level:  LevelDebug,
		Format: FormatText,
		Output: buf,
	}

	l := New(cfg)

	tests := []struct {
		name     string
		logFunc  func()
		expected string
	}{
		{
			name: "debug",
			logFunc: func() {
				l.Debug("debug message", "key", "value")
			},
			expected: "debug message",
		},
		{
			name: "info",
			logFunc: func() {
				l.Info("info message", "key", "value")
			},
			expected: "info message",
		},
		{
			name: "warn",
			logFunc: func() {
				l.Warn("warn message", "key", "value")
			},
			expected: "warn message",
		},
		{
			name: "error",
			logFunc: func() {
				l.Error("error message", "key", "value")
			},
			expected: "error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()
			output := buf.String()
			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected output to contain %q, got %q", tt.expected, output)
			}
		})
	}
}

func TestLoggerWith(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := Config{
		Level:  LevelInfo,
		Format: FormatText,
		Output: buf,
	}

	l := New(cfg)
	l2 := l.With("user_id", "123", "action", "test")

	l2.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "user_id") {
		t.Error("expected output to contain user_id")
	}
	if !strings.Contains(output, "123") {
		t.Error("expected output to contain 123")
	}
	if !strings.Contains(output, "action") {
		t.Error("expected output to contain action")
	}
}

func TestLoggerWithContext(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := Config{
		Level:        LevelInfo,
		Format:       FormatJSON,
		Output:       buf,
		RequestIDKey: "request_id",
	}

	l := New(cfg)

	// Create context with request ID
	ctx := ContextWithRequestID(context.Background(), "test-request-123")
	ctx = ContextWithStartTime(ctx, time.Now())

	l2 := l.WithContext(ctx)
	l2.Info("test message")

	output := buf.String()

	// Parse JSON to verify
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if reqID, ok := logEntry["request_id"].(string); !ok || reqID != "test-request-123" {
		t.Errorf("expected request_id to be 'test-request-123', got %v", logEntry["request_id"])
	}
}

func TestLoggerJSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := Config{
		Level:  LevelInfo,
		Format: FormatJSON,
		Output: buf,
	}

	l := New(cfg)
	l.Info("test message", "key", "value")

	output := buf.String()

	// Parse JSON to verify
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if msg, ok := logEntry["msg"].(string); !ok || msg != "test message" {
		t.Errorf("expected msg to be 'test message', got %v", logEntry["msg"])
	}

	if key, ok := logEntry["key"].(string); !ok || key != "value" {
		t.Errorf("expected key to be 'value', got %v", logEntry["key"])
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"warning", LevelWarn},
		{"WARNING", LevelWarn},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"unknown", LevelInfo}, // default
		{"", LevelInfo},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("ParseLevel(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected Format
	}{
		{"json", FormatJSON},
		{"JSON", FormatJSON},
		{"text", FormatText},
		{"TEXT", FormatText},
		{"console", FormatText},
		{"unknown", FormatText}, // default
		{"", FormatText},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseFormat(tt.input)
			if result != tt.expected {
				t.Errorf("ParseFormat(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestContextWithRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "test-123"

	newCtx := ContextWithRequestID(ctx, requestID)
	result := GetRequestID(newCtx)

	if result != requestID {
		t.Errorf("GetRequestID() = %q, expected %q", result, requestID)
	}
}

func TestGetRequestIDNilContext(t *testing.T) {
	result := GetRequestID(context.Background())
	if result != "" {
		t.Errorf("GetRequestID(empty context) = %q, expected empty string", result)
	}
}

func TestContextWithStartTime(t *testing.T) {
	ctx := context.Background()
	startTime := time.Now()

	newCtx := ContextWithStartTime(ctx, startTime)
	result := GetStartTime(newCtx)

	if !result.Equal(startTime) {
		t.Errorf("GetStartTime() = %v, expected %v", result, startTime)
	}
}

func TestDurationSinceStart(t *testing.T) {
	ctx := context.Background()
	startTime := time.Now().Add(-2 * time.Second)

	newCtx := ContextWithStartTime(ctx, startTime)
	duration := DurationSinceStart(newCtx)

	// Allow some tolerance for test execution time
	if duration < 2*time.Second || duration > 3*time.Second {
		t.Errorf("DurationSinceStart() = %v, expected ~2s", duration)
	}
}

func TestDurationSinceStartNoStartTime(t *testing.T) {
	ctx := context.Background()
	duration := DurationSinceStart(ctx)

	if duration != 0 {
		t.Errorf("DurationSinceStart() = %v, expected 0", duration)
	}
}

func TestDefaultLogger(t *testing.T) {
	// Reset default logger for testing
	defaultLogger = nil

	l := Default()
	if l == nil {
		t.Fatal("Default() returned nil")
	}

	// Should return same instance on subsequent calls
	l2 := Default()
	if l != l2 {
		t.Error("Default() should return the same instance")
	}
}

func TestLoggerWithContextNilContext(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := Config{
		Level:  LevelInfo,
		Format: FormatText,
		Output: buf,
	}

	l := New(cfg)
	l2 := l.WithContext(context.Background())

	// Should not panic and should return logger
	if l2 == nil {
		t.Error("WithContext(empty context) returned nil")
	}
}

func TestLoggerWithKeyValuePairs(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := Config{
		Level:  LevelInfo,
		Format: FormatJSON,
		Output: buf,
	}

	l := New(cfg)
	l2 := l.With("key1", "value1", "key2", 42, "key3", true)
	l2.Info("test")

	output := buf.String()
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if logEntry["key1"] != "value1" {
		t.Errorf("expected key1 to be 'value1', got %v", logEntry["key1"])
	}
	if logEntry["key2"] != float64(42) {
		t.Errorf("expected key2 to be 42, got %v", logEntry["key2"])
	}
	if logEntry["key3"] != true {
		t.Errorf("expected key3 to be true, got %v", logEntry["key3"])
	}
}
