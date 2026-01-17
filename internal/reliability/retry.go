package reliability

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryableError indicates an error that can be retried.
type RetryableError struct {
	Err       error
	Retryable bool
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryable checks if an error is retryable.
func IsRetryable(err error) bool {
	var retryErr *RetryableError
	if errors.As(err, &retryErr) {
		return retryErr.Retryable
	}
	// Default: network errors, timeouts are retryable
	return isTemporaryError(err)
}

func isTemporaryError(err error) bool {
	if err == nil {
		return false
	}
	// Check for common temporary error patterns
	errStr := err.Error()
	temporaryPatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"service unavailable",
		"too many requests",
		"rate limit",
		"ECONNREFUSED",
		"ETIMEDOUT",
		"ECONNRESET",
	}
	for _, pattern := range temporaryPatterns {
		if contains(errStr, pattern) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// BackoffStrategy defines the backoff algorithm.
type BackoffStrategy int

const (
	// ExponentialBackoff increases delay exponentially.
	ExponentialBackoff BackoffStrategy = iota
	// LinearBackoff increases delay linearly.
	LinearBackoff
	// ConstantBackoff uses a constant delay.
	ConstantBackoff
)

// RetryConfig contains configuration for retry behavior.
type RetryConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	Multiplier      float64
	Jitter          float64 // 0.0 to 1.0, adds randomness to delay
	Strategy        BackoffStrategy
	RetryableErrors []error // Specific errors to retry on
}

// DefaultRetryConfig returns sensible defaults for retry behavior.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:   3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
		Strategy:     ExponentialBackoff,
	}
}

// Retryer provides retry functionality with configurable backoff.
type Retryer struct {
	config   RetryConfig
	onRetry  func(attempt int, err error, delay time.Duration)
	randSrc  *rand.Rand
}

// NewRetryer creates a new retryer with the given configuration.
func NewRetryer(config RetryConfig) *Retryer {
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}
	if config.InitialDelay <= 0 {
		config.InitialDelay = time.Second
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 30 * time.Second
	}
	if config.Multiplier <= 0 {
		config.Multiplier = 2.0
	}

	return &Retryer{
		config:  config,
		randSrc: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// OnRetry sets a callback that's called before each retry.
func (r *Retryer) OnRetry(fn func(attempt int, err error, delay time.Duration)) {
	r.onRetry = fn
}

// Do executes the function with retry logic.
func (r *Retryer) Do(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// Execute the function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !r.shouldRetry(err) {
			return err
		}

		// Check if we've exhausted retries
		if attempt >= r.config.MaxRetries {
			return fmt.Errorf("max retries (%d) exceeded: %w", r.config.MaxRetries, lastErr)
		}

		// Calculate delay
		delay := r.calculateDelay(attempt)

		// Call retry callback
		if r.onRetry != nil {
			r.onRetry(attempt+1, err, delay)
		}

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next retry
		}
	}

	return lastErr
}

// DoWithResult executes a function that returns a value with retry logic.
func (r *Retryer) DoWithResult(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	var result interface{}
	err := r.Do(ctx, func() error {
		var err error
		result, err = fn()
		return err
	})
	return result, err
}

func (r *Retryer) shouldRetry(err error) bool {
	// Check if error is in the specific retryable errors list
	for _, retryableErr := range r.config.RetryableErrors {
		if errors.Is(err, retryableErr) {
			return true
		}
	}

	// Check if error is generally retryable
	return IsRetryable(err)
}

func (r *Retryer) calculateDelay(attempt int) time.Duration {
	var delay time.Duration

	switch r.config.Strategy {
	case ExponentialBackoff:
		delay = time.Duration(float64(r.config.InitialDelay) * math.Pow(r.config.Multiplier, float64(attempt)))
	case LinearBackoff:
		delay = r.config.InitialDelay * time.Duration(attempt+1)
	case ConstantBackoff:
		delay = r.config.InitialDelay
	default:
		delay = r.config.InitialDelay
	}

	// Apply jitter
	if r.config.Jitter > 0 {
		jitterRange := float64(delay) * r.config.Jitter
		jitter := (r.randSrc.Float64()*2 - 1) * jitterRange
		delay = time.Duration(float64(delay) + jitter)
	}

	// Cap at max delay
	if delay > r.config.MaxDelay {
		delay = r.config.MaxDelay
	}

	// Ensure non-negative
	if delay < 0 {
		delay = 0
	}

	return delay
}

