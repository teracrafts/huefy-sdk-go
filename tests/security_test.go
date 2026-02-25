package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/teracrafts/huefy-go/security"
)

func TestIsPotentialPIIFieldDetectsKnownFields(t *testing.T) {
	piiFields := []string{
		"ssn", "social_security_number", "password", "creditCard",
		"email", "phone", "firstName", "last_name", "address",
		"passport", "dob", "bank_account", "salary", "ip_address",
	}

	for _, field := range piiFields {
		if !security.IsPotentialPIIField(field) {
			t.Errorf("expected %q to be detected as PII", field)
		}
	}
}

func TestIsPotentialPIIFieldIgnoresNonPII(t *testing.T) {
	safeFields := []string{
		"id", "status", "count", "description", "type",
		"created_at", "updated_at", "color", "size", "category",
	}

	for _, field := range safeFields {
		if security.IsPotentialPIIField(field) {
			t.Errorf("expected %q to NOT be detected as PII", field)
		}
	}
}

func TestDetectPotentialPIIFlat(t *testing.T) {
	data := map[string]any{
		"id":         123,
		"email":      "test@example.com",
		"password":   "secret",
		"status":     "active",
		"first_name": "John",
	}

	found := security.DetectPotentialPII(data, "")

	if len(found) < 3 {
		t.Errorf("expected at least 3 PII fields, got %d: %v", len(found), found)
	}

	// Verify specific fields are found.
	fieldSet := make(map[string]bool)
	for _, f := range found {
		fieldSet[f] = true
	}

	expected := []string{"email", "password", "first_name"}
	for _, e := range expected {
		if !fieldSet[e] {
			t.Errorf("expected %q in PII results", e)
		}
	}
}

func TestDetectPotentialPIINested(t *testing.T) {
	data := map[string]any{
		"user": map[string]any{
			"email":    "test@example.com",
			"profile":  map[string]any{
				"phone":     "555-1234",
				"nickname":  "johndoe",
			},
		},
		"status": "ok",
	}

	found := security.DetectPotentialPII(data, "")

	fieldSet := make(map[string]bool)
	for _, f := range found {
		fieldSet[f] = true
	}

	if !fieldSet["user.email"] {
		t.Error("expected 'user.email' in PII results")
	}
	if !fieldSet["user.profile.phone"] {
		t.Error("expected 'user.profile.phone' in PII results")
	}
}

func TestDetectPotentialPIIEmptyData(t *testing.T) {
	found := security.DetectPotentialPII(map[string]any{}, "")
	if len(found) != 0 {
		t.Errorf("expected 0 PII fields for empty data, got %d", len(found))
	}
}

func TestGenerateHMACSHA256(t *testing.T) {
	// Known HMAC-SHA256 test vector.
	message := "hello world"
	key := "secret"

	sig := security.GenerateHMACSHA256(message, key)

	if sig == "" {
		t.Error("expected non-empty signature")
	}

	// Signature should be consistent.
	sig2 := security.GenerateHMACSHA256(message, key)
	if sig != sig2 {
		t.Error("expected consistent signature for same input")
	}

	// Different message should produce different signature.
	sig3 := security.GenerateHMACSHA256("different", key)
	if sig == sig3 {
		t.Error("expected different signature for different message")
	}

	// Different key should produce different signature.
	sig4 := security.GenerateHMACSHA256(message, "other-key")
	if sig == sig4 {
		t.Error("expected different signature for different key")
	}
}

func TestGetKeyID(t *testing.T) {
	tests := []struct {
		apiKey   string
		expected string
	}{
		{"sk_test_1234567890abcdef", "sk_test_"},
		{"short", "short"},
		{"12345678rest", "12345678"},
		{"", ""},
	}

	for _, tt := range tests {
		result := security.GetKeyID(tt.apiKey)
		if result != tt.expected {
			t.Errorf("GetKeyID(%q) = %q, want %q", tt.apiKey, result, tt.expected)
		}
	}
}

func TestIsServerKey(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"srv_abc123", true},
		{"svr_abc123", true},
		{"server_abc123", true},
		{"SRV_ABC123", true},
		{"pk_abc123", false},
		{"client_abc123", false},
		{"random_key", false},
	}

	for _, tt := range tests {
		result := security.IsServerKey(tt.key)
		if result != tt.expected {
			t.Errorf("IsServerKey(%q) = %v, want %v", tt.key, result, tt.expected)
		}
	}
}

