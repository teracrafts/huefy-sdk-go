# Huefy Go SDK

The official Go SDK for the Huefy email sending platform by TeraCrafts. Send template-based emails with support for multiple providers, automatic retries, and comprehensive error handling.

## Installation

```bash
go get github.com/teracrafts/huefy-sdk-go@v2.0.0
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    huefy "github.com/teracrafts/huefy-sdk-go"
)

func main() {
    // Create client
    client := huefy.NewClient("your-api-key")
    
    // Send email
    response, err := client.SendEmail(context.Background(), &huefy.SendEmailRequest{
        TemplateKey: "welcome-email",
        Data: map[string]interface{}{
            "name": "John Doe",
            "company": "Acme Corp",
        },
        Recipient: "john@example.com",
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Email sent: %s\n", response.MessageID)
}
```

## Features

- ✅ **Template-based emails** - Send emails using predefined templates
- ✅ **Multiple providers** - Support for SES, SendGrid, Mailgun, Mailchimp
- ✅ **Automatic retries** - Configurable retry logic with exponential backoff
- ✅ **Error handling** - Comprehensive error types for different failure scenarios
- ✅ **Context support** - Full context.Context support for timeouts and cancellation
- ✅ **Bulk emails** - Send multiple emails in a single request
- ✅ **Health checks** - Monitor API health status
- ✅ **Type safety** - Full Go type safety with proper error types

## Configuration

### Basic Configuration

```go
client := huefy.NewClient("your-api-key")
```

### Advanced Configuration

```go
client := huefy.NewClient("your-api-key",
    huefy.WithBaseURL("https://api.huefy.com"),
    huefy.WithHTTPClient(&http.Client{
        Timeout: 60 * time.Second,
    }),
    huefy.WithRetryConfig(&huefy.RetryConfig{
        MaxRetries: 5,
        BaseDelay:  time.Second,
        MaxDelay:   30 * time.Second,
        Multiplier: 2.0,
    }),
)
```

## API Reference

### Client Creation

#### `NewClient(apiKey string, opts ...ClientOption) *Client`

Creates a new Huefy client with the provided API key and optional configuration.

**Options:**
- `WithBaseURL(url string)` - Set custom API base URL
- `WithHTTPClient(client *http.Client)` - Use custom HTTP client
- `WithRetryConfig(config *RetryConfig)` - Configure retry behavior

### Email Operations

#### `SendEmail(ctx context.Context, req *SendEmailRequest) (*SendEmailResponse, error)`

Sends a single email using a template.

```go
request := &huefy.SendEmailRequest{
    TemplateKey: "welcome-email",
    Data: map[string]interface{}{
        "name": "John Doe",
        "company": "Acme Corp",
    },
    Recipient: "john@example.com",
    Provider: &huefy.ProviderSendGrid, // Optional
}

response, err := client.SendEmail(ctx, request)
```

#### `SendBulkEmails(ctx context.Context, emails []SendEmailRequest) (*BulkEmailResponse, error)`

Sends multiple emails in a single request.

```go
emails := []huefy.SendEmailRequest{
    {
        TemplateKey: "welcome-email",
        Data: map[string]interface{}{"name": "John"},
        Recipient: "john@example.com",
    },
    {
        TemplateKey: "welcome-email",
        Data: map[string]interface{}{"name": "Jane"},
        Recipient: "jane@example.com",
    },
}

response, err := client.SendBulkEmails(ctx, emails)
```

#### `HealthCheck(ctx context.Context) (*HealthResponse, error)`

Checks the API health status.

```go
health, err := client.HealthCheck(ctx)
fmt.Printf("API Status: %s\n", health.Status)
```

## Error Handling

The SDK provides specific error types for different failure scenarios:

```go
response, err := client.SendEmail(ctx, request)
if err != nil {
    switch {
    case huefy.IsAuthenticationError(err):
        // Handle authentication failures
        log.Printf("Authentication failed: %v", err)
        
    case huefy.IsTemplateNotFoundError(err):
        // Handle template not found
        if templateErr, ok := err.(*huefy.TemplateNotFoundError); ok {
            log.Printf("Template '%s' not found", templateErr.TemplateKey)
        }
        
    case huefy.IsRateLimitError(err):
        // Handle rate limiting
        if rateLimitErr, ok := err.(*huefy.RateLimitError); ok {
            log.Printf("Rate limited. Retry after %d seconds", rateLimitErr.RetryAfter)
        }
        
    case huefy.IsProviderError(err):
        // Handle provider-specific errors
        if providerErr, ok := err.(*huefy.ProviderError); ok {
            log.Printf("Provider %s error: %s", providerErr.Provider, providerErr.ProviderCode)
        }
        
    case huefy.IsNetworkError(err):
        // Handle network errors (automatically retried)
        log.Printf("Network error: %v", err)
        
    default:
        log.Printf("Unknown error: %v", err)
    }
}
```

### Error Types

- `AuthenticationError` - Invalid API key or authentication failure
- `TemplateNotFoundError` - Specified template doesn't exist
- `InvalidTemplateDataError` - Template data validation failed
- `InvalidRecipientError` - Invalid recipient email address
- `ProviderError` - Email provider rejected the message
- `RateLimitError` - Rate limit exceeded
- `NetworkError` - Network connectivity issues
- `TimeoutError` - Request timeout
- `ValidationError` - Request validation failed

## Email Providers

Supported email providers:

```go
const (
    ProviderSES      EmailProvider = "ses"       // Amazon SES
    ProviderSendGrid EmailProvider = "sendgrid"  // SendGrid
    ProviderMailgun  EmailProvider = "mailgun"   // Mailgun
    ProviderMailchimp EmailProvider = "mailchimp" // Mailchimp
)
```

## Retry Configuration

Configure automatic retry behavior for failed requests:

```go
retryConfig := &huefy.RetryConfig{
    MaxRetries: 3,                    // Maximum number of retries
    BaseDelay:  time.Second,          // Initial delay between retries
    MaxDelay:   30 * time.Second,     // Maximum delay between retries
    Multiplier: 2.0,                  // Delay multiplier for exponential backoff
}

client := huefy.NewClient("api-key", huefy.WithRetryConfig(retryConfig))
```

Retries are automatically performed for:
- Network errors
- Timeout errors  
- Rate limit errors
- Server errors (5xx)

## Context Support

All operations support `context.Context` for timeouts and cancellation:

```go
// Request with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

response, err := client.SendEmail(ctx, request)

// Request with cancellation
ctx, cancel := context.WithCancel(context.Background())
go func() {
    // Cancel after some condition
    cancel()
}()

response, err := client.SendEmail(ctx, request)
```

## Examples

See the [examples](examples/) directory for complete examples:

- [Basic Usage](examples/basic/main.go) - Simple email sending
- [Bulk Emails](examples/bulk/main.go) - Sending multiple emails
- [Advanced Usage](examples/advanced/main.go) - Custom configuration and error handling

## Testing

Run the test suite:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -v -cover ./...
```

## Requirements

- Go 1.21 or later
- Valid Huefy API key

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Support

For support and questions:

- 📧 Email: support@teracrafts.com
- 📖 Documentation: https://docs.huefy.com
- 🐛 Issues: https://github.com/teracrafts/huefy-sdk-go/issues