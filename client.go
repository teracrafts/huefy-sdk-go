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
	"context"
	"net/http"
	"time"

	"github.com/teracrafts/huefy-sdk/core/kernel"
)

// EmailProvider re-exports the core EmailProvider type
type EmailProvider = core.EmailProvider

const (
	ProviderSES      = core.ProviderSES
	ProviderSendGrid = core.ProviderSendGrid
	ProviderMailgun  = core.ProviderMailgun
	ProviderMailchimp = core.ProviderMailchimp
)

// Client represents the Huefy SDK client wrapper around the core client
type Client struct {
	coreClient *core.Client
}

// RetryConfig re-exports the core RetryConfig type
type RetryConfig = core.RetryConfig

// DefaultRetryConfig re-exports the core default retry config
var DefaultRetryConfig = core.DefaultRetryConfig

// ClientOption represents an option for configuring the client
type ClientOption func(*clientOptions)

type clientOptions struct {
	baseURL     string
	httpClient  *http.Client
	retryConfig *RetryConfig
}

// WithBaseURL sets a custom base URL for the API
func WithBaseURL(baseURL string) ClientOption {
	return func(o *clientOptions) {
		o.baseURL = baseURL
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(o *clientOptions) {
		o.httpClient = httpClient
	}
}

// WithRetryConfig sets custom retry configuration
func WithRetryConfig(config *RetryConfig) ClientOption {
	return func(o *clientOptions) {
		o.retryConfig = config
	}
}

// NewClient creates a new Huefy SDK client
func NewClient(apiKey string, opts ...ClientOption) *Client {
	// Process options
	options := &clientOptions{
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		retryConfig: DefaultRetryConfig,
	}

	for _, opt := range opts {
		opt(options)
	}

	// Create core options
	var coreOpts []core.ClientOption
	if options.baseURL != "" {
		coreOpts = append(coreOpts, core.WithBaseURL(options.baseURL))
	}
	if options.httpClient != nil {
		coreOpts = append(coreOpts, core.WithHTTPClient(options.httpClient))
	}
	if options.retryConfig != nil {
		coreOpts = append(coreOpts, core.WithRetryConfig(options.retryConfig))
	}

	// Create core client
	coreClient := core.NewClient(apiKey, coreOpts...)

	return &Client{
		coreClient: coreClient,
	}
}

// SendEmailRequest re-exports the core SendEmailRequest type
type SendEmailRequest = core.SendEmailRequest

// SendEmailResponse re-exports the core SendEmailResponse type
type SendEmailResponse = core.SendEmailResponse

// BulkEmailRequest re-exports the core BulkEmailRequest type
type BulkEmailRequest = core.BulkEmailRequest

// BulkEmailResult re-exports the core BulkEmailResult type
type BulkEmailResult = core.BulkEmailResult

// BulkEmailResponse re-exports the core BulkEmailResponse type
type BulkEmailResponse = core.BulkEmailResponse

// HealthResponse re-exports the core HealthResponse type
type HealthResponse = core.HealthResponse

// SendEmail sends a single email using a template
func (c *Client) SendEmail(ctx context.Context, req *SendEmailRequest) (*SendEmailResponse, error) {
	return c.coreClient.SendEmail(ctx, req)
}

// SendBulkEmails sends multiple emails in a single request
func (c *Client) SendBulkEmails(ctx context.Context, emails []SendEmailRequest) (*BulkEmailResponse, error) {
	return c.coreClient.SendBulkEmails(ctx, emails)
}

// HealthCheck checks the API health status
func (c *Client) HealthCheck(ctx context.Context) (*HealthResponse, error) {
	return c.coreClient.HealthCheck(ctx)
}

