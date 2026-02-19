// Package logger provides structured logging capabilities using Go's built-in log/slog.
// It supports JSON and text output formats, configurable log levels, and request ID tracking.
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

// Level represents the logging level.
type Level string

const (
	// LevelDebug enables debug logging.
	LevelDebug Level = "debug"
	// LevelInfo enables info logging.
	LevelInfo Level = "info"
	// LevelWarn enables warning logging.
	LevelWarn Level = "warn"
	// LevelError enables error logging.
	LevelError Level = "error"
)

// Format represents the log output format.
type Format string

const (
	// FormatJSON outputs logs in JSON format.
	FormatJSON Format = "json"
	// FormatText outputs logs in human-readable text format.
	FormatText Format = "text"
)

// Config holds the logger configuration.
type Config struct {
	// Level is the minimum log level to output.
	Level Level
	// Format is the output format (json or text).
	Format Format
	// Output is the writer for log output (defaults to os.Stdout).
	Output io.Writer
	// AddSource adds source code location to log entries.
	AddSource bool
	// RequestIDKey is the context key for request ID tracking.
	RequestIDKey string
}

// DefaultConfig returns a default logger configuration.
func DefaultConfig() Config {
	return Config{
		Level:        LevelInfo,
		Format:       FormatText,
		Output:       os.Stdout,
		AddSource:    false,
		RequestIDKey: "request_id",
	}
}

// Logger wraps slog.Logger with additional functionality.
type Logger struct {
	*slog.Logger
	config Config
}

var (
	defaultLogger *Logger
	initOnce      sync.Once
)

// Init initializes the default logger with the given configuration.
// This should be called once at application startup.
func Init(cfg Config) *Logger {
	initOnce.Do(func() {
		defaultLogger = New(cfg)
	})
	return defaultLogger
}

// Default returns the default logger.
// If Init hasn't been called, it creates a logger with default configuration.
func Default() *Logger {
	if defaultLogger == nil {
		defaultLogger = New(DefaultConfig())
	}
	return defaultLogger
}

// New creates a new logger with the given configuration.
func New(cfg Config) *Logger {
	// Convert our Level to slog.Level
	var level slog.Level
	switch cfg.Level {
	case LevelDebug:
		level = slog.LevelDebug
	case LevelInfo:
		level = slog.LevelInfo
	case LevelWarn:
		level = slog.LevelWarn
	case LevelError:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create handler options
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize attribute handling if needed
			return a
		},
	}

	// Create appropriate handler based on format
	var handler slog.Handler
	switch cfg.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(cfg.Output, opts)
	case FormatText:
		fallthrough
	default:
		handler = slog.NewTextHandler(cfg.Output, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
		config: cfg,
	}
}

// WithContext returns a new logger with context values added.
// It extracts request ID from context if present.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if ctx == nil {
		return l
	}

	// Extract request ID from context if present
	requestID := GetRequestID(ctx)
	if requestID != "" {
		return &Logger{
			Logger: l.Logger.With(l.config.RequestIDKey, requestID),
			config: l.config,
		}
	}

	return l
}

// With returns a new logger with the given key-value pairs added.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		Logger: l.Logger.With(args...),
		config: l.config,
	}
}

// Debug logs a message at debug level.
func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

// Debugf logs a message at debug level with formatting.
func (l *Logger) Debugf(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

// Info logs a message at info level.
func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Infof logs a message at info level with formatting.
func (l *Logger) Infof(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Warn logs a message at warning level.
func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

// Warnf logs a message at warning level with formatting.
func (l *Logger) Warnf(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

// Error logs a message at error level.
func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

// Errorf logs a message at error level with formatting.
func (l *Logger) Errorf(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

// Log logs a message at the specified level.
func (l *Logger) Log(level Level, msg string, args ...any) {
	var logLevel slog.Level
	switch level {
	case LevelDebug:
		logLevel = slog.LevelDebug
	case LevelInfo:
		logLevel = slog.LevelInfo
	case LevelWarn:
		logLevel = slog.LevelWarn
	case LevelError:
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	l.Logger.Log(context.Background(), logLevel, msg, args...)
}

// LogContext logs a message at the specified level with context.
func (l *Logger) LogContext(ctx context.Context, level Level, msg string, args ...any) {
	var logLevel slog.Level
	switch level {
	case LevelDebug:
		logLevel = slog.LevelDebug
	case LevelInfo:
		logLevel = slog.LevelInfo
	case LevelWarn:
		logLevel = slog.LevelWarn
	case LevelError:
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	l.Logger.Log(ctx, logLevel, msg, args...)
}

// ParseLevel parses a string into a Level.
func ParseLevel(s string) Level {
	switch strings.ToLower(s) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// ParseFormat parses a string into a Format.
func ParseFormat(s string) Format {
	switch strings.ToLower(s) {
	case "json":
		return FormatJSON
	case "text", "console":
		return FormatText
	default:
		return FormatText
	}
}

// contextKey is a type for context keys.
type contextKey string

const (
	// RequestIDKey is the context key for request ID.
	RequestIDKey contextKey = "request_id"
	// StartTimeKey is the context key for request start time.
	StartTimeKey contextKey = "start_time"
)

// ContextWithRequestID returns a new context with the given request ID.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID extracts the request ID from the context.
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// ContextWithStartTime returns a new context with the start time.
func ContextWithStartTime(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, StartTimeKey, t)
}

// GetStartTime extracts the start time from the context.
func GetStartTime(ctx context.Context) time.Time {
	if ctx == nil {
		return time.Time{}
	}
	if t, ok := ctx.Value(StartTimeKey).(time.Time); ok {
		return t
	}
	return time.Time{}
}

// DurationSinceStart calculates the duration since the start time in the context.
func DurationSinceStart(ctx context.Context) time.Duration {
	start := GetStartTime(ctx)
	if start.IsZero() {
		return 0
	}
	return time.Since(start)
}
