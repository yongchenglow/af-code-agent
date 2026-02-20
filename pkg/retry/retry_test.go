package retry

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Test errors
var (
	errRetryable    = errors.New("retryable error")
	errNonRetryable = errors.New("non-retryable error")
)

func TestRetryer_New(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		r := New(Config{})

		if r.config.MaxAttempts != 3 {
			t.Errorf("maxAttempts = %d, want 3", r.config.MaxAttempts)
		}
		if r.config.InitialDelay != 1*time.Second {
			t.Errorf("initialDelay = %v, want 1s", r.config.InitialDelay)
		}
		if r.config.MaxDelay != 30*time.Second {
			t.Errorf("maxDelay = %v, want 30s", r.config.MaxDelay)
		}
		if r.config.Multiplier != 2.0 {
			t.Errorf("multiplier = %f, want 2.0", r.config.Multiplier)
		}
	})

	t.Run("custom config", func(t *testing.T) {
		r := New(Config{
			MaxAttempts:  5,
			InitialDelay: 2 * time.Second,
			MaxDelay:     60 * time.Second,
			Multiplier:   1.5,
			Jitter:       0.2,
			Retryable:    NeverRetryable,
		})

		if r.config.MaxAttempts != 5 {
			t.Errorf("maxAttempts = %d, want 5", r.config.MaxAttempts)
		}
		if r.config.InitialDelay != 2*time.Second {
			t.Errorf("initialDelay = %v, want 2s", r.config.InitialDelay)
		}
		if r.config.MaxDelay != 60*time.Second {
			t.Errorf("maxDelay = %v, want 60s", r.config.MaxDelay)
		}
		if r.config.Multiplier != 1.5 {
			t.Errorf("multiplier = %f, want 1.5", r.config.Multiplier)
		}
	})
}

func TestRetryer_Execute_Success(t *testing.T) {
	r := New(Config{MaxAttempts: 3, InitialDelay: 10 * time.Millisecond})

	var callCount int
	fn := func() error {
		callCount++
		return nil
	}

	err := r.Execute(context.Background(), fn)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}
}

func TestRetryer_Execute_Failure(t *testing.T) {
	r := New(Config{MaxAttempts: 3, InitialDelay: 10 * time.Millisecond})

	var callCount int
	fn := func() error {
		callCount++
		return errRetryable
	}

	err := r.Execute(context.Background(), fn)
	if err == nil {
		t.Error("expected error but got none")
	}
	if callCount != 3 {
		t.Errorf("callCount = %d, want 3", callCount)
	}

	if !errors.Is(err, ErrMaxAttemptsExceeded) {
		t.Errorf("error = %v, want ErrMaxAttemptsExceeded", err)
	}
}

func TestRetryer_Execute_SuccessAfterRetries(t *testing.T) {
	r := New(Config{MaxAttempts: 3, InitialDelay: 10 * time.Millisecond})

	var callCount int
	fn := func() error {
		callCount++
		if callCount < 3 {
			return errRetryable
		}
		return nil
	}

	err := r.Execute(context.Background(), fn)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if callCount != 3 {
		t.Errorf("callCount = %d, want 3", callCount)
	}
}

func TestRetryer_Execute_NonRetryableError(t *testing.T) {
	r := New(Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		Retryable: func(err error) bool {
			return errors.Is(err, errRetryable)
		},
	})

	var callCount int
	fn := func() error {
		callCount++
		return errNonRetryable
	}

	err := r.Execute(context.Background(), fn)
	if err != errNonRetryable {
		t.Errorf("error = %v, want errNonRetryable", err)
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}
}

func TestRetryer_Execute_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r := New(Config{MaxAttempts: 3, InitialDelay: 100 * time.Millisecond})

	var callCount int32
	fn := func() error {
		atomic.AddInt32(&callCount, 1)
		return errRetryable
	}

	// Cancel after first attempt
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := r.Execute(ctx, fn)
	if err == nil {
		t.Error("expected error but got none")
	}
	if !errors.Is(err, ErrContextCancelled) {
		t.Errorf("error = %v, want ErrContextCancelled", err)
	}
}

func TestRetryer_ExecuteWithResult_Success(t *testing.T) {
	r := New(Config{MaxAttempts: 3, InitialDelay: 10 * time.Millisecond})

	var callCount int
	fn := func() (interface{}, error) {
		callCount++
		return 42, nil
	}

	result, err := r.ExecuteWithResult(context.Background(), fn)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("result = %d, want 42", result)
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}
}

func TestRetryer_ExecuteWithResult_Failure(t *testing.T) {
	r := New(Config{MaxAttempts: 3, InitialDelay: 10 * time.Millisecond})

	var callCount int
	fn := func() (interface{}, error) {
		callCount++
		return 0, errRetryable
	}

	result, err := r.ExecuteWithResult(context.Background(), fn)
	if err == nil {
		t.Error("expected error but got none")
	}
	if result != nil {
		t.Errorf("result = %v, want nil", result)
	}
	if callCount != 3 {
		t.Errorf("callCount = %d, want 3", callCount)
	}
}

