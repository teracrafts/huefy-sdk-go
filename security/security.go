package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/teracrafts/huefy-go/types"
)

// SignedPayload holds a payload along with its cryptographic signature metadata.
type SignedPayload struct {
	// Data is the original payload as a JSON-serializable map.
	Data map[string]any `json:"data"`

	// Signature is the HMAC-SHA256 hex digest.
	Signature string `json:"signature"`

	// Timestamp is the Unix millisecond timestamp at the time of signing.
	Timestamp int64 `json:"timestamp"`

	// KeyID is the truncated key identifier.
	KeyID string `json:"keyId"`
}

// RequestSignature holds the components needed for request-level HMAC signing.
type RequestSignature struct {
	// Signature is the HMAC-SHA256 hex digest of the request body.
	Signature string

	// Timestamp is the Unix millisecond timestamp at the time of signing.
	Timestamp int64

	// KeyID is the truncated key identifier.
	KeyID string
}

// piiPatterns contains field name substrings that commonly indicate PII.
var piiPatterns = []string{
	"ssn", "social_security", "socialSecurity",
	"password", "passwd", "pass_word",
	"secret", "token", "api_key", "apiKey", "api-key",
	"credit_card", "creditCard", "credit-card",
	"card_number", "cardNumber", "card-number",
	"cvv", "cvc", "ccv",
	"expiry", "expiration", "exp_date",
	"birth", "dob", "date_of_birth", "dateOfBirth",
	"phone", "mobile", "cell", "telephone",
	"address", "street", "city", "zip", "postal",
	"email", "e-mail", "e_mail",
	"first_name", "firstName", "first-name",
	"last_name", "lastName", "last-name",
	"full_name", "fullName", "full-name",
	"driver_license", "driverLicense", "driver-license",
	"passport", "passport_number",
	"national_id", "nationalId", "national-id",
	"tax_id", "taxId", "tax-id",
	"bank_account", "bankAccount", "bank-account",
	"routing_number", "routingNumber", "routing-number",
	"iban", "swift", "bic",
	"ip_address", "ipAddress", "ip-address",
	"mac_address", "macAddress", "mac-address",
	"latitude", "longitude", "lat", "lng", "geo",
	"biometric", "fingerprint", "retina",
	"medical", "health", "diagnosis", "prescription",
	"insurance", "policy_number", "policyNumber",
	"salary", "income", "wage", "compensation",
}

// IsPotentialPIIField checks whether a field name matches any known PII pattern.
func IsPotentialPIIField(fieldName string) bool {
	lower := strings.ToLower(fieldName)
	for _, pattern := range piiPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// DetectPotentialPII recursively scans a map for field names that may contain PII.
// It returns a slice of dot-separated field paths that matched PII patterns.
func DetectPotentialPII(data map[string]any, prefix string) []string {
	var found []string

	for key, value := range data {
		fullPath := key
		if prefix != "" {
			fullPath = prefix + "." + key
		}

		if IsPotentialPIIField(key) {
			found = append(found, fullPath)
		}

		// Recurse into nested maps.
		if nested, ok := value.(map[string]any); ok {
			found = append(found, DetectPotentialPII(nested, fullPath)...)
		}
	}

	return found
}

// WarnIfPotentialPII checks data for potential PII fields and logs a warning
// via the provided logger if any are found.
func WarnIfPotentialPII(data map[string]any, dataType string, logger types.Logger) {
	if logger == nil {
		return
	}

	piiFields := DetectPotentialPII(data, "")
	if len(piiFields) > 0 {
		logger.Warn(fmt.Sprintf(
			"potential PII detected in %s data: [%s]. Consider removing or encrypting these fields.",
			dataType,
			strings.Join(piiFields, ", "),
		))
	}
}

// GenerateHMACSHA256 generates an HMAC-SHA256 hex digest for the given message
// using the provided key.
func GenerateHMACSHA256(message, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// GetKeyID returns the first 8 characters of the API key as a key identifier.
func GetKeyID(apiKey string) string {
	if len(apiKey) < 8 {
		return apiKey
	}
	return apiKey[:8]
}

// IsServerKey reports whether the given API key looks like a server key
// (starts with "srv_", "svr_", or "server_").
func IsServerKey(apiKey string) bool {
	lower := strings.ToLower(apiKey)
	return strings.HasPrefix(lower, "srv_") ||
		strings.HasPrefix(lower, "svr_") ||
		strings.HasPrefix(lower, "server_")
}

// IsClientKey reports whether the given API key looks like a client key
// (starts with "pk_", "pub_", or "client_").
func IsClientKey(apiKey string) bool {
	lower := strings.ToLower(apiKey)
	return strings.HasPrefix(lower, "pk_") ||
		strings.HasPrefix(lower, "pub_") ||
		strings.HasPrefix(lower, "client_")
}

// CreateRequestSignature generates a RequestSignature for the given body and
// API key. The signature is an HMAC-SHA256 of the body concatenated with the
// current Unix millisecond timestamp.
func CreateRequestSignature(body, apiKey string) RequestSignature {
	timestamp := time.Now().UnixMilli()
	message := fmt.Sprintf("%s.%d", body, timestamp)
	signature := GenerateHMACSHA256(message, apiKey)

	return RequestSignature{
		Signature: signature,
		Timestamp: timestamp,
		KeyID:     GetKeyID(apiKey),
	}
}

// VerifyRequestSignature verifies that a request signature is valid and not
// expired. maxAgeMs specifies the maximum age of the signature in milliseconds;
// if 0, a default of 300000 (5 minutes) is used.
func VerifyRequestSignature(body, signature string, timestamp int64, apiKey string, maxAgeMs int64) bool {
	if maxAgeMs <= 0 {
		maxAgeMs = 300000 // 5 minutes default.
	}

	// Check timestamp freshness.
	now := time.Now().UnixMilli()
	age := now - timestamp
	if age < 0 {
		age = -age
	}
	if age > maxAgeMs {
		return false
	}

	// Regenerate and compare.
	message := fmt.Sprintf("%s.%d", body, timestamp)
	expected := GenerateHMACSHA256(message, apiKey)

	return hmac.Equal([]byte(signature), []byte(expected))
}

// SignPayload signs a data map with the given API key and timestamp, returning
// a SignedPayload that includes the original data, signature, timestamp, and
// key identifier.
func SignPayload(data map[string]any, apiKey string, timestamp int64) SignedPayload {
	// Serialize the data deterministically enough for signing.
	// For template purposes, we use fmt.Sprintf; real implementations should
	// use canonical JSON serialization.
	message := fmt.Sprintf("%v.%d", data, timestamp)
	signature := GenerateHMACSHA256(message, apiKey)

	return SignedPayload{
		Data:      data,
		Signature: signature,
		Timestamp: timestamp,
		KeyID:     GetKeyID(apiKey),
	}
}
