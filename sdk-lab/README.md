# Huefy Go SDK Lab

A standalone verification runner for the Huefy Go SDK.

## Scenarios

1. **Initialization** — create client with a dummy key, verify no error
2. **Config validation** — empty API key returns an error
3. **HMAC signing** — sign payload with HMAC-SHA256, verify 64-char hex result
4. **Error sanitization** — IP and email redacted from error messages
5. **PII detection** — email and SSN fields detected in data map
6. **Circuit breaker state** — new circuit breaker starts in CLOSED state
7. **Health check** — GET /health; passes regardless of network outcome
8. **Cleanup** — close client gracefully

## Run

From `sdks/go/`:

```bash
go run ./sdk-lab
```

> Note: `go mod tidy` must be run inside `sdk-lab/` before first use.
