package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	gohttp "net/http"
	"os"
	"strconv"
	"time"

	"github.com/teracrafts/huefy-go/config"
	sdkerrors "github.com/teracrafts/huefy-go/errors"
	"github.com/teracrafts/huefy-go/internal/version"
	"github.com/teracrafts/huefy-go/security"
	"github.com/teracrafts/huefy-go/types"
)

const (
	// BASE_URL is the production API base URL.
	BASE_URL = "https://api.huefy.dev/api/v1/sdk"

	// LOCAL_BASE_URL is the local development API base URL.
	LOCAL_BASE_URL = "https://api.huefy.on/api/v1/sdk"
)

// GetBaseURL returns the appropriate base URL based on the HUEFY_MODE
// environment variable. If set to "local", it returns the local base URL;
// otherwise, it returns the production base URL.
func GetBaseURL() string {
	mode := os.Getenv("HUEFY_MODE")
	if mode == "local" {
		return LOCAL_BASE_URL
	}
	return BASE_URL
}

// Client is the internal HTTP client that handles request execution, retries,
// circuit breaking, and request signing.
type Client struct {
	httpClient     *gohttp.Client
	apiKey         string
	config         *config.Config
	circuitBreaker *CircuitBreaker
	logger         types.Logger
}

// NewClient creates a new internal HTTP client from the given API key and config.
func NewClient(apiKey string, cfg *config.Config) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = GetBaseURL()
		cfg.BaseURL = baseURL
	}

	transport := gohttp.DefaultTransport.(*gohttp.Transport).Clone()
	transport.MaxIdleConnsPerHost = 20

	httpClient := &gohttp.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
	}

	cb := NewCircuitBreaker(
		cfg.CircuitBreakerConfig.FailureThreshold,
		cfg.CircuitBreakerConfig.ResetTimeout,
		cfg.CircuitBreakerConfig.HalfOpenRequests,
	)

	logger := cfg.Logger
	if logger == nil {
		logger = types.NewNoopLogger()
	}

	return &Client{
		httpClient:     httpClient,
		apiKey:         apiKey,
		config:         cfg,
		circuitBreaker: cb,
		logger:         logger,
	}
}

// Request executes an HTTP request with retry logic, circuit breaking, and
// optional request signing. It returns the raw response body on success.
func (c *Client) Request(ctx context.Context, method, path string, body any) ([]byte, error) {
	currentKey := c.apiKey
	attempted401Rotation := false

	// Capture the response body from the successful attempt inside the retry loop.
	var responseBody []byte

	retryFn := func() error {
		var innerBody []byte
		var innerErr error

		err := c.circuitBreaker.Execute(func() error {
			innerBody, innerErr = c.doRequestFull(ctx, method, path, body, currentKey)
			return innerErr
		})

		if err != nil {
			// Check for 401 and attempt key rotation.
			if sdkErr, ok := err.(*sdkerrors.HuefyError); ok {
				if sdkErr.StatusCode == 401 && !attempted401Rotation && c.config.SecondaryAPIKey != "" {
					c.logger.Warn("received 401, rotating to secondary API key")
					currentKey = c.config.SecondaryAPIKey
					attempted401Rotation = true
					// Return a new recoverable error to trigger retry (don't mutate original).
					return &sdkerrors.HuefyError{
						Code:        sdkErr.Code,
						Message:     sdkErr.Message,
						Recoverable: true,
						StatusCode:  sdkErr.StatusCode,
						RequestID:   sdkErr.RequestID,
						Timestamp:   sdkErr.Timestamp,
						Details:     sdkErr.Details,
						RetryAfter:  sdkErr.RetryAfter,
					}
				}
			}
			return err
		}

		responseBody = innerBody
		return nil
	}

	// Use the retry wrapper.
	err := WithRetry(ctx, retryFn, &c.config.RetryConfig, c.logger)
	if err != nil {
		if c.config.EnableErrorSanitization {
			if sdkErr, ok := err.(*sdkerrors.HuefyError); ok {
				sdkErr.Message = sdkerrors.SanitizeErrorMessage(sdkErr.Message, nil)
				return nil, sdkErr
			}
		}
		return nil, err
	}

	return responseBody, nil
}

