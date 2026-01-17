package security

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter provides rate limiting for API calls.
type RateLimiter struct {
	// Configuration
	requestsPerMinute int
	requestsPerHour   int
	tokensPerMinute   int
	tokensPerHour     int
	burstSize         int

	// State
	minuteRequests   int
	hourRequests     int
	minuteTokens     int
	hourTokens       int
	lastMinuteReset  time.Time
	lastHourReset    time.Time

	// Token bucket
	tokens     float64
	lastRefill time.Time
	refillRate float64 // tokens per second

	mu sync.Mutex
}

// RateLimitConfig contains configuration for the rate limiter.
type RateLimitConfig struct {
	RequestsPerMinute int
	RequestsPerHour   int
	TokensPerMinute   int
	TokensPerHour     int
	BurstSize         int
}

// DefaultRateLimitConfig returns sensible defaults for LLM rate limiting.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerMinute: 60,
		RequestsPerHour:   500,
		TokensPerMinute:   100000,
		TokensPerHour:     1000000,
		BurstSize:         10,
	}
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	now := time.Now()
	return &RateLimiter{
		requestsPerMinute: config.RequestsPerMinute,
		requestsPerHour:   config.RequestsPerHour,
		tokensPerMinute:   config.TokensPerMinute,
		tokensPerHour:     config.TokensPerHour,
		burstSize:         config.BurstSize,
		lastMinuteReset:   now,
		lastHourReset:     now,
		tokens:            float64(config.BurstSize),
		lastRefill:        now,
		refillRate:        float64(config.RequestsPerMinute) / 60.0,
	}
}

// RateLimitError represents a rate limit error.
type RateLimitError struct {
	LimitType   string
	Current     int
	Limit       int
	RetryAfter  time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded: %s (%d/%d), retry after %v",
		e.LimitType, e.Current, e.Limit, e.RetryAfter)
}

// Allow checks if a request is allowed and updates counters.
func (rl *RateLimiter) Allow() error {
	return rl.AllowN(1, 0)
}

// AllowN checks if N requests with the given token count are allowed.
func (rl *RateLimiter) AllowN(requests int, tokens int) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.resetCountersIfNeeded()
	rl.refillTokenBucket()

	// Check token bucket (for burst control)
	if rl.tokens < float64(requests) {
		waitTime := time.Duration((float64(requests) - rl.tokens) / rl.refillRate * float64(time.Second))
		return &RateLimitError{
			LimitType:  "burst",
			Current:    int(rl.tokens),
			Limit:      rl.burstSize,
			RetryAfter: waitTime,
		}
	}

	// Check minute limits
	if rl.minuteRequests+requests > rl.requestsPerMinute {
		waitTime := time.Until(rl.lastMinuteReset.Add(time.Minute))
		return &RateLimitError{
			LimitType:  "requests_per_minute",
			Current:    rl.minuteRequests,
			Limit:      rl.requestsPerMinute,
			RetryAfter: waitTime,
		}
	}

	if tokens > 0 && rl.minuteTokens+tokens > rl.tokensPerMinute {
		waitTime := time.Until(rl.lastMinuteReset.Add(time.Minute))
		return &RateLimitError{
			LimitType:  "tokens_per_minute",
			Current:    rl.minuteTokens,
			Limit:      rl.tokensPerMinute,
			RetryAfter: waitTime,
		}
	}

	// Check hour limits
	if rl.hourRequests+requests > rl.requestsPerHour {
		waitTime := time.Until(rl.lastHourReset.Add(time.Hour))
		return &RateLimitError{
			LimitType:  "requests_per_hour",
			Current:    rl.hourRequests,
			Limit:      rl.requestsPerHour,
			RetryAfter: waitTime,
		}
	}

	if tokens > 0 && rl.hourTokens+tokens > rl.tokensPerHour {
		waitTime := time.Until(rl.lastHourReset.Add(time.Hour))
		return &RateLimitError{
			LimitType:  "tokens_per_hour",
			Current:    rl.hourTokens,
			Limit:      rl.tokensPerHour,
			RetryAfter: waitTime,
		}
	}

	// Update counters
	rl.tokens -= float64(requests)
	rl.minuteRequests += requests
	rl.hourRequests += requests
	rl.minuteTokens += tokens
	rl.hourTokens += tokens

	return nil
}

// Wait blocks until a request is allowed or context is cancelled.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	return rl.WaitN(ctx, 1, 0)
}

