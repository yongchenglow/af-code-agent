// Package retry implements retry logic with exponential backoff for fault tolerance.
// It provides configurable retry behavior for transient failures.
package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"strings"
	"time"
)

// Errors returned by the retry package.
var (
	// ErrMaxAttemptsExceeded is returned when all retry attempts are exhausted.
	ErrMaxAttemptsExceeded = errors.New("max retry attempts exceeded")
	// ErrContextCancelled is returned when the context is cancelled during retry.
	ErrContextCancelled = errors.New("context cancelled during retry")
)

// Config holds the retry configuration.
type Config struct {
	// MaxAttempts is the maximum number of retry attempts (including the initial attempt).
	// Default is 3 if not specified.
	MaxAttempts int
	// InitialDelay is the delay before the first retry.
	// Default is 1 second if not specified.
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries.
	// Default is 30 seconds if not specified.
	MaxDelay time.Duration
	// Multiplier is the factor by which the delay increases after each attempt.
	// Default is 2.0 if not specified.
	Multiplier float64
	// Jitter adds randomness to the delay to prevent thundering herd.
	// Value between 0.0 (no jitter) and 1.0 (max jitter).
	// Default is 0.1 if not specified.
	Jitter float64
	// Retryable is a function that determines if an error is retryable.
	// If nil, all errors are considered retryable.
	Retryable func(error) bool
	// OnRetry is an optional callback function called before each retry.
	// It receives the attempt number (1-based) and the error that triggered the retry.
	OnRetry func(attempt int, err error)
}

// DefaultConfig returns a default retry configuration.
func DefaultConfig() Config {
	return Config{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
		Retryable:    nil, // All errors are retryable by default
		OnRetry:      nil,
	}
}

// Retryer implements retry logic with exponential backoff.
type Retryer struct {
	config Config
}

// New creates a new Retryer with the given configuration.
func New(config Config) *Retryer {
	// Apply defaults
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 3
	}
	if config.InitialDelay <= 0 {
		config.InitialDelay = 1 * time.Second
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 30 * time.Second
	}
	if config.Multiplier <= 0 {
		config.Multiplier = 2.0
	}
	if config.Jitter < 0 || config.Jitter > 1 {
		config.Jitter = 0.1
	}

	return &Retryer{
		config: config,
	}
}

// Execute runs the given function with retry logic.
// It returns the result of the first successful attempt or the last error.
func (r *Retryer) Execute(ctx context.Context, fn func() error) error {
	var lastErr error
	var attempt int

	for attempt = 1; attempt <= r.config.MaxAttempts; attempt++ {
		// Check context before attempting
		if ctx.Err() != nil {
			return fmt.Errorf("%w: %w", ErrContextCancelled, ctx.Err())
		}

		// Execute the function
		err := fn()
		if err == nil {
			// Success
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !r.isRetryable(err) {
			// Non-retryable error, fail immediately
			return err
		}

		// Check if we have more attempts
		if attempt >= r.config.MaxAttempts {
			// No more attempts
			break
		}

		// Call onRetry callback if provided
		if r.config.OnRetry != nil {
			r.config.OnRetry(attempt, err)
		}

		// Calculate delay
		delay := r.calculateDelay(attempt)

		// Wait for the delay or context cancellation
		if !r.wait(ctx, delay) {
			return fmt.Errorf("%w: %w", ErrContextCancelled, ctx.Err())
		}
	}

	return fmt.Errorf("%w: %w", ErrMaxAttemptsExceeded, lastErr)
}

// ExecuteWithResult runs the given function with retry logic and returns its result.
// Note: For typed results, use Execute and handle type conversion externally.
func (r *Retryer) ExecuteWithResult(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	var lastErr error
	var attempt int
	var result interface{}

	for attempt = 1; attempt <= r.config.MaxAttempts; attempt++ {
		// Check context before attempting
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %w", ErrContextCancelled, ctx.Err())
		}

		// Execute the function
		var err error
		result, err = fn()
		if err == nil {
			// Success
			return result, nil
		}

		lastErr = err

		// Check if we should retry
		if !r.isRetryable(err) {
			// Non-retryable error, fail immediately
			return nil, err
		}

		// Check if we have more attempts
		if attempt >= r.config.MaxAttempts {
			// No more attempts
			break
		}

		// Call onRetry callback if provided
		if r.config.OnRetry != nil {
			r.config.OnRetry(attempt, err)
		}

		// Calculate delay
		delay := r.calculateDelay(attempt)

		// Wait for the delay or context cancellation
		if !r.wait(ctx, delay) {
			return nil, fmt.Errorf("%w: %w", ErrContextCancelled, ctx.Err())
		}
	}

	return nil, fmt.Errorf("%w: %w", ErrMaxAttemptsExceeded, lastErr)
}

