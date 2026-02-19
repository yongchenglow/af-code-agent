# Reliability Features

This document describes the reliability enhancements implemented in the GitHub Code Review Agent, including circuit breaker and retry patterns.

## Overview

The agent implements two key reliability patterns to handle transient failures and prevent cascading failures:

1. **Circuit Breaker Pattern** - Prevents cascading failures by failing fast when a service is unhealthy
2. **Retry with Exponential Backoff** - Handles transient failures gracefully with intelligent retry logic

## Circuit Breaker Pattern

### Package
```
pkg/circuitbreaker/circuitbreaker.go
```

### States

The circuit breaker has three states:

1. **Closed** (Normal Operation)
   - Requests pass through normally
   - Failures are counted
   - Opens when failure threshold is reached

2. **Open** (Failing Fast)
   - All requests are rejected immediately
   - Returns `ErrCircuitOpen`
   - Transitions to Half-Open after timeout

3. **Half-Open** (Testing Recovery)
   - Limited number of requests allowed
   - Success closes the circuit
   - Failure opens the circuit

### Configuration

```go
circuitbreaker.Config{
    Threshold:           5,      // Failures before opening
    Timeout:             1 * time.Minute,  // Time before half-open
    HalfOpenMaxRequests: 3,      // Requests allowed in half-open
    Name:                "ai-service",
}
```

### Default Configurations

#### AI Service Circuit Breaker
- **Threshold**: 5 consecutive failures
- **Timeout**: 1 minute
- **Half-Open Max Requests**: 3
- **Name**: "ai-service"

#### GitHub API Circuit Breaker
- **Threshold**: 10 consecutive failures
- **Timeout**: 2 minutes
- **Half-Open Max Requests**: 5
- **Name**: "github-api"

### Usage Example

```go
cb := circuitbreaker.New(circuitbreaker.Config{
    Threshold: 5,
    Timeout: 1 * time.Minute,
    Name: "my-service",
})

err := cb.Execute(func() error {
    // Your operation here
    return someService.Call()
})

if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
    // Handle gracefully - service is unavailable
    log.Warn("Service temporarily unavailable")
}
```

### Metrics

The circuit breaker exports comprehensive metrics:

```go
metrics := cb.Metrics()
fmt.Printf("State: %v\n", metrics.State)
fmt.Printf("Failure Count: %d\n", metrics.FailureCount)
fmt.Printf("Total Requests: %d\n", metrics.TotalRequests)
fmt.Printf("Total Failures: %d\n", metrics.TotalFailures)
fmt.Printf("State Changes: %d\n", metrics.StateChangeCount)
```

### Monitoring

Monitor these key metrics:

| Metric | Alert Threshold | Action |
|--------|----------------|--------|
| Circuit Opens per Hour | > 20 | Investigate service health |
| Failure Rate | > 25% | Review error logs |
| Time in Open State | > 5 min | Check service recovery |

## Retry with Exponential Backoff

### Package
```
pkg/retry/retry.go
```

### Configuration

```go
retry.Config{
    MaxAttempts:    3,              // Total attempts (initial + retries)
    InitialDelay:   1 * time.Second, // Delay before first retry
    MaxDelay:       30 * time.Second, // Maximum delay cap
    Multiplier:     2.0,            // Exponential backoff multiplier
    Jitter:         0.1,            // Randomness to prevent thundering herd
    Retryable:      retry.TransientRetryable, // Error classification
}
```

### Default Configurations

#### AI API Calls
```go
retry.Config{
    MaxAttempts:    3,
    InitialDelay:   1 * time.Second,
    MaxDelay:       30 * time.Second,
    Multiplier:     2.0,
    Jitter:         0.1,
    Retryable:      retry.TransientRetryable,
}
```

#### GitHub API Calls
```go
retry.Config{
    MaxAttempts:    5,
    InitialDelay:   500 * time.Millisecond,
    MaxDelay:       60 * time.Second,
    Multiplier:     2.0,
    Jitter:         0.1,
    Retryable:      retry.TransientRetryable,
}
```

#### Webhook Processing
```go
retry.Config{
    MaxAttempts:    2,
    InitialDelay:   100 * time.Millisecond,
    MaxDelay:       1 * time.Second,
    Multiplier:     2.0,
    Jitter:         0.05,
}
```

### Retryable Errors

The retry package automatically classifies errors as retryable or non-retryable:

**Retryable (Transient) Errors:**
- Timeouts
- Connection reset/refused/closed
- Network unreachable
- Rate limit errors (429)
- Service unavailable (503)
- Gateway timeout (504)

**Non-Retryable (Permanent) Errors:**
- Invalid input
- Authentication failures
- File not found
- Permission denied

### Usage Example

```go
retryer := retry.New(retry.Config{
    MaxAttempts: 3,
    InitialDelay: 1 * time.Second,
    MaxDelay: 30 * time.Second,
    Multiplier: 2.0,
    Retryable: func(err error) bool {
        return retry.IsRetryableError(err)
    },
    OnRetry: func(attempt int, err error) {
        log.Warn("Retrying", "attempt", attempt, "error", err)
    },
})

err := retryer.Execute(ctx, func() error {
    return externalService.Call()
})

if errors.Is(err, retry.ErrMaxAttemptsExceeded) {
    log.Error("Operation failed after all retries")
}
```

### Exponential Backoff Calculation

