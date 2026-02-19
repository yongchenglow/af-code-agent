// Package circuitbreaker implements the circuit breaker pattern for fault tolerance.
// It prevents cascading failures by failing fast when a service is unhealthy.
package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	// StateClosed is the normal state where requests pass through.
	// Failures are counted and may transition to Open state.
	StateClosed State = iota
	// StateOpen is the tripped state where requests fail immediately.
	// After the timeout period, transitions to HalfOpen state.
	StateOpen
	// StateHalfOpen is the testing state where a limited number of requests
	// are allowed through to test if the service has recovered.
	StateHalfOpen
)

// String returns a string representation of the state.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Errors returned by the circuit breaker.
var (
	// ErrCircuitOpen is returned when the circuit is open and requests are rejected.
	ErrCircuitOpen = errors.New("circuit breaker is open")
	// ErrMaxRequestsExceeded is returned when max requests in half-open state is exceeded.
	ErrMaxRequestsExceeded = errors.New("max requests in half-open state exceeded")
)

// Config holds the circuit breaker configuration.
type Config struct {
	// Threshold is the number of consecutive failures before opening the circuit.
	// Default is 5 if not specified.
	Threshold int
	// Timeout is the duration the circuit stays open before transitioning to half-open.
	// Default is 1 minute if not specified.
	Timeout time.Duration
	// HalfOpenMaxRequests is the maximum number of requests allowed in half-open state.
	// Default is 3 if not specified.
	HalfOpenMaxRequests int
	// Name is an optional name for the circuit breaker (for logging/metrics).
	Name string
}

// Metrics holds circuit breaker metrics.
type Metrics struct {
	// State is the current state of the circuit breaker.
	State State
	// FailureCount is the current consecutive failure count.
	FailureCount int
	// SuccessCount is the count of successful requests in half-open state.
	SuccessCount int
	// TotalRequests is the total number of requests processed.
	TotalRequests int64
	// TotalFailures is the total number of failures.
	TotalFailures int64
	// TotalSuccesses is the total number of successes.
	TotalSuccesses int64
	// LastFailureTime is the time of the last failure.
	LastFailureTime time.Time
	// LastStateChange is the time of the last state change.
	LastStateChange time.Time
	// StateChangeCount is the number of state changes.
	StateChangeCount int64
}

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	config  Config
	state   State
	mu      sync.RWMutex
	cond    *sync.Cond

	// Counters
	failureCount      int
	successCount      int
	totalRequests     int64
	totalFailures     int64
	totalSuccesses    int64
	stateChangeCount  int64

	// Timing
	lastFailureTime   time.Time
	lastStateChange   time.Time

	// Half-open state tracking
	halfOpenRequests int
}

// New creates a new circuit breaker with the given configuration.
func New(config Config) *CircuitBreaker {
	// Apply defaults
	if config.Threshold <= 0 {
		config.Threshold = 5
	}
	if config.Timeout <= 0 {
		config.Timeout = 1 * time.Minute
	}
	if config.HalfOpenMaxRequests <= 0 {
		config.HalfOpenMaxRequests = 3
	}

	cb := &CircuitBreaker{
		config:          config,
		state:           StateClosed,
		lastStateChange: time.Now(),
	}
	cb.cond = sync.NewCond(&cb.mu)

	return cb
}

// Execute runs the given function with circuit breaker protection.
// It returns an error if the circuit is open or if the function fails.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()

	// Check if we should allow the request
	if !cb.allowRequest() {
		cb.mu.Unlock()
		cb.totalRequests++
		return ErrCircuitOpen
	}

	cb.mu.Unlock()

	// Execute the function
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalRequests++

	if err != nil {
		cb.onFailure()
		return err
	}

	cb.onSuccess()
	return nil
}

// ExecuteWithResult runs the given function with circuit breaker protection and returns its result.
// Note: For typed results, use Execute and handle type conversion externally.
func (cb *CircuitBreaker) ExecuteWithResult(fn func() (interface{}, error)) (interface{}, error) {
	cb.mu.Lock()

	// Check if we should allow the request
	if !cb.allowRequest() {
		cb.mu.Unlock()
		cb.totalRequests++
		return nil, ErrCircuitOpen
	}

	cb.mu.Unlock()

	// Execute the function
	result, err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalRequests++

	if err != nil {
		cb.onFailure()
		return nil, err
	}

	cb.onSuccess()
	return result, nil
}

// allowRequest checks if a request should be allowed.
// Must be called with cb.mu held.
func (cb *CircuitBreaker) allowRequest() bool {
	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if timeout has elapsed
		if time.Since(cb.lastStateChange) >= cb.config.Timeout {
			cb.transitionTo(StateHalfOpen)
			return true
		}
		return false
	case StateHalfOpen:
		// Allow limited requests in half-open state
		return cb.halfOpenRequests < cb.config.HalfOpenMaxRequests
	default:
		return false
	}
}

// onSuccess handles a successful request.
// Must be called with cb.mu held.
func (cb *CircuitBreaker) onSuccess() {
	cb.totalSuccesses++

	switch cb.state {
	case StateHalfOpen:
		cb.successCount++
		cb.halfOpenRequests++
		// If we've had enough successes, close the circuit
		if cb.successCount >= cb.config.Threshold {
			cb.transitionTo(StateClosed)
		}
	case StateClosed:
		cb.failureCount = 0 // Reset failure count on success
	}
}

// onFailure handles a failed request.
// Must be called with cb.mu held.
func (cb *CircuitBreaker) onFailure() {
	cb.totalFailures++
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		// Open circuit if threshold reached
		if cb.failureCount >= cb.config.Threshold {
			cb.transitionTo(StateOpen)
		}
	case StateHalfOpen:
		// Immediately open circuit on failure in half-open state
		cb.transitionTo(StateOpen)
	}
}

// transitionTo changes the circuit breaker state.
// Must be called with cb.mu held.
func (cb *CircuitBreaker) transitionTo(newState State) {
	if cb.state == newState {
		return
	}

	cb.state = newState
	cb.lastStateChange = time.Now()
	cb.stateChangeCount++

	switch newState {
	case StateClosed:
		cb.failureCount = 0
		cb.successCount = 0
		cb.halfOpenRequests = 0
	case StateOpen:
		cb.successCount = 0
		cb.halfOpenRequests = 0
		// Signal any waiting goroutines
		cb.cond.Broadcast()
	case StateHalfOpen:
		cb.successCount = 0
		cb.halfOpenRequests = 0
	}
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Metrics returns the current metrics for the circuit breaker.
func (cb *CircuitBreaker) Metrics() Metrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return Metrics{
		State:            cb.state,
		FailureCount:     cb.failureCount,
		SuccessCount:     cb.successCount,
		TotalRequests:    cb.totalRequests,
		TotalFailures:    cb.totalFailures,
		TotalSuccesses:   cb.totalSuccesses,
		LastFailureTime:  cb.lastFailureTime,
		LastStateChange:  cb.lastStateChange,
		StateChangeCount: cb.stateChangeCount,
	}
}

// Reset resets the circuit breaker to its initial closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.transitionTo(StateClosed)
	cb.totalRequests = 0
	cb.totalFailures = 0
	cb.totalSuccesses = 0
	cb.failureCount = 0
	cb.successCount = 0
	cb.halfOpenRequests = 0
	cb.lastFailureTime = time.Time{}
}

// Name returns the circuit breaker name.
func (cb *CircuitBreaker) Name() string {
	return cb.config.Name
}

// String returns a string representation of the circuit breaker.
func (cb *CircuitBreaker) String() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return cb.config.Name + "(" + cb.state.String() + ")"
}
