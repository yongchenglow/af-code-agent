package circuitbreaker

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Test errors
var (
	errTest = errors.New("test error")
)

func TestCircuitBreaker_New(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		cb := New(Config{})

		if cb.config.Threshold != 5 {
			t.Errorf("threshold = %d, want 5", cb.config.Threshold)
		}
		if cb.config.Timeout != 1*time.Minute {
			t.Errorf("timeout = %v, want 1m", cb.config.Timeout)
		}
		if cb.config.HalfOpenMaxRequests != 3 {
			t.Errorf("half-open max requests = %d, want 3", cb.config.HalfOpenMaxRequests)
		}
		if cb.state != StateClosed {
			t.Errorf("state = %v, want closed", cb.state)
		}
	})

	t.Run("custom config", func(t *testing.T) {
		cb := New(Config{
			Threshold:           10,
			Timeout:             2 * time.Minute,
			HalfOpenMaxRequests: 5,
			Name:                "test-cb",
		})

		if cb.config.Threshold != 10 {
			t.Errorf("threshold = %d, want 10", cb.config.Threshold)
		}
		if cb.config.Timeout != 2*time.Minute {
			t.Errorf("timeout = %v, want 2m", cb.config.Timeout)
		}
		if cb.config.HalfOpenMaxRequests != 5 {
			t.Errorf("half-open max requests = %d, want 5", cb.config.HalfOpenMaxRequests)
		}
		if cb.config.Name != "test-cb" {
			t.Errorf("name = %q, want %q", cb.config.Name, "test-cb")
		}
	})
}

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	cb := New(Config{Threshold: 3, Timeout: 100 * time.Millisecond})

	var callCount int
	fn := func() error {
		callCount++
		return nil
	}

	// Execute successfully
	err := cb.Execute(fn)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}
	if cb.State() != StateClosed {
		t.Errorf("state = %v, want closed", cb.State())
	}

	metrics := cb.Metrics()
	if metrics.TotalRequests != 1 {
		t.Errorf("totalRequests = %d, want 1", metrics.TotalRequests)
	}
	if metrics.TotalSuccesses != 1 {
		t.Errorf("totalSuccesses = %d, want 1", metrics.TotalSuccesses)
	}
	if metrics.TotalFailures != 0 {
		t.Errorf("totalFailures = %d, want 0", metrics.TotalFailures)
	}
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	cb := New(Config{Threshold: 3, Timeout: 100 * time.Millisecond})

	var callCount int
	fn := func() error {
		callCount++
		return errTest
	}

	// Execute with failure
	err := cb.Execute(fn)
	if err != errTest {
		t.Errorf("error = %v, want errTest", err)
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}
	if cb.State() != StateClosed {
		t.Errorf("state = %v, want closed", cb.State())
	}

	metrics := cb.Metrics()
	if metrics.TotalRequests != 1 {
		t.Errorf("totalRequests = %d, want 1", metrics.TotalRequests)
	}
	if metrics.TotalFailures != 1 {
		t.Errorf("totalFailures = %d, want 1", metrics.TotalFailures)
	}
	if metrics.FailureCount != 1 {
		t.Errorf("failureCount = %d, want 1", metrics.FailureCount)
	}
}

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	t.Run("closed to open", func(t *testing.T) {
		cb := New(Config{Threshold: 3, Timeout: 100 * time.Millisecond})
		fn := func() error { return errTest }

		// Fail 3 times to open circuit
		for i := 0; i < 3; i++ {
			err := cb.Execute(fn)
			if err != errTest {
				t.Errorf("iteration %d: error = %v, want errTest", i, err)
			}
		}

		if cb.State() != StateOpen {
			t.Errorf("state = %v, want open", cb.State())
		}

		metrics := cb.Metrics()
		if metrics.StateChangeCount != 1 {
			t.Errorf("stateChangeCount = %d, want 1", metrics.StateChangeCount)
		}
	})

	t.Run("open to half-open", func(t *testing.T) {
		cb := New(Config{Threshold: 2, Timeout: 50 * time.Millisecond})
		fn := func() error { return errTest }

		// Open circuit
		_ = cb.Execute(fn)
		_ = cb.Execute(fn)

		if cb.State() != StateOpen {
			t.Fatalf("state = %v, want open", cb.State())
		}

		// Wait for timeout
		time.Sleep(60 * time.Millisecond)

		// Next request should transition to half-open
		successFn := func() error { return nil }
		err := cb.Execute(successFn)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if cb.State() != StateHalfOpen {
			t.Errorf("state = %v, want half-open", cb.State())
		}
	})

	t.Run("half-open to closed", func(t *testing.T) {
		cb := New(Config{Threshold: 2, Timeout: 50 * time.Millisecond})
		failFn := func() error { return errTest }
		successFn := func() error { return nil }

		// Open circuit
		_ = cb.Execute(failFn)
		_ = cb.Execute(failFn)

		// Wait for timeout
		time.Sleep(60 * time.Millisecond)

		// Succeed enough times to close
		for i := 0; i < 2; i++ {
			err := cb.Execute(successFn)
			if err != nil {
				t.Errorf("iteration %d: unexpected error: %v", i, err)
			}
		}

		if cb.State() != StateClosed {
			t.Errorf("state = %v, want closed", cb.State())
		}
	})

	t.Run("half-open to open", func(t *testing.T) {
		cb := New(Config{Threshold: 2, Timeout: 50 * time.Millisecond})
		failFn := func() error { return errTest }
		successFn := func() error { return nil }

		// Open circuit
		_ = cb.Execute(failFn)
		_ = cb.Execute(failFn)

		// Wait for timeout
		time.Sleep(60 * time.Millisecond)

		// Succeed once
		err := cb.Execute(successFn)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if cb.State() != StateHalfOpen {
			t.Fatalf("state = %v, want half-open", cb.State())
		}

		// Fail once to reopen
		err = cb.Execute(failFn)
		if err != errTest {
			t.Errorf("error = %v, want errTest", err)
		}

		if cb.State() != StateOpen {
			t.Errorf("state = %v, want open", cb.State())
		}
	})
}

