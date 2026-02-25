package tests

import (
	"errors"
	"testing"
	"time"

	internalhttp "github.com/teracrafts/huefy-go/internal/http"
)

func TestCircuitBreakerStartsClosed(t *testing.T) {
	cb := internalhttp.NewCircuitBreaker(3, 1*time.Second, 1)
	if state := cb.GetState(); state != internalhttp.StateClosed {
		t.Errorf("expected state CLOSED, got %s", state)
	}
}

func TestCircuitBreakerOpensAfterThreshold(t *testing.T) {
	cb := internalhttp.NewCircuitBreaker(3, 1*time.Second, 1)
	testErr := errors.New("test error")

	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	if state := cb.GetState(); state != internalhttp.StateOpen {
		t.Errorf("expected state OPEN after %d failures, got %s", 3, state)
	}
}

func TestCircuitBreakerRejectsWhenOpen(t *testing.T) {
	cb := internalhttp.NewCircuitBreaker(2, 5*time.Second, 1)
	testErr := errors.New("test error")

	// Trip the circuit.
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	// Next call should be rejected.
	err := cb.Execute(func() error {
		t.Error("function should not have been called when circuit is open")
		return nil
	})

	if err == nil {
		t.Error("expected error when circuit is open, got nil")
	}

	var circuitErr *internalhttp.CircuitOpenError
	if !errors.As(err, &circuitErr) {
		t.Errorf("expected CircuitOpenError, got %T", err)
	}
}

func TestCircuitBreakerTransitionsToHalfOpen(t *testing.T) {
	cb := internalhttp.NewCircuitBreaker(2, 100*time.Millisecond, 1)
	testErr := errors.New("test error")

	// Trip the circuit.
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	// Wait for reset timeout.
	time.Sleep(150 * time.Millisecond)

	if state := cb.GetState(); state != internalhttp.StateHalfOpen {
		t.Errorf("expected state HALF_OPEN after reset timeout, got %s", state)
	}
}

func TestCircuitBreakerClosesOnHalfOpenSuccess(t *testing.T) {
	cb := internalhttp.NewCircuitBreaker(2, 100*time.Millisecond, 1)
	testErr := errors.New("test error")

	// Trip the circuit.
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	// Wait for reset timeout.
	time.Sleep(150 * time.Millisecond)

	// Successful call should close the circuit.
	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("expected nil error on half-open success, got %v", err)
	}

	if state := cb.GetState(); state != internalhttp.StateClosed {
		t.Errorf("expected state CLOSED after half-open success, got %s", state)
	}
}

func TestCircuitBreakerReopensOnHalfOpenFailure(t *testing.T) {
	cb := internalhttp.NewCircuitBreaker(2, 100*time.Millisecond, 1)
	testErr := errors.New("test error")

	// Trip the circuit.
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	// Wait for reset timeout.
	time.Sleep(150 * time.Millisecond)

	// Failing call in half-open should reopen.
	_ = cb.Execute(func() error {
		return testErr
	})

	if state := cb.GetState(); state != internalhttp.StateOpen {
		t.Errorf("expected state OPEN after half-open failure, got %s", state)
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	cb := internalhttp.NewCircuitBreaker(2, 1*time.Second, 1)
	testErr := errors.New("test error")

	// Trip the circuit.
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	cb.Reset()

	if state := cb.GetState(); state != internalhttp.StateClosed {
		t.Errorf("expected state CLOSED after reset, got %s", state)
	}

	stats := cb.GetStats()
	if stats.Failures != 0 {
		t.Errorf("expected 0 failures after reset, got %d", stats.Failures)
	}
}

func TestCircuitBreakerGetStats(t *testing.T) {
	cb := internalhttp.NewCircuitBreaker(5, 1*time.Second, 1)

	// Execute a few successful calls.
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error {
			return nil
		})
	}

	stats := cb.GetStats()
	if stats.Successes != 3 {
		t.Errorf("expected 3 successes, got %d", stats.Successes)
	}
	if stats.TotalRequests != 3 {
		t.Errorf("expected 3 total requests, got %d", stats.TotalRequests)
	}
}

func TestCircuitBreakerSuccessResetsFailureCount(t *testing.T) {
	cb := internalhttp.NewCircuitBreaker(3, 1*time.Second, 1)
	testErr := errors.New("test error")

	// Record 2 failures (below threshold).
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	// A success should reset the failure count.
	_ = cb.Execute(func() error {
		return nil
	})

	// Now 2 more failures should NOT trip the circuit (count was reset).
	for i := 0; i < 2; i++ {
		_ = cb.Execute(func() error {
			return testErr
		})
	}

	if state := cb.GetState(); state != internalhttp.StateClosed {
		t.Errorf("expected state CLOSED (failures reset by success), got %s", state)
	}
}