// WaitN blocks until N requests with given tokens are allowed.
func (rl *RateLimiter) WaitN(ctx context.Context, requests int, tokens int) error {
	for {
		err := rl.AllowN(requests, tokens)
		if err == nil {
			return nil
		}

		rlErr, ok := err.(*RateLimitError)
		if !ok {
			return err
		}

		// Wait for retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(rlErr.RetryAfter):
			// Retry
		}
	}
}

func (rl *RateLimiter) resetCountersIfNeeded() {
	now := time.Now()

	// Reset minute counters
	if now.Sub(rl.lastMinuteReset) >= time.Minute {
		rl.minuteRequests = 0
		rl.minuteTokens = 0
		rl.lastMinuteReset = now
	}

	// Reset hour counters
	if now.Sub(rl.lastHourReset) >= time.Hour {
		rl.hourRequests = 0
		rl.hourTokens = 0
		rl.lastHourReset = now
	}
}

func (rl *RateLimiter) refillTokenBucket() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	rl.lastRefill = now

	rl.tokens += elapsed * rl.refillRate
	if rl.tokens > float64(rl.burstSize) {
		rl.tokens = float64(rl.burstSize)
	}
}

// Status returns the current rate limit status.
func (rl *RateLimiter) Status() RateLimitStatus {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.resetCountersIfNeeded()
	rl.refillTokenBucket()

	return RateLimitStatus{
		MinuteRequests:    rl.minuteRequests,
		MinuteRequestsMax: rl.requestsPerMinute,
		HourRequests:      rl.hourRequests,
		HourRequestsMax:   rl.requestsPerHour,
		MinuteTokens:      rl.minuteTokens,
		MinuteTokensMax:   rl.tokensPerMinute,
		HourTokens:        rl.hourTokens,
		HourTokensMax:     rl.tokensPerHour,
		BurstAvailable:    int(rl.tokens),
		BurstMax:          rl.burstSize,
	}
}

// RateLimitStatus represents the current state of rate limits.
type RateLimitStatus struct {
	MinuteRequests    int
	MinuteRequestsMax int
	HourRequests      int
	HourRequestsMax   int
	MinuteTokens      int
	MinuteTokensMax   int
	HourTokens        int
	HourTokensMax     int
	BurstAvailable    int
	BurstMax          int
}

// Reset resets all rate limit counters.
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rl.minuteRequests = 0
	rl.hourRequests = 0
	rl.minuteTokens = 0
	rl.hourTokens = 0
	rl.lastMinuteReset = now
	rl.lastHourReset = now
	rl.tokens = float64(rl.burstSize)
	rl.lastRefill = now
}

// RecordTokens records token usage after a successful request.
func (rl *RateLimiter) RecordTokens(tokens int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.minuteTokens += tokens
	rl.hourTokens += tokens
}

// SlidingWindowLimiter provides sliding window rate limiting.
type SlidingWindowLimiter struct {
	windowSize time.Duration
	maxCount   int
	timestamps []time.Time
	mu         sync.Mutex
}

// NewSlidingWindowLimiter creates a new sliding window rate limiter.
func NewSlidingWindowLimiter(windowSize time.Duration, maxCount int) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		windowSize: windowSize,
		maxCount:   maxCount,
		timestamps: make([]time.Time, 0),
	}
}

// Allow checks if a request is allowed.
func (sw *SlidingWindowLimiter) Allow() bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-sw.windowSize)

	// Remove old timestamps
	validIdx := 0
	for i, ts := range sw.timestamps {
		if ts.After(windowStart) {
			validIdx = i
			break
		}
	}
	if validIdx > 0 {
		sw.timestamps = sw.timestamps[validIdx:]
	}

	// Check limit
	if len(sw.timestamps) >= sw.maxCount {
		return false
	}

	// Add new timestamp
	sw.timestamps = append(sw.timestamps, now)
	return true
}

// Count returns the current count within the window.
func (sw *SlidingWindowLimiter) Count() int {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-sw.windowSize)

	count := 0
	for _, ts := range sw.timestamps {
		if ts.After(windowStart) {
			count++
		}
	}
	return count
}

// TimeUntilAvailable returns how long to wait until a request is allowed.
func (sw *SlidingWindowLimiter) TimeUntilAvailable() time.Duration {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-sw.windowSize)

	// Count valid timestamps
	validCount := 0
	var oldestValid time.Time
	for _, ts := range sw.timestamps {
		if ts.After(windowStart) {
			if validCount == 0 {
				oldestValid = ts
			}
			validCount++
		}
	}

	if validCount < sw.maxCount {
		return 0
	}

	// Wait until oldest timestamp expires
	return oldestValid.Add(sw.windowSize).Sub(now)
}