func TestCircuitBreaker_Open_RejectsRequests(t *testing.T) {
	cb := New(Config{Threshold: 2, Timeout: 1 * time.Second})
	failFn := func() error { return errTest }

	// Open circuit
	_ = cb.Execute(failFn)
	_ = cb.Execute(failFn)

	if cb.State() != StateOpen {
		t.Fatalf("state = %v, want open", cb.State())
	}

	// Requests should be rejected immediately
	var callCount int32
	testFn := func() error {
		atomic.AddInt32(&callCount, 1)
		return nil
	}

	err := cb.Execute(testFn)
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("error = %v, want ErrCircuitOpen", err)
	}
	if callCount != 0 {
		t.Errorf("callCount = %d, want 0 (function should not be called)", callCount)
	}

	metrics := cb.Metrics()
	if metrics.TotalRequests != 3 {
		t.Errorf("totalRequests = %d, want 3", metrics.TotalRequests)
	}
}

func TestCircuitBreaker_HalfOpen_LimitsRequests(t *testing.T) {
	cb := New(Config{Threshold: 10, Timeout: 50 * time.Millisecond, HalfOpenMaxRequests: 2})
	failFn := func() error { return errTest }
	successFn := func() error { return nil }

	// Open circuit (need 10 failures)
	for i := 0; i < 10; i++ {
		_ = cb.Execute(failFn)
	}

	if cb.State() != StateOpen {
		t.Fatalf("state = %v, want open", cb.State())
	}

	// Wait for timeout to expire
	time.Sleep(60 * time.Millisecond)

	// First request should transition to half-open and succeed
	err := cb.Execute(successFn)
	if err != nil {
		t.Errorf("first request: unexpected error: %v", err)
	}

	// Second request should succeed (still in half-open, under limit)
	err = cb.Execute(successFn)
	if err != nil {
		t.Errorf("second request: unexpected error: %v", err)
	}

	// Verify circuit is still in half-open state (threshold is 10)
	if cb.State() != StateHalfOpen {
		t.Fatalf("state after 2 successes = %v, want half-open", cb.State())
	}

	// Third request should be rejected (half-open max requests reached)
	err = cb.Execute(successFn)
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("error = %v, want ErrCircuitOpen", err)
	}
}

