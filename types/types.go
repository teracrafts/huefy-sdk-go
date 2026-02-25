package types

// Logger defines the logging interface used throughout the Huefy SDK.
// Implementations must be safe for concurrent use.
type Logger interface {
	// Debug logs a debug-level message.
	Debug(msg string)

	// Info logs an informational message.
	Info(msg string)

	// Warn logs a warning message.
	Warn(msg string)

	// Error logs an error message.
	Error(msg string)
}

// HealthResponse represents the response from the API health check endpoint.
type HealthResponse struct {
	// Status is the health status (e.g., "ok", "degraded").
	Status string `json:"status"`

	// Timestamp is the server timestamp in ISO 8601 format.
	Timestamp string `json:"timestamp"`

	// Version is the API version string.
	Version string `json:"version"`
}
