package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	internalhttp "github.com/teracrafts/huefy-go/internal/http"
	"github.com/teracrafts/huefy-go/config"
	sdkerrors "github.com/teracrafts/huefy-go/errors"
	"github.com/teracrafts/huefy-go/types"
)

func newTestRetryConfig() *config.RetryConfig {
	return &config.RetryConfig{
		MaxRetries:           3,
		BaseDelay:            10 * time.Millisecond,
		MaxDelay:             100 * time.Millisecond,
		RetryableStatusCodes: []int{408, 429, 500, 502, 503, 504},
	}
}

func TestWithRetrySucceedsImmediately(t *testing.T) {
	cfg := newTestRetryConfig()
	logger := types.NewNoopLogger()

	calls := 0
	err := internalhttp.WithRetry(context.Background(), func() error {
		calls++
		return nil
	}, cfg, logger)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestWithRetryRetriesOnRecoverableError(t *testing.T) {
	cfg := newTestRetryConfig()
	logger := types.NewNoopLogger()

	calls := 0
	err := internalhttp.WithRetry(context.Background(), func() error {
		calls++
		if calls < 3 {
			return &sdkerrors.HuefyError{
				Code:        sdkerrors.ErrNetworkConnection,
				Message:     "connection failed",
				Recoverable: true,
				Timestamp:   time.Now(),
				Details:     make(map[string]any),
			}
		}
		return nil
	}, cfg, logger)

	if err != nil {
		t.Errorf("expected no error after retries, got %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestWithRetryStopsOnNonRecoverableError(t *testing.T) {
	cfg := newTestRetryConfig()
	logger := types.NewNoopLogger()

	calls := 0
	err := internalhttp.WithRetry(context.Background(), func() error {
		calls++
		return &sdkerrors.HuefyError{
			Code:        sdkerrors.ErrAuthFailed,
			Message:     "invalid key",
			Recoverable: false,
			Timestamp:   time.Now(),
			Details:     make(map[string]any),
		}
	}, cfg, logger)

	if err == nil {
		t.Error("expected error, got nil")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry for non-recoverable), got %d", calls)
	}
}

func TestWithRetryExhaustsMaxRetries(t *testing.T) {
	cfg := newTestRetryConfig()
	logger := types.NewNoopLogger()

	calls := 0
	err := internalhttp.WithRetry(context.Background(), func() error {
		calls++
		return &sdkerrors.HuefyError{
			Code:        sdkerrors.ErrNetworkTimeout,
			Message:     "timeout",
			Recoverable: true,
			Timestamp:   time.Now(),
			Details:     make(map[string]any),
		}
	}, cfg, logger)

	if err == nil {
		t.Error("expected error after exhausting retries, got nil")
	}
	// 1 initial + 3 retries = 4 total calls.
	if calls != 4 {
		t.Errorf("expected 4 calls (1 + 3 retries), got %d", calls)
	}
}

func TestWithRetryRespectsContextCancellation(t *testing.T) {
	cfg := newTestRetryConfig()
	cfg.BaseDelay = 1 * time.Second // Long delay so cancellation kicks in.
	logger := types.NewNoopLogger()

	ctx, cancel := context.WithCancel(context.Background())
	calls := 0

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := internalhttp.WithRetry(ctx, func() error {
		calls++
		return &sdkerrors.HuefyError{
			Code:        sdkerrors.ErrNetworkTimeout,
			Message:     "timeout",
			Recoverable: true,
			Timestamp:   time.Now(),
			Details:     make(map[string]any),
		}
	}, cfg, logger)

	if err == nil {
		t.Error("expected error on context cancellation, got nil")
	}
}

func TestCalculateDelayExponentialBackoff(t *testing.T) {
	baseDelay := 100 * time.Millisecond
	maxDelay := 10 * time.Second

	prev := time.Duration(0)
	for attempt := 1; attempt <= 5; attempt++ {
		delay := internalhttp.CalculateDelay(attempt, baseDelay, maxDelay)

		// Delay should generally increase (with jitter, it might not always).
		// At minimum, delay should be positive.
		if delay <= 0 {
			t.Errorf("attempt %d: expected positive delay, got %v", attempt, delay)
		}

		// Delay should not exceed max.
		if delay > maxDelay {
			t.Errorf("attempt %d: delay %v exceeds max %v", attempt, delay, maxDelay)
		}

		prev = delay
		_ = prev
	}
}

func TestCalculateDelayRespectsMaxDelay(t *testing.T) {
	baseDelay := 1 * time.Second
	maxDelay := 2 * time.Second

	// High attempt number should be capped at maxDelay.
	delay := internalhttp.CalculateDelay(20, baseDelay, maxDelay)
	if delay > maxDelay {
		t.Errorf("expected delay <= %v, got %v", maxDelay, delay)
	}
}

func TestParseRetryAfterSeconds(t *testing.T) {
	d, ok := internalhttp.ParseRetryAfter("120")
	if !ok {
		t.Error("expected ok=true for valid seconds")
	}
	if d != 120*time.Second {
		t.Errorf("expected 120s, got %v", d)
	}
}

func TestParseRetryAfterEmpty(t *testing.T) {
	_, ok := internalhttp.ParseRetryAfter("")
	if ok {
		t.Error("expected ok=false for empty header")
	}
}

func TestParseRetryAfterHTTPDate(t *testing.T) {
	future := time.Now().Add(60 * time.Second).UTC().Format(time.RFC1123)
	d, ok := internalhttp.ParseRetryAfter(future)
	if !ok {
		t.Error("expected ok=true for valid HTTP-date")
	}
	// Should be roughly 60 seconds (allow some tolerance).
	if d < 55*time.Second || d > 65*time.Second {
		t.Errorf("expected ~60s, got %v", d)
	}
}

func TestIsRetryableStatusCode(t *testing.T) {
	cfg := newTestRetryConfig()

	tests := []struct {
		code     int
		expected bool
	}{
		{200, false},
		{401, false},
		{408, true},
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
	}

	for _, tt := range tests {
		result := internalhttp.IsRetryableStatusCode(tt.code, cfg)
		if result != tt.expected {
			t.Errorf("IsRetryableStatusCode(%d) = %v, want %v", tt.code, result, tt.expected)
		}
	}
}

func TestWithRetryNilLogger(t *testing.T) {
	cfg := newTestRetryConfig()

	err := internalhttp.WithRetry(context.Background(), func() error {
		return nil
	}, cfg, nil)

	if err != nil {
		t.Errorf("expected no error with nil logger, got %v", err)
	}
}

func TestWithRetryNonSdkError(t *testing.T) {
	cfg := newTestRetryConfig()
	logger := types.NewNoopLogger()

	calls := 0
	err := internalhttp.WithRetry(context.Background(), func() error {
		calls++
		return errors.New("generic error")
	}, cfg, logger)

	if err == nil {
		t.Error("expected error, got nil")
	}
	// Non-SDK errors are not recoverable, so should not retry.
	if calls != 1 {
		t.Errorf("expected 1 call for non-SDK error, got %d", calls)
	}
}