func TestCircuitBreaker_ExecuteWithResult(t *testing.T) {
	cb := New(Config{Threshold: 2, Timeout: 100 * time.Millisecond})

	t.Run("success", func(t *testing.T) {
		fn := func() (interface{}, error) { return 42, nil }

		result, err := cb.ExecuteWithResult(fn)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != 42 {
			t.Errorf("result = %d, want 42", result)
		}
	})

	t.Run("failure", func(t *testing.T) {
		fn := func() (interface{}, error) { return 0, errTest }

		result, err := cb.ExecuteWithResult(fn)
		if err != errTest {
			t.Errorf("error = %v, want errTest", err)
		}
		if result != nil {
			t.Errorf("result = %v, want nil", result)
		}
	})

	t.Run("circuit open", func(t *testing.T) {
		// Open circuit first
		failFn := func() error { return errTest }
		_ = cb.Execute(failFn)
		_ = cb.Execute(failFn)

		fn := func() (interface{}, error) { return 42, nil }
		result, err := cb.ExecuteWithResult(fn)
		if !errors.Is(err, ErrCircuitOpen) {
			t.Errorf("error = %v, want ErrCircuitOpen", err)
		}
		if result != nil {
			t.Errorf("result = %v, want nil", result)
		}
	})
}

func TestCircuitBreaker_Metrics(t *testing.T) {
	cb := New(Config{Threshold: 2, Timeout: 50 * time.Millisecond, Name: "test-metrics"})
	failFn := func() error { return errTest }
	successFn := func() error { return nil }

	// Initial metrics
	metrics := cb.Metrics()
	if metrics.State != StateClosed {
		t.Errorf("initial state = %v, want closed", metrics.State)
	}
	if metrics.FailureCount != 0 {
		t.Errorf("initial failureCount = %d, want 0", metrics.FailureCount)
	}

	// Fail once
	_ = cb.Execute(failFn)
	metrics = cb.Metrics()
	if metrics.FailureCount != 1 {
		t.Errorf("after 1 failure: failureCount = %d, want 1", metrics.FailureCount)
	}
	if metrics.TotalFailures != 1 {
		t.Errorf("after 1 failure: totalFailures = %d, want 1", metrics.TotalFailures)
	}
	if metrics.LastFailureTime.IsZero() {
		t.Error("lastFailureTime should be set")
	}

	// Open circuit
	_ = cb.Execute(failFn)
	metrics = cb.Metrics()
	if metrics.State != StateOpen {
		t.Errorf("after 2 failures: state = %v, want open", metrics.State)
	}
	if metrics.StateChangeCount != 1 {
		t.Errorf("stateChangeCount = %d, want 1", metrics.StateChangeCount)
	}

	// Wait and recover
	time.Sleep(60 * time.Millisecond)
	_ = cb.Execute(successFn)
	_ = cb.Execute(successFn)

	metrics = cb.Metrics()
	if metrics.State != StateClosed {
		t.Errorf("after recovery: state = %v, want closed", metrics.State)
	}
	if metrics.TotalSuccesses != 2 {
		t.Errorf("totalSuccesses = %d, want 2", metrics.TotalSuccesses)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := New(Config{Threshold: 2, Timeout: 100 * time.Millisecond})
	failFn := func() error { return errTest }

	// Open circuit
	_ = cb.Execute(failFn)
	_ = cb.Execute(failFn)

	if cb.State() != StateOpen {
		t.Fatalf("state = %v, want open", cb.State())
	}

	// Reset
	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("after reset: state = %v, want closed", cb.State())
	}

	metrics := cb.Metrics()
	if metrics.TotalRequests != 0 {
		t.Errorf("after reset: totalRequests = %d, want 0", metrics.TotalRequests)
	}
	if metrics.TotalFailures != 0 {
		t.Errorf("after reset: totalFailures = %d, want 0", metrics.TotalFailures)
	}
	if metrics.FailureCount != 0 {
		t.Errorf("after reset: failureCount = %d, want 0", metrics.FailureCount)
	}
}

