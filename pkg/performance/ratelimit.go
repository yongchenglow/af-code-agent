package performance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/go-github/v57/github"
)

// RateLimitMonitor monitors GitHub API rate limits
type RateLimitMonitor struct {
	client        *github.Client
	mu            sync.RWMutex
	lastCheck     time.Time
	remaining     int
	limit         int
	resetTime     time.Time
	checkInterval time.Duration
}

// NewRateLimitMonitor creates a new rate limit monitor
func NewRateLimitMonitor(client *github.Client) *RateLimitMonitor {
	return &RateLimitMonitor{
		client:        client,
		checkInterval: 5 * time.Minute,
	}
}

// RateLimitInfo contains rate limit information
type RateLimitInfo struct {
	Remaining int
	Limit     int
	ResetTime time.Time
	ResetIn   time.Duration
}

// Check checks the current rate limit status
func (r *RateLimitMonitor) Check(ctx context.Context) (*RateLimitInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Use cached data if recent
	if time.Since(r.lastCheck) < r.checkInterval && r.limit > 0 {
		return &RateLimitInfo{
			Remaining: r.remaining,
			Limit:     r.limit,
			ResetTime: r.resetTime,
			ResetIn:   time.Until(r.resetTime),
		}, nil
	}

	// Fetch fresh rate limit data
	rateLimits, _, err := r.client.RateLimits(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limits: %w", err)
	}

	if rateLimits.Core != nil {
		r.remaining = rateLimits.Core.Remaining
		r.limit = rateLimits.Core.Limit
		r.resetTime = rateLimits.Core.Reset.Time
		r.lastCheck = time.Now()
	}

	return &RateLimitInfo{
		Remaining: r.remaining,
		Limit:     r.limit,
		ResetTime: r.resetTime,
		ResetIn:   time.Until(r.resetTime),
	}, nil
}

// WaitIfNeeded waits if rate limit is close to exhaustion
func (r *RateLimitMonitor) WaitIfNeeded(ctx context.Context, threshold int) error {
	info, err := r.Check(ctx)
	if err != nil {
		return err
	}

	// If remaining requests are below threshold, wait until reset
	if info.Remaining < threshold {
		waitDuration := info.ResetIn
		if waitDuration > 0 {
			select {
			case <-time.After(waitDuration):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return nil
}

// PercentRemaining returns the percentage of rate limit remaining
func (r *RateLimitMonitor) PercentRemaining() float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.limit == 0 {
		return 100.0
	}

	return float64(r.remaining) / float64(r.limit) * 100.0
}

// ShouldWait returns true if we should wait before making more requests
func (r *RateLimitMonitor) ShouldWait(threshold int) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.remaining < threshold
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	tokens     int
	maxTokens  int
	refillRate time.Duration
	mu         sync.Mutex
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Wait waits for a token to become available
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		rl.mu.Lock()

		// Refill tokens based on time passed
		rl.refill()

		if rl.tokens > 0 {
			rl.tokens--
			rl.mu.Unlock()
			return nil
		}

		// Calculate wait time
		waitTime := rl.refillRate

		rl.mu.Unlock()

		// Wait for next refill
		select {
		case <-time.After(waitTime):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// refill adds tokens based on elapsed time
func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)

	// Calculate how many tokens to add
	tokensToAdd := int(elapsed / rl.refillRate)

	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefill = now
	}
}

// Available returns the number of available tokens
func (rl *RateLimiter) Available() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()
	return rl.tokens
}

// BackoffStrategy defines exponential backoff strategy
type BackoffStrategy struct {
	initialDelay time.Duration
	maxDelay     time.Duration
	multiplier   float64
	attempt      int
	mu           sync.Mutex
}

// NewBackoffStrategy creates a new backoff strategy
func NewBackoffStrategy(initialDelay, maxDelay time.Duration) *BackoffStrategy {
	return &BackoffStrategy{
		initialDelay: initialDelay,
		maxDelay:     maxDelay,
		multiplier:   2.0,
		attempt:      0,
	}
}

// Next calculates the next backoff delay
func (b *BackoffStrategy) Next() time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.attempt++

	delay := float64(b.initialDelay) * float64(b.attempt) * b.multiplier
	if delay > float64(b.maxDelay) {
		delay = float64(b.maxDelay)
	}

	return time.Duration(delay)
}

// Reset resets the backoff attempt counter
func (b *BackoffStrategy) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.attempt = 0
}

// Attempt returns the current attempt number
func (b *BackoffStrategy) Attempt() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.attempt
}

// RetryWithBackoff retries a function with exponential backoff
func RetryWithBackoff(
	ctx context.Context,
	maxAttempts int,
	backoff *BackoffStrategy,
	fn func() error,
) error {
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := fn()
		if err == nil {
			backoff.Reset()
			return nil
		}

		lastErr = err

		if attempt < maxAttempts-1 {
			delay := backoff.Next()

			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("max retry attempts (%d) exceeded: %w", maxAttempts, lastErr)
}

// AdaptiveRateLimiter adjusts rate based on success/failure
type AdaptiveRateLimiter struct {
	limiter            *RateLimiter
	successCount       int
	failureCount       int
	mu                 sync.Mutex
	adjustmentInterval int
}

// NewAdaptiveRateLimiter creates an adaptive rate limiter
func NewAdaptiveRateLimiter(initialRate int, refillRate time.Duration) *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		limiter:            NewRateLimiter(initialRate, refillRate),
		adjustmentInterval: 10,
	}
}

// Wait waits for a token
func (a *AdaptiveRateLimiter) Wait(ctx context.Context) error {
	return a.limiter.Wait(ctx)
}

// RecordSuccess records a successful request
func (a *AdaptiveRateLimiter) RecordSuccess() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.successCount++
	a.adjust()
}

// RecordFailure records a failed request
func (a *AdaptiveRateLimiter) RecordFailure() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.failureCount++
	a.adjust()
}

// adjust adjusts the rate limit based on success/failure ratio
func (a *AdaptiveRateLimiter) adjust() {
	total := a.successCount + a.failureCount

	if total%a.adjustmentInterval == 0 && total > 0 {
		successRate := float64(a.successCount) / float64(total)

		// Increase tokens if high success rate
		if successRate > 0.95 && a.limiter.maxTokens < 100 {
			a.limiter.maxTokens += 5
		}

		// Decrease tokens if low success rate
		if successRate < 0.8 && a.limiter.maxTokens > 10 {
			a.limiter.maxTokens -= 5
		}

		// Reset counters
		a.successCount = 0
		a.failureCount = 0
	}
}
