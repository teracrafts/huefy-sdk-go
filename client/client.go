package client

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/teracrafts/huefy-go/config"
	sdkerrors "github.com/teracrafts/huefy-go/errors"
	internalhttp "github.com/teracrafts/huefy-go/internal/http"
	"github.com/teracrafts/huefy-go/types"
)

// Client is the main Huefy API client. Create one using NewClient.
type Client struct {
	httpClient *internalhttp.Client
	config     config.Config
}

// NewClient creates a new Huefy API client with the provided API key
// and optional configuration options.
//
// Example:
//
//	c := client.NewClient("your-api-key",
//	    config.WithTimeout(10 * time.Second),
//	    config.WithLogger(types.NewConsoleLogger()),
//	)
//	defer c.Close()
func NewClient(apiKey string, opts ...config.Option) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, sdkerrors.NewError(sdkerrors.ErrAuthMissingKey, "api_key is required")
	}

	cfg := config.DefaultConfig(apiKey)
	cfg.Apply(opts...)

	httpClient := internalhttp.NewClient(apiKey, &cfg)

	return &Client{
		httpClient: httpClient,
		config:     cfg,
	}, nil
}

// HealthCheck performs a health check against the Huefy API.
// It returns a HealthResponse on success or an error if the request fails.
func (c *Client) HealthCheck(ctx context.Context) (*types.HealthResponse, error) {
	data, err := c.httpClient.Request(ctx, "GET", "/health", nil)
	if err != nil {
		return nil, err
	}

	var health types.HealthResponse
	if err := json.Unmarshal(data, &health); err != nil {
		return nil, err
	}

	return &health, nil
}

// Request performs an HTTP request through the base client's infrastructure,
// including retry logic, circuit breaking, and optional request signing.
// It returns the raw response body on success.
func (c *Client) Request(ctx context.Context, method, path string, body any) ([]byte, error) {
	return c.httpClient.Request(ctx, method, path, body)
}

// GetLogger returns the logger used by the client.
func (c *Client) GetLogger() types.Logger {
	return c.config.Logger
}

// Close releases any resources held by the client. It should be called when
// the client is no longer needed.
func (c *Client) Close() {
	c.httpClient.Close()
}