// RetryResult contains information about a retry operation.
type RetryResult struct {
	Attempts   int
	TotalTime  time.Duration
	LastError  error
	Successful bool
}

// DoWithStats executes with retry and returns statistics.
func (r *Retryer) DoWithStats(ctx context.Context, fn func() error) RetryResult {
	startTime := time.Now()
	result := RetryResult{}

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		result.Attempts = attempt + 1

		err := fn()
		if err == nil {
			result.Successful = true
			result.TotalTime = time.Since(startTime)
			return result
		}

		result.LastError = err

		if !r.shouldRetry(err) || attempt >= r.config.MaxRetries {
			break
		}

		delay := r.calculateDelay(attempt)
		if r.onRetry != nil {
			r.onRetry(attempt+1, err, delay)
		}

		select {
		case <-ctx.Done():
			result.LastError = ctx.Err()
			result.TotalTime = time.Since(startTime)
			return result
		case <-time.After(delay):
		}
	}

	result.TotalTime = time.Since(startTime)
	return result
}

// Circuit breaker states
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CircuitBreaker prevents repeated calls to a failing service.
type CircuitBreaker struct {
	maxFailures     int
	resetTimeout    time.Duration
	halfOpenMax     int
	failures        int
	successes       int
	state           CircuitState
	lastFailureTime time.Time
	onStateChange   func(from, to CircuitState)
}

// CircuitBreakerConfig contains configuration for the circuit breaker.
type CircuitBreakerConfig struct {
	MaxFailures  int
	ResetTimeout time.Duration
	HalfOpenMax  int
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: 30 * time.Second,
		HalfOpenMax:  3,
	}
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.MaxFailures <= 0 {
		config.MaxFailures = 5
	}
	if config.ResetTimeout <= 0 {
		config.ResetTimeout = 30 * time.Second
	}
	if config.HalfOpenMax <= 0 {
		config.HalfOpenMax = 3
	}

	return &CircuitBreaker{
		maxFailures:  config.MaxFailures,
		resetTimeout: config.ResetTimeout,
		halfOpenMax:  config.HalfOpenMax,
		state:        CircuitClosed,
	}
}

// OnStateChange sets a callback for state changes.
func (cb *CircuitBreaker) OnStateChange(fn func(from, to CircuitState)) {
	cb.onStateChange = fn
}

// Execute runs the function through the circuit breaker.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.allowRequest() {
		return fmt.Errorf("circuit breaker is open")
	}

	err := fn()
	cb.recordResult(err)
	return err
}

func (cb *CircuitBreaker) allowRequest() bool {
	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if reset timeout has passed
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.transitionTo(CircuitHalfOpen)
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

func (cb *CircuitBreaker) recordResult(err error) {
	if err == nil {
		cb.recordSuccess()
	} else {
		cb.recordFailure()
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	switch cb.state {
	case CircuitClosed:
		cb.failures = 0
	case CircuitHalfOpen:
		cb.successes++
		if cb.successes >= cb.halfOpenMax {
			cb.transitionTo(CircuitClosed)
		}
	}
}

func (cb *CircuitBreaker) recordFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitClosed:
		if cb.failures >= cb.maxFailures {
			cb.transitionTo(CircuitOpen)
		}
	case CircuitHalfOpen:
		cb.transitionTo(CircuitOpen)
	}
}

func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	oldState := cb.state
	cb.state = newState
	cb.failures = 0
	cb.successes = 0

	if cb.onStateChange != nil {
		cb.onStateChange(oldState, newState)
	}
}

// State returns the current circuit breaker state.
func (cb *CircuitBreaker) State() CircuitState {
	return cb.state
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.transitionTo(CircuitClosed)
}

// String returns a string representation of the circuit state.
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}