// doRequestFull performs a single HTTP request and returns the response body.
func (c *Client) doRequestFull(ctx context.Context, method, path string, body any, apiKey string) ([]byte, error) {
	url := c.config.BaseURL + path

	var bodyReader io.Reader
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, sdkerrors.NewErrorWithCause(sdkerrors.ErrValidationFailed, "failed to marshal request body", err)
		}
		// Normalize to sorted-key JSON so HMAC signatures are consistent across
		// SDKs regardless of struct field order (Go) vs sort_keys=True (Python).
		var canonical map[string]any
		if jsonErr := json.Unmarshal(bodyBytes, &canonical); jsonErr == nil {
			if sorted, sortErr := json.Marshal(canonical); sortErr == nil {
				bodyBytes = sorted
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := gohttp.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, sdkerrors.NewErrorWithCause(sdkerrors.ErrNetworkConnection, "failed to create request", err)
	}

	// Set standard headers.
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("User-Agent", fmt.Sprintf("huefy-go/%s", version.GetVersion()))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Optional HMAC request signing.
	if c.config.EnableRequestSigning && bodyBytes != nil {
		sig := security.CreateRequestSignature(string(bodyBytes), apiKey)
		req.Header.Set("X-Signature", sig.Signature)
		req.Header.Set("X-Timestamp", strconv.FormatInt(sig.Timestamp, 10))
		req.Header.Set("X-Key-Id", sig.KeyID)
	}

	c.logger.Debug(fmt.Sprintf("request: %s %s", method, url))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, sdkerrors.NetworkError(fmt.Sprintf("request failed: %s %s", method, url), err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, sdkerrors.NetworkError("failed to read response body", err)
	}

	if resp.StatusCode >= 400 {
		sdkErr := c.buildErrorFromResponse(resp, respBody)
		return nil, sdkErr
	}

	c.parseRateLimitHeaders(resp.Header)
	c.logger.Debug(fmt.Sprintf("response: %d %s", resp.StatusCode, url))
	return respBody, nil
}

// buildErrorFromResponse constructs an HuefyError from an HTTP error response.
func (c *Client) buildErrorFromResponse(resp *gohttp.Response, body []byte) *sdkerrors.HuefyError {
	code := sdkerrors.ErrNetworkConnection
	recoverable := false
	message := fmt.Sprintf("API request failed with status %d", resp.StatusCode)

	switch {
	case resp.StatusCode == 401:
		code = sdkerrors.ErrAuthFailed
		message = "authentication failed: invalid API key"
	case resp.StatusCode == 403:
		code = sdkerrors.ErrAuthFailed
		message = "authorization failed: insufficient permissions"
	case resp.StatusCode == 404:
		code = sdkerrors.ErrValidationFailed
		message = "resource not found"
	case resp.StatusCode == 408:
		code = sdkerrors.ErrNetworkTimeout
		message = "request timeout"
		recoverable = true
	case resp.StatusCode == 429:
		code = sdkerrors.ErrRateLimited
		message = "rate limited"
		recoverable = true
	case resp.StatusCode >= 500:
		code = sdkerrors.ErrServerError
		message = fmt.Sprintf("server error: %d", resp.StatusCode)
		recoverable = true
	}

	// Try to extract message from response body.
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err == nil {
		if msg, ok := parsed["message"].(string); ok {
			message = msg
		}
		if msg, ok := parsed["error"].(string); ok {
			message = msg
		}
	}

	sdkErr := &sdkerrors.HuefyError{
		Code:        code,
		Message:     message,
		Recoverable: recoverable,
		StatusCode:  resp.StatusCode,
		RequestID:   resp.Header.Get("X-Request-Id"),
		Timestamp:   time.Now(),
		Details:     make(map[string]any),
	}

	// Parse Retry-After header if present.
	if retryAfter, ok := ParseRetryAfter(resp.Header.Get("Retry-After")); ok {
		sdkErr.RetryAfter = retryAfter
	}

	return sdkErr
}

// parseRateLimitHeaders reads X-RateLimit-* headers and fires the configured
// callbacks when all three headers are present.
func (c *Client) parseRateLimitHeaders(headers gohttp.Header) {
	limitStr := headers.Get("X-RateLimit-Limit")
	remainingStr := headers.Get("X-RateLimit-Remaining")
	resetStr := headers.Get("X-RateLimit-Reset")

	if limitStr == "" || remainingStr == "" || resetStr == "" {
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return
	}
	remaining, err := strconv.Atoi(remainingStr)
	if err != nil {
		return
	}
	resetUnix, err := strconv.ParseInt(resetStr, 10, 64)
	if err != nil {
		return
	}

	info := config.RateLimitInfo{
		Limit:     limit,
		Remaining: remaining,
		ResetAt:   time.Unix(resetUnix, 0),
	}

	if c.config.OnRateLimitUpdate != nil {
		c.config.OnRateLimitUpdate(info)
	}
	if c.config.OnRateLimitWarning != nil && limit > 0 && remaining < limit/5 {
		c.config.OnRateLimitWarning(info)
	}
}

// Close releases any resources held by the HTTP client.
func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}
