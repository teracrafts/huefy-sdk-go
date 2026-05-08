package config

import (
	"net/http"
	"time"

	"github.com/teracrafts/huefy-go/types"
)

// RateLimitInfo holds the parsed rate limit state from response headers.
type RateLimitInfo struct {
	// Limit is the total number of requests allowed in the current window.
	Limit int

	// Remaining is the number of requests remaining in the current window.
	Remaining int

	// ResetAt is the time when the rate limit window resets.
	ResetAt time.Time
}

// Config holds all configuration for the Huefy client.
type Config struct {
	// APIKey is the primary API key for authentication.
	APIKey string

	// BaseURL is the base URL of the Huefy API.
	BaseURL string

	// HTTPTransport overrides the default HTTP transport. Intended for
	// deterministic integration harnesses such as the SDK lab.
	HTTPTransport http.RoundTripper

	// Timeout is the HTTP request timeout duration.
	Timeout time.Duration

	// RetryConfig configures the retry behavior.
	RetryConfig RetryConfig

	// CircuitBreakerConfig configures the circuit breaker behavior.
	CircuitBreakerConfig CircuitBreakerConfig

	// Logger is the logger used by the SDK.
	Logger types.Logger

	// SecondaryAPIKey is an optional secondary API key used for key rotation.
	SecondaryAPIKey string

	// EnableRequestSigning enables HMAC request signing.
	EnableRequestSigning bool

	// EnableErrorSanitization enables sanitization of sensitive data in error messages.
	EnableErrorSanitization bool

	// OnRateLimitUpdate is called after each response when rate limit headers are present.
	OnRateLimitUpdate func(RateLimitInfo)

	// OnRateLimitWarning is called when remaining requests fall below 20% of the limit.
	OnRateLimitWarning func(RateLimitInfo)
}

// RetryConfig configures the retry behavior for failed requests.
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts. Default: 3.
	MaxRetries int

	// BaseDelay is the base delay between retries. Default: 1s.
	BaseDelay time.Duration

	// MaxDelay is the maximum delay between retries. Default: 30s.
	MaxDelay time.Duration

	// RetryableStatusCodes is the list of HTTP status codes that trigger a retry.
	RetryableStatusCodes []int
}

// CircuitBreakerConfig configures the circuit breaker behavior.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures before the circuit opens. Default: 5.
	FailureThreshold int

	// ResetTimeout is the duration the circuit stays open before transitioning to half-open. Default: 30s.
	ResetTimeout time.Duration

	// HalfOpenRequests is the number of requests allowed in half-open state. Default: 1.
	HalfOpenRequests int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig(apiKey string) Config {
	return Config{
		APIKey:  apiKey,
		Timeout: 30 * time.Second,
		RetryConfig: RetryConfig{
			MaxRetries: 3,
			BaseDelay:  500 * time.Millisecond,
			MaxDelay:   10 * time.Second,
			RetryableStatusCodes: []int{408, 429, 500, 502, 503, 504},
		},
		CircuitBreakerConfig: CircuitBreakerConfig{
			FailureThreshold: 5,
			ResetTimeout:     30 * time.Second,
			HalfOpenRequests: 1,
		},
		Logger:                  types.NewNoopLogger(),
		EnableRequestSigning:    false,
		EnableErrorSanitization: false,
	}
}

// Option is a functional option for configuring the Huefy client.
type Option func(*Config)

// WithBaseURL sets the base URL for the API client.
func WithBaseURL(url string) Option {
	return func(c *Config) {
		c.BaseURL = url
	}
}

// WithTimeout sets the HTTP request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithHTTPTransport sets a custom HTTP transport for the underlying client.
func WithHTTPTransport(transport http.RoundTripper) Option {
	return func(c *Config) {
		c.HTTPTransport = transport
	}
}

// WithRetryConfig sets the retry configuration.
func WithRetryConfig(rc RetryConfig) Option {
	return func(c *Config) {
		c.RetryConfig = rc
	}
}

// WithCircuitBreakerConfig sets the circuit breaker configuration.
func WithCircuitBreakerConfig(cbc CircuitBreakerConfig) Option {
	return func(c *Config) {
		c.CircuitBreakerConfig = cbc
	}
}

// WithLogger sets the logger for the client.
func WithLogger(logger types.Logger) Option {
	return func(c *Config) {
		c.Logger = logger
	}
}

// WithSecondaryAPIKey sets a secondary API key for key rotation on 401 responses.
func WithSecondaryAPIKey(key string) Option {
	return func(c *Config) {
		c.SecondaryAPIKey = key
	}
}

// WithRequestSigning enables HMAC request signing for enhanced security.
func WithRequestSigning(enable bool) Option {
	return func(c *Config) {
		c.EnableRequestSigning = enable
	}
}

// WithErrorSanitization enables sanitization of sensitive data in error messages.
func WithErrorSanitization(enable bool) Option {
	return func(c *Config) {
		c.EnableErrorSanitization = enable
	}
}

// WithOnRateLimitUpdate sets a callback invoked after each response when rate
// limit headers are present.
func WithOnRateLimitUpdate(fn func(RateLimitInfo)) Option {
	return func(c *Config) {
		c.OnRateLimitUpdate = fn
	}
}

// WithOnRateLimitWarning sets a callback invoked when remaining requests fall
// below 20% of the limit.
func WithOnRateLimitWarning(fn func(RateLimitInfo)) Option {
	return func(c *Config) {
		c.OnRateLimitWarning = fn
	}
}

// Apply applies all options to the config.
func (c *Config) Apply(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}
