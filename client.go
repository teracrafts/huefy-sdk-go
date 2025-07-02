// Package huefy provides a Go SDK for the Huefy email sending platform.
//
// The Huefy Go SDK allows you to send template-based emails through the Huefy API
// with support for multiple email providers, retry logic, and comprehensive error handling.
//
// Basic usage:
//
//	client := huefy.NewClient("your-api-key")
//	resp, err := client.SendEmail(context.Background(), &huefy.SendEmailRequest{
//		TemplateKey: "welcome-email",
//		Data: map[string]interface{}{
//			"name":    "John Doe",
//			"company": "Acme Corp",
//		},
//		Recipient: "john@example.com",
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Email sent: %s\n", resp.MessageID)
package huefy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// EmailProvider represents supported email providers
type EmailProvider string

const (
	ProviderSES      EmailProvider = "ses"
	ProviderSendGrid EmailProvider = "sendgrid"
	ProviderMailgun  EmailProvider = "mailgun"
	ProviderMailchimp EmailProvider = "mailchimp"
)

// Client represents the Huefy SDK client
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	retryConfig *RetryConfig
}

// RetryConfig configures retry behavior for failed requests
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Multiplier float64
}

// DefaultRetryConfig provides sensible defaults for retry behavior
var DefaultRetryConfig = &RetryConfig{
	MaxRetries: 3,
	BaseDelay:  time.Second,
	MaxDelay:   30 * time.Second,
	Multiplier: 2.0,
}

// ClientOption configures a Client
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL for the API
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = strings.TrimSuffix(baseURL, "/")
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithRetryConfig sets custom retry configuration
func WithRetryConfig(config *RetryConfig) ClientOption {
	return func(c *Client) {
		c.retryConfig = config
	}
}

// NewClient creates a new Huefy SDK client
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:      apiKey,
		baseURL:     "https://api.huefy.com",
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		retryConfig: DefaultRetryConfig,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// SendEmailRequest represents a request to send an email
type SendEmailRequest struct {
	TemplateKey string                 `json:"templateKey"`
	Data        map[string]interface{} `json:"data"`
	Recipient   string                 `json:"recipient"`
	Provider    *EmailProvider         `json:"providerType,omitempty"`
}

// SendEmailResponse represents the response from sending an email
type SendEmailResponse struct {
	Success   bool          `json:"success"`
	Message   string        `json:"message"`
	MessageID string        `json:"messageId"`
	Provider  EmailProvider `json:"provider"`
}

// BulkEmailRequest represents a request to send multiple emails
type BulkEmailRequest struct {
	Emails []SendEmailRequest `json:"emails"`
}

// BulkEmailResult represents the result of a single email in a bulk operation
type BulkEmailResult struct {
	Success bool                `json:"success"`
	Result  *SendEmailResponse  `json:"result,omitempty"`
	Error   *ErrorResponse      `json:"error,omitempty"`
}

// BulkEmailResponse represents the response from sending multiple emails
type BulkEmailResponse struct {
	Results []BulkEmailResult `json:"results"`
}

// HealthResponse represents the API health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version,omitempty"`
}

// SendEmail sends a single email using a template
func (c *Client) SendEmail(ctx context.Context, req *SendEmailRequest) (*SendEmailResponse, error) {
	if req == nil {
		return nil, NewValidationError("request cannot be nil")
	}

	if err := c.validateSendEmailRequest(req); err != nil {
		return nil, err
	}

	var resp SendEmailResponse
	err := c.doRequest(ctx, "POST", "/api/v1/sdk/emails/send", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// SendBulkEmails sends multiple emails in a single request
func (c *Client) SendBulkEmails(ctx context.Context, emails []SendEmailRequest) (*BulkEmailResponse, error) {
	if len(emails) == 0 {
		return nil, NewValidationError("emails slice cannot be empty")
	}

	// Validate each email request
	for i, email := range emails {
		if err := c.validateSendEmailRequest(&email); err != nil {
			return nil, fmt.Errorf("validation failed for email %d: %w", i, err)
		}
	}

	req := &BulkEmailRequest{Emails: emails}
	var resp BulkEmailResponse
	err := c.doRequest(ctx, "POST", "/api/v1/sdk/emails/bulk", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// HealthCheck checks the API health status
func (c *Client) HealthCheck(ctx context.Context) (*HealthResponse, error) {
	var resp HealthResponse
	err := c.doRequest(ctx, "GET", "/api/v1/sdk/health", nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// validateSendEmailRequest validates a send email request
func (c *Client) validateSendEmailRequest(req *SendEmailRequest) error {
	if req.TemplateKey == "" {
		return NewValidationError("templateKey is required")
	}

	if req.Recipient == "" {
		return NewValidationError("recipient is required")
	}

	// Basic email validation
	if !strings.Contains(req.Recipient, "@") || !strings.Contains(req.Recipient, ".") {
		return NewInvalidRecipientError(fmt.Sprintf("invalid email address: %s", req.Recipient))
	}

	if req.Data == nil {
		return NewValidationError("data is required")
	}

	// Validate provider if specified
	if req.Provider != nil {
		switch *req.Provider {
		case ProviderSES, ProviderSendGrid, ProviderMailgun, ProviderMailchimp:
			// Valid provider
		default:
			return NewValidationError(fmt.Sprintf("invalid provider: %s", *req.Provider))
		}
	}

	return nil
}

// doRequest performs an HTTP request with retry logic
func (c *Client) doRequest(ctx context.Context, method, path string, reqBody, respBody interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff
			delay := time.Duration(float64(c.retryConfig.BaseDelay) * 
				fmt.Sprintf("%.0f", c.retryConfig.Multiplier*float64(attempt-1)))
			if delay > c.retryConfig.MaxDelay {
				delay = c.retryConfig.MaxDelay
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := c.performRequest(ctx, method, path, reqBody, respBody)
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry certain errors
		if !isRetryableError(err) {
			return err
		}
	}

	return lastErr
}

// performRequest performs a single HTTP request
func (c *Client) performRequest(ctx context.Context, method, path string, reqBody, respBody interface{}) error {
	// Build URL
	u, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return fmt.Errorf("failed to build URL: %w", err)
	}

	// Prepare request body
	var body io.Reader
	if reqBody != nil {
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Huefy-Go-SDK/1.0.0")

	// Perform request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return NewNetworkError(fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	// Read response body
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		var errorResp ErrorResponse
		if err := json.Unmarshal(respData, &errorResp); err != nil {
			// Fallback to generic error
			errorResp = ErrorResponse{
				Error: ErrorDetail{
					Code:    fmt.Sprintf("HTTP_%d", resp.StatusCode),
					Message: string(respData),
				},
			}
		}
		return createErrorFromResponse(&errorResp, resp.StatusCode)
	}

	// Parse success response
	if respBody != nil {
		if err := json.Unmarshal(respData, respBody); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	switch e := err.(type) {
	case *NetworkError, *TimeoutError:
		return true
	case *RateLimitError:
		return true
	case *HuefyError:
		// Retry on 5xx server errors
		return e.Code == "INTERNAL_SERVER_ERROR" || 
			   e.Code == "SERVICE_UNAVAILABLE" ||
			   e.Code == "BAD_GATEWAY"
	default:
		return false
	}
}