// isRetryable checks if an error should trigger a retry.
func (r *Retryer) isRetryable(err error) bool {
	if r.config.Retryable == nil {
		return true
	}
	return r.config.Retryable(err)
}

// calculateDelay calculates the delay for the given attempt using exponential backoff.
func (r *Retryer) calculateDelay(attempt int) time.Duration {
	// Calculate exponential delay: initialDelay * multiplier^(attempt-1)
	expDelay := float64(r.config.InitialDelay) * math.Pow(r.config.Multiplier, float64(attempt-1))

	// Cap at max delay
	delay := time.Duration(math.Min(expDelay, float64(r.config.MaxDelay)))

	// Add jitter
	if r.config.Jitter > 0 {
		jitterRange := float64(delay) * r.config.Jitter
		jitter := (rand.Float64() * 2 * jitterRange) - jitterRange
		delay = time.Duration(float64(delay) + jitter)

		// Ensure delay is not negative
		if delay < 0 {
			delay = 0
		}
	}

	return delay
}

// wait waits for the specified duration or until context is cancelled.
// Returns true if the wait completed, false if cancelled.
func (r *Retryer) wait(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// DefaultRetryer is the default retryer used by package-level functions.
var DefaultRetryer = New(DefaultConfig())

// Execute runs the given function with the default retry logic.
func Execute(ctx context.Context, fn func() error) error {
	return DefaultRetryer.Execute(ctx, fn)
}

// ExecuteWithResult runs the given function with the default retry logic and returns its result.
func ExecuteWithResult(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	return DefaultRetryer.ExecuteWithResult(ctx, fn)
}

// IsRetryableError determines if an error is likely transient and retryable.
// This is a common implementation that checks for various transient error types.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Check for common transient error patterns
	transientPatterns := []string{
		"timeout",
		"temporary",
		"connection reset",
		"connection refused",
		"connection closed",
		"network is unreachable",
		"no such host",
		"i/o timeout",
		"server timeout",
		"gateway timeout",
		"service unavailable",
		"too many requests",
		"rate limit",
		"deadline exceeded",
		"resource temporarily unavailable",
	}

	for _, pattern := range transientPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	// Check for net.Error
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	return false
}

// AlwaysRetryable is a Retryable function that considers all errors retryable.
func AlwaysRetryable(error) bool {
	return true
}

// NeverRetryable is a Retryable function that considers no errors retryable.
func NeverRetryable(error) bool {
	return false
}

// TransientRetryable is a Retryable function that uses IsRetryableError.
func TransientRetryable(err error) bool {
	return IsRetryableError(err)
}

// ConfigPreset provides common retry configuration presets.
type ConfigPreset struct{}

// Conservative returns a conservative retry configuration (fewer retries, shorter delays).
func (ConfigPreset) Conservative() Config {
	return Config{
		MaxAttempts:  2,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   1.5,
		Jitter:       0.1,
		Retryable:    TransientRetryable,
		OnRetry:      nil,
	}
}

// Moderate returns a moderate retry configuration (balanced retries and delays).
func (ConfigPreset) Moderate() Config {
	return Config{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
		Retryable:    TransientRetryable,
		OnRetry:      nil,
	}
}

// Aggressive returns an aggressive retry configuration (more retries, longer delays).
func (ConfigPreset) Aggressive() Config {
	return Config{
		MaxAttempts:  5,
		InitialDelay: 2 * time.Second,
		MaxDelay:     60 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.2,
		Retryable:    TransientRetryable,
		OnRetry:      nil,
	}
}

// ForAI returns a retry configuration optimized for AI API calls.
func (ConfigPreset) ForAI() Config {
	return Config{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
		Retryable:    TransientRetryable,
		OnRetry: func(attempt int, err error) {
			// Can be used for logging
		},
	}
}

// ForGitHubAPI returns a retry configuration optimized for GitHub API calls.
func (ConfigPreset) ForGitHubAPI() Config {
	return Config{
		MaxAttempts:  5,
		InitialDelay: 500 * time.Millisecond,
		MaxDelay:     60 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
		Retryable:    TransientRetryable,
		OnRetry: func(attempt int, err error) {
			// Can be used for logging
		},
	}
}

// ForWebhook returns a retry configuration optimized for webhook processing.
func (ConfigPreset) ForWebhook() Config {
	return Config{
		MaxAttempts:  2,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.05,
		Retryable:    TransientRetryable,
		OnRetry:      nil,
	}
}