func TestCircuitBreaker_Name(t *testing.T) {
	cb := New(Config{Name: "my-circuit-breaker"})
	if cb.Name() != "my-circuit-breaker" {
		t.Errorf("name = %q, want %q", cb.Name(), "my-circuit-breaker")
	}
}

func TestCircuitBreaker_String(t *testing.T) {
	cb := New(Config{Name: "test", Threshold: 2})
	expected := "test(closed)"
	if cb.String() != expected {
		t.Errorf("string = %q, want %q", cb.String(), expected)
	}

	// Open it
	failFn := func() error { return errTest }
	_ = cb.Execute(failFn)
	_ = cb.Execute(failFn)

	expected = "test(open)"
	if cb.String() != expected {
		t.Errorf("string = %q, want %q", cb.String(), expected)
	}
}

func TestCircuitBreaker_Concurrent(t *testing.T) {
	cb := New(Config{Threshold: 10, Timeout: 100 * time.Millisecond})

	var wg sync.WaitGroup
	var successCount, failureCount int32

	// Start multiple goroutines
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fn := func() error {
				if id%2 == 0 {
					return nil
				}
				return errTest
			}
			err := cb.Execute(fn)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			} else if !errors.Is(err, ErrCircuitOpen) {
				atomic.AddInt32(&failureCount, 1)
			}
		}(i)
	}

	wg.Wait()

	// Should have processed requests without panics
	metrics := cb.Metrics()
	if metrics.TotalRequests != 100 {
		t.Errorf("totalRequests = %d, want 100", metrics.TotalRequests)
	}
}

func TestCircuitBreaker_TimeoutBehavior(t *testing.T) {
	cb := New(Config{Threshold: 5, Timeout: 100 * time.Millisecond})
	failFn := func() error { return errTest }
	successFn := func() error { return nil }

	// Open circuit (need 5 failures)
	for i := 0; i < 5; i++ {
		_ = cb.Execute(failFn)
	}

	if cb.State() != StateOpen {
		t.Fatalf("state = %v, want open", cb.State())
	}

	// Try immediately - should fail
	err := cb.Execute(successFn)
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("immediate retry: error = %v, want ErrCircuitOpen", err)
	}

	// Wait for timeout
	time.Sleep(110 * time.Millisecond)

	// Should allow request and transition to half-open
	err = cb.Execute(successFn)
	if err != nil {
		t.Errorf("after timeout: error = %v, want nil", err)
	}

	if cb.State() != StateHalfOpen {
		t.Errorf("state = %v, want half-open", cb.State())
	}
}

func TestState_String(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCircuitBreaker_SuccessResetsFailureCount(t *testing.T) {
	cb := New(Config{Threshold: 3, Timeout: 100 * time.Millisecond})
	failFn := func() error { return errTest }
	successFn := func() error { return nil }

	// Fail twice (but not enough to open)
	_ = cb.Execute(failFn)
	_ = cb.Execute(failFn)

	metrics := cb.Metrics()
	if metrics.FailureCount != 2 {
		t.Errorf("failureCount = %d, want 2", metrics.FailureCount)
	}

	// Succeed once
	_ = cb.Execute(successFn)

	metrics = cb.Metrics()
	if metrics.FailureCount != 0 {
		t.Errorf("after success: failureCount = %d, want 0", metrics.FailureCount)
	}

	// Fail twice again - should still not open (threshold is 3)
	_ = cb.Execute(failFn)
	_ = cb.Execute(failFn)

	if cb.State() != StateClosed {
		t.Errorf("state = %v, want closed", cb.State())
	}
}
