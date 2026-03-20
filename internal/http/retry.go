package http

import (
	"context"
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/teracrafts/huefy-go/config"
	sdkerrors "github.com/teracrafts/huefy-go/errors"
	"github.com/teracrafts/huefy-go/types"
)

// WithRetry executes fn with retry logic based on the provided RetryConfig.
// It retries on recoverable errors and respects context cancellation.
func WithRetry(ctx context.Context, fn func() error, cfg *config.RetryConfig, logger types.Logger) error {
	if logger == nil {
		logger = types.NewNoopLogger()
	}

	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := CalculateDelay(attempt, cfg.BaseDelay, cfg.MaxDelay)

			// If the last error has a RetryAfter hint, use it if it's longer.
			if sdkErr, ok := lastErr.(*sdkerrors.HuefyError); ok && sdkErr.RetryAfter > 0 {
				if sdkErr.RetryAfter > delay {
					delay = sdkErr.RetryAfter
				}
			}

			logger.Info("retrying request (attempt " + strconv.Itoa(attempt) + "/" + strconv.Itoa(cfg.MaxRetries) + ") after " + delay.String())

			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return sdkerrors.NewErrorWithCause(sdkerrors.ErrNetworkTimeout, "request cancelled during retry backoff", ctx.Err())
			case <-timer.C:
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if the error is recoverable.
		if !sdkerrors.IsRecoverable(err) {
			// Also check for CircuitOpenError which is not recoverable.
			if _, ok := err.(*CircuitOpenError); ok {
				return err
			}
			// Non-recoverable HuefyError.
			if _, ok := err.(*sdkerrors.HuefyError); ok {
				return err
			}
			// Unknown error type, don't retry.
			return err
		}

		logger.Warn("request failed (attempt " + strconv.Itoa(attempt+1) + "/" + strconv.Itoa(cfg.MaxRetries+1) + "): " + err.Error())
	}

	return lastErr
}

// CalculateDelay computes the retry delay for the given attempt using
// exponential backoff with ±20 % jitter. The formula is:
//
//	min(maxDelay, baseDelay * 2^attempt) * random(0.8, 1.2)
func CalculateDelay(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	// Exponential backoff.
	exp := math.Pow(2, float64(attempt))
	delay := time.Duration(float64(baseDelay) * exp)

	// Cap at max delay.
	if delay > maxDelay {
		delay = maxDelay
	}

	// Apply ±20% jitter (0.8 to 1.2 multiplier).
	jitter := 0.8 + rand.Float64()*0.4
	delay = time.Duration(float64(delay) * jitter)

	return delay
}

// ParseRetryAfter parses the value of a Retry-After HTTP header.
// It supports both delay-seconds (integer) and HTTP-date formats.
// Returns the parsed duration and true if parsing succeeded.
func ParseRetryAfter(header string) (time.Duration, bool) {
	if header == "" {
		return 0, false
	}

	// Try parsing as seconds first.
	if seconds, err := strconv.Atoi(header); err == nil {
		return time.Duration(seconds) * time.Second, true
	}

	// Try parsing as HTTP-date.
	if t, err := time.Parse(time.RFC1123, header); err == nil {
		delay := time.Until(t)
		if delay < 0 {
			delay = 0
		}
		return delay, true
	}

	return 0, false
}

// IsRetryableStatusCode checks whether the given HTTP status code is in the
// list of retryable status codes from the config.
func IsRetryableStatusCode(code int, cfg *config.RetryConfig) bool {
	for _, retryable := range cfg.RetryableStatusCodes {
		if code == retryable {
			return true
		}
	}
	return false
}