Delay formula: `min(initialDelay * multiplier^(attempt-1), maxDelay)`

Example with InitialDelay=1s, Multiplier=2.0, MaxDelay=30s:
- Attempt 1: 1 second
- Attempt 2: 2 seconds
- Attempt 3: 4 seconds
- Attempt 4: 8 seconds
- Attempt 5: 16 seconds
- Attempt 6+: 30 seconds (capped)

### Jitter

Jitter adds randomness to prevent the "thundering herd" problem where many clients retry simultaneously:

```
actualDelay = delay ± (delay * jitter)
```

With 10% jitter, a 10 second delay becomes 9-11 seconds.

## Integration Points

### Reviewer Agent

The code reviewer integrates both patterns:

```go
// In agents/reviewer/reviewer.go
reviewer := NewReviewerWithConfig(agent, ReviewerConfig{
    AICircuitBreaker: circuitbreaker.New(...),
    AIRetryer: retry.New(...),
})

report, err := reviewer.ReviewCode(ctx, files, prContext)
```

### GitHub Client

The GitHub API client includes built-in protection:

```go
client, err := github.NewClient(token)
// Circuit breaker and retryer are automatically configured

files, err := client.GetPRFiles(ctx, owner, repo, prNumber)
// Automatically retries on transient errors
// Opens circuit breaker on repeated failures
```

## Error Handling

### Circuit Breaker Errors

```go
err := cb.Execute(func() error { ... })

if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
    // Service is temporarily unavailable
    // Return graceful error to user
    return nil, ErrServiceUnavailable
}
```

### Retry Errors

```go
err := retryer.Execute(ctx, func() error { ... })

if errors.Is(err, retry.ErrMaxAttemptsExceeded) {
    // All retries exhausted
    // Log and return error
    log.Error("Operation failed after retries")
}

if errors.Is(err, retry.ErrContextCancelled) {
    // Context cancelled during retry
    // Clean up and exit
    return ctx.Err()
}
```

## Best Practices

### 1. Configure Appropriately

- **AI Services**: Lower threshold (5), shorter timeout (1m)
- **GitHub API**: Higher threshold (10), longer timeout (2m)
- **Critical Operations**: More retries, longer delays
- **User-Facing**: Fewer retries, faster failure

### 2. Monitor and Alert

Set up alerts for:
- High circuit breaker open rates
- Excessive retry attempts
- Increasing failure trends

### 3. Log Strategically

```go
// Log retry attempts
config.OnRetry = func(attempt int, err error) {
    log.Warn("Retry attempt",
        "attempt", attempt,
        "error", err,
        "operation", operationName)
}

// Log circuit state changes
if cb.State() != previousState {
    log.Info("Circuit state changed",
        "from", previousState,
        "to", cb.State(),
        "metrics", cb.Metrics())
}
```

### 4. Test Failure Scenarios

Include tests for:
- Circuit breaker state transitions
- Retry behavior with different error types
- Context cancellation during retries
- Concurrent access patterns

## Testing

### Unit Tests

Both packages have comprehensive test suites:

```bash
# Run circuit breaker tests
go test ./pkg/circuitbreaker/... -v

# Run retry tests
go test ./pkg/retry/... -v
```

### Test Coverage

Key scenarios tested:
- State transitions (Closed → Open → Half-Open → Closed)
- Request rejection in Open state
- Half-Open request limiting
- Exponential backoff calculation
- Context cancellation
- Concurrent access
- Error classification

## Troubleshooting

### Circuit Breaker Won't Close

**Symptoms**: Circuit stays open indefinitely

**Causes**:
1. Service still failing (check logs)
2. Timeout too long
3. Half-Open requests failing

**Solutions**:
1. Check service health
2. Reduce timeout for testing
3. Increase HalfOpenMaxRequests

### Too Many Retries

**Symptoms**: High latency, many retry logs

**Causes**:
1. MaxAttempts too high
2. InitialDelay too long
3. Non-retryable errors being retried

**Solutions**:
1. Reduce MaxAttempts
2. Lower InitialDelay
3. Improve error classification

### Thundering Herd

**Symptoms**: Spikes in errors after outage

**Causes**:
1. No jitter configured
2. All clients retry simultaneously

**Solutions**:
1. Enable jitter (0.1-0.2)
2. Stagger client startup
3. Use longer MaxDelay

## Performance Impact

### Overhead

- **Circuit Breaker**: < 1μs per operation (lock contention only)
- **Retry**: Only on failure (successful operations have no overhead)

### Memory

- **Circuit Breaker**: ~200 bytes per instance
- **Retry**: ~100 bytes per instance

### Recommendations

- Create one circuit breaker per service
- Reuse retryers with same configuration
- Monitor metrics in production

## Future Enhancements

Planned improvements:

1. **Metrics Export**: Prometheus/Grafana integration
2. **Adaptive Thresholds**: Auto-adjust based on error rates
3. **Cascading Protection**: Coordinate circuit breakers across services
4. **Request Hedging**: Send duplicate requests after delay
5. **Bulkhead Pattern**: Limit concurrent requests per service

## References

- [Circuit Breaker Pattern (Martin Fowler)](https://martinfowler.com/bliki/CircuitBreaker.html)
- [Release It! by Michael Nygard](https://smile.amazon.com/Release-Design-Deploy-Production-Ready-Software/dp/1680502395)
- [Exponential Backoff](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/)
