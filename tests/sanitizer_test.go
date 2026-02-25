package tests

import (
	"strings"
	"testing"

	sdkerrors "github.com/teracrafts/huefy-go/errors"
)

func TestSanitizeUnixPaths(t *testing.T) {
	msg := "failed to read /home/user/config/secrets.json"
	result := sdkerrors.SanitizeErrorMessage(msg, nil)

	if strings.Contains(result, "/home/user") {
		t.Errorf("expected Unix path to be sanitized, got %q", result)
	}
	if !strings.Contains(result, "[PATH_REDACTED]") {
		t.Errorf("expected [PATH_REDACTED] in result, got %q", result)
	}
}

func TestSanitizeWindowsPaths(t *testing.T) {
	msg := `failed to read C:\Users\admin\config\secrets.json`
	result := sdkerrors.SanitizeErrorMessage(msg, nil)

	if strings.Contains(result, `C:\Users`) {
		t.Errorf("expected Windows path to be sanitized, got %q", result)
	}
	if !strings.Contains(result, "[PATH_REDACTED]") {
		t.Errorf("expected [PATH_REDACTED] in result, got %q", result)
	}
}

func TestSanitizeIPv4Addresses(t *testing.T) {
	msg := "connection failed to 192.168.1.100:5432"
	result := sdkerrors.SanitizeErrorMessage(msg, nil)

	if strings.Contains(result, "192.168.1.100") {
		t.Errorf("expected IPv4 to be sanitized, got %q", result)
	}
	if !strings.Contains(result, "[IP_REDACTED]") {
		t.Errorf("expected [IP_REDACTED] in result, got %q", result)
	}
}

func TestSanitizeAPIKeys(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		redacted string
	}{
		{
			name:    "SDK key",
			input:   "invalid key: sk_test_1234567890abcdef",
			redacted: "[KEY_REDACTED]",
		},
		{
			name:    "server key",
			input:   "invalid key: srv_production_abcdefgh12345678",
			redacted: "[SERVER_KEY_REDACTED]",
		},
		{
			name:    "API key with prefix",
			input:   "failed with api_key_abcdef1234567890",
			redacted: "[KEY_REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sdkerrors.SanitizeErrorMessage(tt.input, nil)
			if !strings.Contains(result, tt.redacted) {
				t.Errorf("expected %q in result, got %q", tt.redacted, result)
			}
		})
	}
}

func TestSanitizeEmails(t *testing.T) {
	msg := "user not found: john.doe@example.com"
	result := sdkerrors.SanitizeErrorMessage(msg, nil)

	if strings.Contains(result, "john.doe@example.com") {
		t.Errorf("expected email to be sanitized, got %q", result)
	}
	if !strings.Contains(result, "[EMAIL_REDACTED]") {
		t.Errorf("expected [EMAIL_REDACTED] in result, got %q", result)
	}
}

func TestSanitizeConnectionStrings(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"MongoDB", "connection failed: mongodb://admin:password@host:27017/db"},
		{"PostgreSQL", "connection failed: postgres://user:pass@host:5432/db"},
		{"MySQL", "connection failed: mysql://user:pass@host:3306/db"},
		{"Redis", "connection failed: redis://user:pass@host:6379/0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sdkerrors.SanitizeErrorMessage(tt.input, nil)
			if !strings.Contains(result, "[CONNECTION_STRING_REDACTED]") {
				t.Errorf("expected [CONNECTION_STRING_REDACTED] in result, got %q", result)
			}
		})
	}
}

func TestSanitizeMultiplePatterns(t *testing.T) {
	msg := "error at /home/user/app connecting to 10.0.0.1 with key sk_live_abcdefghijklmnop for user@example.com via postgres://admin:secret@db:5432/mydb"
	result := sdkerrors.SanitizeErrorMessage(msg, nil)

	if strings.Contains(result, "/home/user") {
		t.Error("expected path to be sanitized")
	}
	if strings.Contains(result, "10.0.0.1") {
		t.Error("expected IP to be sanitized")
	}
	if strings.Contains(result, "user@example.com") {
		t.Error("expected email to be sanitized")
	}
	if strings.Contains(result, "postgres://") {
		t.Error("expected connection string to be sanitized")
	}
}

