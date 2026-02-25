package http

import (
	"fmt"
	"sync"
	"time"
)

// State represents the state of a circuit breaker.
type State int

const (
	// StateClosed means the circuit is healthy and requests flow normally.
	StateClosed State = iota

	// StateOpen means the circuit has tripped and requests are blocked.
	StateOpen

	// StateHalfOpen means the circuit is testing whether the service has recovered.
	StateHalfOpen
)

// String returns a human-readable name for the circuit breaker state.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitOpenError is returned when the circuit breaker is open and a request
// is rejected without being attempted.
type CircuitOpenError struct {
	// ResetAt is the time when the circuit breaker will transition to half-open.
	ResetAt time.Time
}

// Error implements the error interface.
func (e *CircuitOpenError) Error() string {
	return fmt.Sprintf("circuit breaker is open, will reset at %s", e.ResetAt.Format(time.RFC3339))
}

// CircuitBreakerStats holds the current statistics of the circuit breaker.
type CircuitBreakerStats struct {
	State           State
	Failures        int
	Successes       int
	TotalRequests   int
	LastFailureTime time.Time
}

// CircuitBreaker implements the circuit breaker pattern to prevent cascading
// failures when a downstream service is unhealthy.
type CircuitBreaker struct {
	mu sync.Mutex

	state            State
	failureThreshold int
	resetTimeout     time.Duration
	halfOpenRequests int

	failureCount     int
	successCount     int
	totalRequests    int
	halfOpenAttempts int
	lastFailureTime  time.Time
	openedAt         time.Time
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration.
func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration, halfOpenRequests int) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		halfOpenRequests: halfOpenRequests,
	}
}

// Execute wraps a function call with circuit breaker logic. If the circuit is
// open, requests are rejected immediately. In half-open state, a limited number
// of requests are allowed through to test recovery.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()

	cb.totalRequests++

	switch cb.state {
	case StateOpen:
		// Check if enough time has passed to transition to half-open.
		if time.Since(cb.openedAt) >= cb.resetTimeout {
			cb.state = StateHalfOpen
			cb.halfOpenAttempts = 0
			cb.mu.Unlock()
			return cb.executeHalfOpen(fn)
		}
		resetAt := cb.openedAt.Add(cb.resetTimeout)
		cb.mu.Unlock()
		return &CircuitOpenError{ResetAt: resetAt}

	case StateHalfOpen:
		if cb.halfOpenAttempts >= cb.halfOpenRequests {
			resetAt := cb.openedAt.Add(cb.resetTimeout)
			cb.mu.Unlock()
			return &CircuitOpenError{ResetAt: resetAt}
		}
		cb.mu.Unlock()
		return cb.executeHalfOpen(fn)

	default: // StateClosed
		cb.mu.Unlock()
		return cb.executeClosed(fn)
	}
}

// executeClosed runs the function in the closed state and tracks failures.
func (cb *CircuitBreaker) executeClosed(fn func() error) error {
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		if cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
			cb.openedAt = time.Now()
		}
		return err
	}

	// Success resets failure count.
	cb.failureCount = 0
	cb.successCount++
	return nil
}

// executeHalfOpen runs the function in the half-open state.
func (cb *CircuitBreaker) executeHalfOpen(fn func() error) error {
	cb.mu.Lock()
	cb.halfOpenAttempts++
	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		// Failure in half-open state reopens the circuit.
		cb.state = StateOpen
		cb.openedAt = time.Now()
		cb.failureCount++
		cb.lastFailureTime = time.Now()
		return err
	}

	// Success in half-open state closes the circuit.
	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount++
	cb.halfOpenAttempts = 0
	return nil
}

// GetState returns the current state of the circuit breaker.
func (cb *CircuitBreaker) GetState() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check if open circuit should transition to half-open.
	if cb.state == StateOpen && time.Since(cb.openedAt) >= cb.resetTimeout {
		cb.state = StateHalfOpen
		cb.halfOpenAttempts = 0
	}

	return cb.state
}

// Reset resets the circuit breaker to its initial closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.totalRequests = 0
	cb.halfOpenAttempts = 0
	cb.lastFailureTime = time.Time{}
	cb.openedAt = time.Time{}
}

// GetStats returns the current statistics of the circuit breaker.
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return CircuitBreakerStats{
		State:           cb.state,
		Failures:        cb.failureCount,
		Successes:       cb.successCount,
		TotalRequests:   cb.totalRequests,
		LastFailureTime: cb.lastFailureTime,
	}
}
