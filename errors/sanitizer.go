package errors

import (
	"regexp"
	"sync"
)

// ErrorSanitizationConfig controls which patterns are sanitized in error messages.
type ErrorSanitizationConfig struct {
	// SanitizeFilePaths removes Unix and Windows file paths.
	SanitizeFilePaths bool

	// SanitizeIPAddresses removes IPv4 addresses.
	SanitizeIPAddresses bool

	// SanitizeAPIKeys removes API key patterns (SDK, server, CLI keys).
	SanitizeAPIKeys bool

	// SanitizeEmails removes email addresses.
	SanitizeEmails bool

	// SanitizeConnectionStrings removes database/service connection strings.
	SanitizeConnectionStrings bool
}

// sanitizationPattern holds a compiled regex and its replacement string.
type sanitizationPattern struct {
	regex       *regexp.Regexp
	replacement string
	category    string
}

var (
	defaultSanitizationConfig *ErrorSanitizationConfig
	sanitizationMu            sync.RWMutex
)

func init() {
	defaultSanitizationConfig = &ErrorSanitizationConfig{
		SanitizeFilePaths:         true,
		SanitizeIPAddresses:       true,
		SanitizeAPIKeys:           true,
		SanitizeEmails:            true,
		SanitizeConnectionStrings: true,
	}
}

// buildPatterns constructs the sanitization patterns based on the config.
func buildPatterns(cfg *ErrorSanitizationConfig) []sanitizationPattern {
	var patterns []sanitizationPattern

	if cfg.SanitizeFilePaths {
		patterns = append(patterns,
			sanitizationPattern{
				regex:       regexp.MustCompile(`(?:/[a-zA-Z0-9._-]+){2,}`),
				replacement: "[PATH_REDACTED]",
				category:    "unix_path",
			},
			sanitizationPattern{
				regex:       regexp.MustCompile(`[a-zA-Z]:\\(?:[a-zA-Z0-9._-]+\\){1,}`),
				replacement: "[PATH_REDACTED]",
				category:    "windows_path",
			},
		)
	}

	if cfg.SanitizeIPAddresses {
		patterns = append(patterns,
			sanitizationPattern{
				regex:       regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`),
				replacement: "[IP_REDACTED]",
				category:    "ipv4",
			},
		)
	}

	if cfg.SanitizeAPIKeys {
		patterns = append(patterns,
			sanitizationPattern{
				regex:       regexp.MustCompile(`(?i)(?:sk|pk|ak|key|token|secret|api[_-]?key)[_-]?[a-zA-Z0-9]{16,}`),
				replacement: "[KEY_REDACTED]",
				category:    "sdk_key",
			},
			sanitizationPattern{
				regex:       regexp.MustCompile(`(?i)(?:srv|svr|server)[_-]?[a-zA-Z0-9]{16,}`),
				replacement: "[SERVER_KEY_REDACTED]",
				category:    "server_key",
			},
			sanitizationPattern{
				regex:       regexp.MustCompile(`(?i)(?:cli|cmd|console)[_-]?[a-zA-Z0-9]{16,}`),
				replacement: "[CLI_KEY_REDACTED]",
				category:    "cli_key",
			},
		)
	}

	if cfg.SanitizeEmails {
		patterns = append(patterns,
			sanitizationPattern{
				regex:       regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
				replacement: "[EMAIL_REDACTED]",
				category:    "email",
			},
		)
	}

	if cfg.SanitizeConnectionStrings {
		patterns = append(patterns,
			sanitizationPattern{
				regex:       regexp.MustCompile(`(?i)(?:mongodb|postgres|mysql|redis|amqp|mssql)://[^\s]+`),
				replacement: "[CONNECTION_STRING_REDACTED]",
				category:    "connection_string",
			},
		)
	}

	return patterns
}

// SanitizeErrorMessage removes sensitive data from an error message based on the
// provided sanitization config. If config is nil, the default config is used.
func SanitizeErrorMessage(message string, cfg *ErrorSanitizationConfig) string {
	if cfg == nil {
		cfg = GetDefaultSanitizationConfig()
	}

	patterns := buildPatterns(cfg)
	result := message
	for _, p := range patterns {
		result = p.regex.ReplaceAllString(result, p.replacement)
	}
	return result
}

// SetDefaultSanitizationConfig sets the global default sanitization config.
func SetDefaultSanitizationConfig(cfg *ErrorSanitizationConfig) {
	sanitizationMu.Lock()
	defer sanitizationMu.Unlock()
	defaultSanitizationConfig = cfg
}

// GetDefaultSanitizationConfig returns the global default sanitization config.
func GetDefaultSanitizationConfig() *ErrorSanitizationConfig {
	sanitizationMu.RLock()
	defer sanitizationMu.RUnlock()
	cp := *defaultSanitizationConfig
	return &cp
}