func TestSanitizeWithCustomConfig(t *testing.T) {
	cfg := &sdkerrors.ErrorSanitizationConfig{
		SanitizeFilePaths:         false,
		SanitizeIPAddresses:       true,
		SanitizeAPIKeys:           false,
		SanitizeEmails:            false,
		SanitizeConnectionStrings: false,
	}

	msg := "error at /home/user/app connecting to 10.0.0.1 for user@example.com"
	result := sdkerrors.SanitizeErrorMessage(msg, cfg)

	// Only IP should be sanitized.
	if strings.Contains(result, "10.0.0.1") {
		t.Error("expected IP to be sanitized")
	}
	if !strings.Contains(result, "user@example.com") {
		t.Error("expected email to NOT be sanitized (disabled in config)")
	}
}

func TestSanitizeEmptyMessage(t *testing.T) {
	result := sdkerrors.SanitizeErrorMessage("", nil)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestSanitizeNoSensitiveData(t *testing.T) {
	msg := "request failed with status 500"
	result := sdkerrors.SanitizeErrorMessage(msg, nil)
	if result != msg {
		t.Errorf("expected unchanged message %q, got %q", msg, result)
	}
}

func TestSetAndGetDefaultSanitizationConfig(t *testing.T) {
	original := sdkerrors.GetDefaultSanitizationConfig()

	custom := &sdkerrors.ErrorSanitizationConfig{
		SanitizeFilePaths:         false,
		SanitizeIPAddresses:       false,
		SanitizeAPIKeys:           true,
		SanitizeEmails:            true,
		SanitizeConnectionStrings: false,
	}

	sdkerrors.SetDefaultSanitizationConfig(custom)
	retrieved := sdkerrors.GetDefaultSanitizationConfig()

	if retrieved.SanitizeFilePaths != false {
		t.Error("expected SanitizeFilePaths to be false")
	}
	if retrieved.SanitizeIPAddresses != false {
		t.Error("expected SanitizeIPAddresses to be false")
	}
	if retrieved.SanitizeAPIKeys != true {
		t.Error("expected SanitizeAPIKeys to be true")
	}

	// Restore original.
	sdkerrors.SetDefaultSanitizationConfig(original)
}

func TestGetDefaultSanitizationConfigReturnsCopy(t *testing.T) {
	cfg1 := sdkerrors.GetDefaultSanitizationConfig()
	cfg2 := sdkerrors.GetDefaultSanitizationConfig()

	// Modifying one should not affect the other.
	cfg1.SanitizeEmails = false
	if cfg2.SanitizeEmails == cfg1.SanitizeEmails {
		// This could be a coincidence if default is false; check against known default.
		cfg3 := sdkerrors.GetDefaultSanitizationConfig()
		if cfg3.SanitizeEmails == false {
			t.Error("expected GetDefaultSanitizationConfig to return independent copies")
		}
	}
}

func TestSanitizeAllDisabled(t *testing.T) {
	cfg := &sdkerrors.ErrorSanitizationConfig{
		SanitizeFilePaths:         false,
		SanitizeIPAddresses:       false,
		SanitizeAPIKeys:           false,
		SanitizeEmails:            false,
		SanitizeConnectionStrings: false,
	}

	msg := "error at /home/user/app connecting to 10.0.0.1 for user@example.com via postgres://admin:pass@host/db"
	result := sdkerrors.SanitizeErrorMessage(msg, cfg)

	if result != msg {
		t.Errorf("expected unchanged message when all sanitization disabled, got %q", result)
	}
}