func TestRetryer_ExecuteWithResult_SuccessAfterRetries(t *testing.T) {
	r := New(Config{MaxAttempts: 3, InitialDelay: 10 * time.Millisecond})

	var callCount int
	fn := func() (interface{}, error) {
		callCount++
		if callCount < 2 {
			return 0, errRetryable
		}
		return 99, nil
	}

	result, err := r.ExecuteWithResult(context.Background(), fn)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != 99 {
		t.Errorf("result = %d, want 99", result)
	}
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2", callCount)
	}
}

func TestRetryer_CalculateDelay(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		attempt int
		wantMin time.Duration
		wantMax time.Duration
	}{
		{
			name: "first attempt",
			config: Config{
				InitialDelay: 1 * time.Second,
				MaxDelay:     30 * time.Second,
				Multiplier:   2.0,
				Jitter:       0,
			},
			attempt: 1,
			wantMin: 1 * time.Second,
			wantMax: 1 * time.Second,
		},
		{
			name: "second attempt",
			config: Config{
				InitialDelay: 1 * time.Second,
				MaxDelay:     30 * time.Second,
				Multiplier:   2.0,
				Jitter:       0,
			},
			attempt: 2,
			wantMin: 2 * time.Second,
			wantMax: 2 * time.Second,
		},
		{
			name: "third attempt",
			config: Config{
				InitialDelay: 1 * time.Second,
				MaxDelay:     30 * time.Second,
				Multiplier:   2.0,
				Jitter:       0,
			},
			attempt: 3,
			wantMin: 4 * time.Second,
			wantMax: 4 * time.Second,
		},
		{
			name: "capped at max delay",
			config: Config{
				InitialDelay: 1 * time.Second,
				MaxDelay:     5 * time.Second,
				Multiplier:   2.0,
				Jitter:       0,
			},
			attempt: 10,
			wantMin: 5 * time.Second,
			wantMax: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New(tt.config)
			delay := r.calculateDelay(tt.attempt)

			if delay < tt.wantMin || delay > tt.wantMax {
				t.Errorf("delay = %v, want between %v and %v", delay, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestRetryer_OnRetryCallback(t *testing.T) {
	r := New(Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		OnRetry: func(attempt int, err error) {
			// Callback called
		},
	})

	var callCount int
	var retryCount int
	r.config.OnRetry = func(attempt int, err error) {
		retryCount++
	}

	fn := func() error {
		callCount++
		return errRetryable
	}

	_ = r.Execute(context.Background(), fn)

	if callCount != 3 {
		t.Errorf("callCount = %d, want 3", callCount)
	}
	if retryCount != 2 {
		t.Errorf("retryCount = %d, want 2", retryCount)
	}
}

func TestRetryer_ExponentialBackoff(t *testing.T) {
	r := New(Config{
		MaxAttempts:  4,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     500 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0,
	})

	var callCount int
	var delays []time.Duration
	var lastTime time.Time

	r.config.OnRetry = func(attempt int, err error) {
		now := time.Now()
		if !lastTime.IsZero() {
			delays = append(delays, now.Sub(lastTime))
		}
		lastTime = now
	}

	fn := func() error {
		callCount++
		return errRetryable
	}

	startTime := time.Now()
	_ = r.Execute(context.Background(), fn)

	if callCount != 4 {
		t.Fatalf("callCount = %d, want 4", callCount)
	}
	// We expect 3 retries (after attempts 1, 2, 3), but only 2 delays measured
	// because the first onRetry doesn't have a previous time to measure from
	if len(delays) != 2 {
		t.Fatalf("delays length = %d, want 2", len(delays))
	}

	// Verify total time is approximately the sum of expected delays
	// Expected: 50ms (after attempt 1) + 100ms (after attempt 2) + 200ms (after attempt 3) = 350ms
	totalTime := time.Since(startTime)
	expectedMin := 300 * time.Millisecond // Allow some tolerance
	expectedMax := 500 * time.Millisecond

	if totalTime < expectedMin || totalTime > expectedMax {
		t.Errorf("total time = %v, want between %v and %v", totalTime, expectedMin, expectedMax)
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"timeout error", errors.New("i/o timeout"), true},
		{"temporary error", errors.New("temporary failure"), true},
		{"connection reset", errors.New("connection reset by peer"), true},
		{"connection refused", errors.New("connection refused"), true},
		{"rate limit", errors.New("rate limit exceeded"), true},
		{"non-retryable", errors.New("invalid input"), false},
		{"permanent error", errors.New("file not found"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryableError(tt.err)
			if got != tt.want {
				t.Errorf("IsRetryableError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestRetryableFunctions(t *testing.T) {
	t.Run("AlwaysRetryable", func(t *testing.T) {
		if !AlwaysRetryable(errRetryable) {
			t.Error("AlwaysRetryable should return true")
		}
		if !AlwaysRetryable(errNonRetryable) {
			t.Error("AlwaysRetryable should return true")
		}
	})

	t.Run("NeverRetryable", func(t *testing.T) {
		if NeverRetryable(errRetryable) {
			t.Error("NeverRetryable should return false")
		}
		if NeverRetryable(errNonRetryable) {
			t.Error("NeverRetryable should return false")
		}
	})

	t.Run("TransientRetryable", func(t *testing.T) {
		if !TransientRetryable(errors.New("timeout")) {
			t.Error("TransientRetryable should return true for timeout")
		}
		if TransientRetryable(errors.New("invalid input")) {
			t.Error("TransientRetryable should return false for invalid input")
		}
	})
}

func TestConfigPresets(t *testing.T) {
	preset := ConfigPreset{}

	t.Run("Conservative", func(t *testing.T) {
		cfg := preset.Conservative()
		if cfg.MaxAttempts != 2 {
			t.Errorf("maxAttempts = %d, want 2", cfg.MaxAttempts)
		}
		if cfg.InitialDelay != 500*time.Millisecond {
			t.Errorf("initialDelay = %v, want 500ms", cfg.InitialDelay)
		}
	})

	t.Run("Moderate", func(t *testing.T) {
		cfg := preset.Moderate()
		if cfg.MaxAttempts != 3 {
			t.Errorf("maxAttempts = %d, want 3", cfg.MaxAttempts)
		}
		if cfg.InitialDelay != 1*time.Second {
			t.Errorf("initialDelay = %v, want 1s", cfg.InitialDelay)
		}
	})

	t.Run("Aggressive", func(t *testing.T) {
		cfg := preset.Aggressive()
		if cfg.MaxAttempts != 5 {
			t.Errorf("maxAttempts = %d, want 5", cfg.MaxAttempts)
		}
		if cfg.InitialDelay != 2*time.Second {
			t.Errorf("initialDelay = %v, want 2s", cfg.InitialDelay)
		}
	})

	t.Run("ForAI", func(t *testing.T) {
		cfg := preset.ForAI()
		if cfg.MaxAttempts != 3 {
			t.Errorf("maxAttempts = %d, want 3", cfg.MaxAttempts)
		}
		if cfg.OnRetry == nil {
			t.Error("OnRetry should not be nil")
		}
	})

	t.Run("ForGitHubAPI", func(t *testing.T) {
		cfg := preset.ForGitHubAPI()
		if cfg.MaxAttempts != 5 {
			t.Errorf("maxAttempts = %d, want 5", cfg.MaxAttempts)
		}
		if cfg.InitialDelay != 500*time.Millisecond {
			t.Errorf("initialDelay = %v, want 500ms", cfg.InitialDelay)
		}
	})

	t.Run("ForWebhook", func(t *testing.T) {
		cfg := preset.ForWebhook()
		if cfg.MaxAttempts != 2 {
			t.Errorf("maxAttempts = %d, want 2", cfg.MaxAttempts)
		}
		if cfg.InitialDelay != 100*time.Millisecond {
			t.Errorf("initialDelay = %v, want 100ms", cfg.InitialDelay)
		}
	})
}

func TestDefaultRetryer(t *testing.T) {
	// Test that default retryer works
	var callCount int
	fn := func() error {
		callCount++
		if callCount < 3 {
			return errRetryable
		}
		return nil
	}

	err := Execute(context.Background(), fn)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if callCount != 3 {
		t.Errorf("callCount = %d, want 3", callCount)
	}
}

func TestRetryer_Concurrent(t *testing.T) {
	r := New(Config{MaxAttempts: 2, InitialDelay: 10 * time.Millisecond})

	var wg sync.WaitGroup
	var successCount, failureCount int32

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fn := func() error {
				if id%2 == 0 {
					return nil
				}
				return errRetryable
			}
			err := r.Execute(context.Background(), fn)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			} else {
				atomic.AddInt32(&failureCount, 1)
			}
		}(i)
	}

	wg.Wait()

	// Should have processed all requests without panics
	if successCount+failureCount != 50 {
		t.Errorf("total processed = %d, want 50", successCount+failureCount)
	}
}

func TestRetryer_Jitter(t *testing.T) {
	r := New(Config{
		MaxAttempts:  2,
		InitialDelay: 100 * time.Millisecond,
		Multiplier:   1.0,
		Jitter:       0.5,
	})

	// Run multiple times to verify jitter creates variation
	delays := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		delays[i] = r.calculateDelay(1)
	}

	// Verify all delays are within acceptable range (50ms to 150ms with 50% jitter)
	for i, delay := range delays {
		if delay < 50*time.Millisecond || delay > 150*time.Millisecond {
			t.Errorf("delay[%d] = %v, want between 50ms and 150ms", i, delay)
		}
	}

	// Verify there's some variation (not all the same)
	allSame := true
	for i := 1; i < len(delays); i++ {
		if delays[i] != delays[0] {
			allSame = false
			break
		}
	}

	// With 50% jitter, it's extremely unlikely all 10 delays would be identical
	// This is a probabilistic check, but should pass >99.9% of the time
	if allSame {
		t.Error("jitter should create variation in delays")
	}
}

func TestRetryer_WaitCancellation(t *testing.T) {
	r := New(Config{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Wait should return false immediately
	if r.wait(ctx, 1*time.Second) {
		t.Error("wait should return false for cancelled context")
	}
}
