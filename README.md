# huefy-go

Official Go SDK for [Huefy](https://huefy.dev) — transactional email delivery made simple.

## Installation

```bash
go get github.com/teracrafts/huefy-go@v1.0.0
```

## Requirements

- Go 1.21+

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    huefy "github.com/teracrafts/huefy-go"
)

func main() {
    client, err := huefy.NewClient("sdk_your_api_key")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()
    response, err := client.SendEmail(ctx, &huefy.SendEmailRequest{
        TemplateKey: "welcome-email",
        Recipient: huefy.SendEmailRecipient{
            Email: "alice@example.com",
            Type:  "cc",
            Data:  map[string]any{"firstName": "Alice"},
        },
        Data: map[string]any{"firstName": "Alice", "trialDays": 14},
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Message ID:", response.MessageID)
}
```

## Key Features

- **Context-based cancellation** — every method accepts a `context.Context` for deadlines and cancellation
- **Functional options** — `With*` options keep the constructor ergonomic and forward-compatible
- **Retry with exponential backoff** — configurable attempts, base delay, ceiling, and jitter
- **Circuit breaker** — opens after 5 consecutive failures, probes after 30 s
- **HMAC-SHA256 signing** — `WithRequestSigning(true)` for additional integrity verification
- **Key rotation** — `WithSecondaryAPIKey` for seamless failover
- **Rate limit callbacks** — `WithRateLimitCallback` fires whenever rate-limit headers change
- **Thread-safe** — safe for concurrent use across goroutines
- **PII detection** — warns when template variables contain sensitive field patterns

## Configuration Reference

| Option | Default | Description |
|--------|---------|-------------|
| `WithBaseURL(url)` | `https://api.huefy.dev/api/v1/sdk` | Override the API base URL |
| `WithTimeout(d)` | `30s` | Per-request timeout |
| `WithRetryConfig(cfg)` | see below | Retry behaviour |
| `WithCircuitBreakerConfig(cfg)` | see below | Circuit breaker behaviour |
| `WithLogger(l)` | `ConsoleLogger` | Custom logging sink |
| `WithSecondaryAPIKey(key)` | — | Backup key used during key rotation |
| `WithRequestSigning(true)` | `false` | Enable HMAC-SHA256 request signing |
| `WithRateLimitCallback(fn)` | — | Callback fired on rate-limit header changes |

### RetryConfig defaults

| Field | Default | Description |
|-------|---------|-------------|
| `MaxAttempts` | `3` | Total attempts including the first |
| `BaseDelay` | `500ms` | Exponential backoff base delay |
| `MaxDelay` | `10s` | Maximum backoff delay |
| `Jitter` | `0.2` | Random jitter factor (0–1) |

### CircuitBreakerConfig defaults

| Field | Default | Description |
|-------|---------|-------------|
| `FailureThreshold` | `5` | Consecutive failures before circuit opens |
| `ResetTimeout` | `30s` | Duration before half-open probe |

## Bulk Email

```go
bulk, err := client.SendBulkEmails(ctx, &huefy.SendBulkEmailsRequest{
    TemplateKey: "promo",
    Recipients: []huefy.BulkRecipient{
        {Email: "bob@example.com"},
        {Email: "carol@example.com"},
    },
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Sent: %d, Failed: %d\n", bulk.TotalSent, bulk.TotalFailed)
```

## Error Handling

```go
import (
    huefy "github.com/teracrafts/huefy-go"
    "errors"
)

response, err := client.SendEmail(ctx, req)
if err != nil {
    var authErr *huefy.HuefyAuthError
    var rateLimitErr *huefy.HuefyRateLimitError
    var circuitErr *huefy.HuefyCircuitOpenError

    switch {
    case errors.As(err, &authErr):
        log.Println("Invalid API key")
    case errors.As(err, &rateLimitErr):
        log.Printf("Rate limited. Retry after %ds", rateLimitErr.RetryAfter)
    case errors.As(err, &circuitErr):
        log.Println("Circuit open — service unavailable, backing off")
    default:
        log.Fatal(err)
    }
}
```

### Error Code Reference

| Type | Code | Meaning |
|------|------|---------|
| `HuefyInitError` | 1001 | Client failed to initialise |
| `HuefyAuthError` | 1102 | API key rejected |
| `HuefyNetworkError` | 1201 | Upstream request failed |
| `HuefyCircuitOpenError` | 1301 | Circuit breaker tripped |
| `HuefyRateLimitError` | 2003 | Rate limit exceeded |
| `HuefyTemplateMissingError` | 2005 | Template key not found |

## Health Check

```go
health, err := client.HealthCheck(ctx)
if err != nil {
    log.Fatal(err)
}
if health.Status != "healthy" {
    log.Printf("Huefy degraded: %s", health.Status)
}
```

## Local Development

Set `HUEFY_MODE=local` to target `https://api.huefy.on/api/v1/sdk`, or use `WithBaseURL`. To bypass Caddy and hit the raw app port, set `http://localhost:8080/api/v1/sdk` explicitly:

```go
client, err := huefy.NewClient(
    "sdk_local_key",
    huefy.WithBaseURL("https://api.huefy.on/api/v1/sdk"),
)
```

## Developer Guide

Full documentation, advanced patterns, and provider configuration are in the [Go Developer Guide](../../docs/spec/guides/go.guide.md).

## License

MIT
