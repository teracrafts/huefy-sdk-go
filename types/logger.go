package types

import (
	"fmt"
	"time"
)

// ConsoleLogger is a Logger implementation that writes to stdout with
// timestamps and log level prefixes.
type ConsoleLogger struct{}

// NewConsoleLogger returns a new ConsoleLogger.
func NewConsoleLogger() *ConsoleLogger {
	return &ConsoleLogger{}
}

// Debug logs a debug-level message to stdout.
func (l *ConsoleLogger) Debug(msg string) {
	fmt.Printf("[%s] DEBUG: %s\n", time.Now().Format(time.RFC3339), msg)
}

// Info logs an informational message to stdout.
func (l *ConsoleLogger) Info(msg string) {
	fmt.Printf("[%s] INFO:  %s\n", time.Now().Format(time.RFC3339), msg)
}

// Warn logs a warning message to stdout.
func (l *ConsoleLogger) Warn(msg string) {
	fmt.Printf("[%s] WARN:  %s\n", time.Now().Format(time.RFC3339), msg)
}

// Error logs an error message to stdout.
func (l *ConsoleLogger) Error(msg string) {
	fmt.Printf("[%s] ERROR: %s\n", time.Now().Format(time.RFC3339), msg)
}

// NoopLogger is a Logger implementation that discards all log messages.
type NoopLogger struct{}

// NewNoopLogger returns a new NoopLogger.
func NewNoopLogger() *NoopLogger {
	return &NoopLogger{}
}

// Debug is a no-op.
func (l *NoopLogger) Debug(_ string) {}

// Info is a no-op.
func (l *NoopLogger) Info(_ string) {}

// Warn is a no-op.
func (l *NoopLogger) Warn(_ string) {}

// Error is a no-op.
func (l *NoopLogger) Error(_ string) {}