func TestIsClientKey(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"pk_abc123", true},
		{"pub_abc123", true},
		{"client_abc123", true},
		{"PK_ABC123", true},
		{"srv_abc123", false},
		{"server_abc123", false},
		{"random_key", false},
	}

	for _, tt := range tests {
		result := security.IsClientKey(tt.key)
		if result != tt.expected {
			t.Errorf("IsClientKey(%q) = %v, want %v", tt.key, result, tt.expected)
		}
	}
}

func TestCreateRequestSignature(t *testing.T) {
	body := `{"key":"value"}`
	apiKey := "sk_test_1234567890abcdef"

	sig := security.CreateRequestSignature(body, apiKey)

	if sig.Signature == "" {
		t.Error("expected non-empty signature")
	}
	if sig.Timestamp <= 0 {
		t.Error("expected positive timestamp")
	}
	if sig.KeyID != "sk_test_" {
		t.Errorf("expected key ID 'sk_test_', got %q", sig.KeyID)
	}
}

func TestVerifyRequestSignature(t *testing.T) {
	body := `{"key":"value"}`
	apiKey := "sk_test_1234567890abcdef"

	sig := security.CreateRequestSignature(body, apiKey)

	// Valid signature should verify.
	valid := security.VerifyRequestSignature(body, sig.Signature, sig.Timestamp, apiKey, 300000)
	if !valid {
		t.Error("expected signature to verify")
	}

	// Wrong body should fail.
	valid = security.VerifyRequestSignature(`{"wrong":"body"}`, sig.Signature, sig.Timestamp, apiKey, 300000)
	if valid {
		t.Error("expected verification to fail for wrong body")
	}

	// Wrong key should fail.
	valid = security.VerifyRequestSignature(body, sig.Signature, sig.Timestamp, "wrong_key_12345678", 300000)
	if valid {
		t.Error("expected verification to fail for wrong key")
	}

	// Tampered signature should fail.
	valid = security.VerifyRequestSignature(body, "tamperedsignature", sig.Timestamp, apiKey, 300000)
	if valid {
		t.Error("expected verification to fail for tampered signature")
	}
}

func TestVerifyRequestSignatureExpired(t *testing.T) {
	body := `{"key":"value"}`
	apiKey := "sk_test_1234567890abcdef"

	// Create a signature with a very old timestamp.
	oldTimestamp := time.Now().Add(-10 * time.Minute).UnixMilli()
	message := body + "." + formatInt64(oldTimestamp)
	signature := security.GenerateHMACSHA256(message, apiKey)

	// Should fail with a short max age.
	valid := security.VerifyRequestSignature(body, signature, oldTimestamp, apiKey, 60000)
	if valid {
		t.Error("expected expired signature to fail verification")
	}
}

func TestVerifyRequestSignatureDefaultMaxAge(t *testing.T) {
	body := `{"key":"value"}`
	apiKey := "sk_test_1234567890abcdef"

	sig := security.CreateRequestSignature(body, apiKey)

	// Passing 0 should use default max age (5 minutes).
	valid := security.VerifyRequestSignature(body, sig.Signature, sig.Timestamp, apiKey, 0)
	if !valid {
		t.Error("expected signature to verify with default max age")
	}
}

func TestSignPayload(t *testing.T) {
	data := map[string]any{"key": "value"}
	apiKey := "sk_test_1234567890abcdef"
	timestamp := time.Now().UnixMilli()

	signed := security.SignPayload(data, apiKey, timestamp)

	if signed.Signature == "" {
		t.Error("expected non-empty signature")
	}
	if signed.Timestamp != timestamp {
		t.Errorf("expected timestamp %d, got %d", timestamp, signed.Timestamp)
	}
	if signed.KeyID != "sk_test_" {
		t.Errorf("expected key ID 'sk_test_', got %q", signed.KeyID)
	}
	if signed.Data == nil {
		t.Error("expected non-nil data")
	}
}

func TestWarnIfPotentialPIINilLogger(t *testing.T) {
	// Should not panic with nil logger.
	data := map[string]any{"email": "test@example.com"}
	security.WarnIfPotentialPII(data, "test", nil)
}

// formatInt64 is a helper to format int64 to string for test use.
func formatInt64(n int64) string {
	return fmt.Sprintf("%d", n)
}
