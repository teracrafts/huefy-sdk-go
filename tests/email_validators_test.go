package tests

import (
	"strings"
	"testing"

	"github.com/teracrafts/huefy-go/validators"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid", "user@example.com", false},
		{"empty", "", true},
		{"no domain", "user@", true},
		{"no at sign", "userexample.com", true},
		{"too long", strings.Repeat("a", 250) + "@b.co", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validators.ValidateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail(%q) error = %v, wantErr %v", tt.email, err, tt.wantErr)
			}
		})
	}
}

func TestValidateTemplateKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"valid", "welcome-email", false},
		{"empty", "", true},
		{"too long", strings.Repeat("a", 101), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validators.ValidateTemplateKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTemplateKey(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
			}
		})
	}
}

func TestValidateBulkCount(t *testing.T) {
	tests := []struct {
		name    string
		count   int
		wantErr bool
	}{
		{"valid", 10, false},
		{"zero", 0, true},
		{"over limit", 101, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validators.ValidateBulkCount(tt.count)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBulkCount(%d) error = %v, wantErr %v", tt.count, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSendEmailInput(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		errs := validators.ValidateSendEmailInput("tpl", map[string]string{"name": "John"}, "user@test.com")
		if len(errs) != 0 {
			t.Errorf("expected no errors, got %v", errs)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		errs := validators.ValidateSendEmailInput("", nil, "bad")
		if len(errs) == 0 {
			t.Error("expected errors, got none")
		}
	})
}
