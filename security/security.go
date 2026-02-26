package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// piiPatterns contains normalized (lowercase, no hyphens/underscores) field name
// substrings that commonly indicate PII. Field names are normalized the same way
// before matching, so these patterns match regardless of separator style.
var piiPatterns = []string{
	"email", "phone", "telephone", "mobile",
	"ssn", "socialsecurity",
	"creditcard", "cardnumber", "cvv",
	"password", "passwd", "secret", "token",
	"apikey", "privatekey",
	"accesstoken", "refreshtoken", "authtoken",
	"address", "street", "zipcode", "postalcode",
	"dateofbirth", "dob", "birthdate",
	"passport", "driverlicense", "nationalid",
	"bankaccount", "routingnumber", "iban", "swift",
}

// IsPotentialPIIField checks whether a field name matches any known PII pattern.
// The field name is normalized by lowercasing and removing hyphens and underscores
// before checking for substring matches.
func IsPotentialPIIField(fieldName string) bool {
	normalized := strings.ToLower(fieldName)
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, "_", "")
	for _, pattern := range piiPatterns {
		if strings.Contains(normalized, strings.ToLower(pattern)) {
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
// (starts with "srv_").
func IsServerKey(apiKey string) bool {
	lower := strings.ToLower(apiKey)
	return strings.HasPrefix(lower, "srv_")
}

// IsClientKey reports whether the given API key looks like a client key
// (starts with "sdk_" or "cli_").
func IsClientKey(apiKey string) bool {
	lower := strings.ToLower(apiKey)
	return strings.HasPrefix(lower, "sdk_") ||
		strings.HasPrefix(lower, "cli_")
}

// CreateRequestSignature generates a RequestSignature for the given body and
// API key. The signature is an HMAC-SHA256 of "timestamp.body" using the
// current Unix millisecond timestamp.
func CreateRequestSignature(body, apiKey string) RequestSignature {
	timestamp := time.Now().UnixMilli()
	message := fmt.Sprintf("%d.%s", timestamp, body)
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
	message := fmt.Sprintf("%d.%s", timestamp, body)
	expected := GenerateHMACSHA256(message, apiKey)

	return hmac.Equal([]byte(signature), []byte(expected))
}

// SignPayload signs a data map with the given API key and timestamp, returning
// a SignedPayload that includes the original data, signature, timestamp, and
// key identifier. The data is serialized using json.Marshal which produces
// deterministic output with sorted keys.
func SignPayload(data map[string]any, apiKey string, timestamp int64) SignedPayload {
	// Serialize the data deterministically using JSON (encoding/json sorts map keys).
	bodyBytes, err := json.Marshal(data)
	if err != nil {
		// Fall back to empty body on marshal failure; callers should ensure
		// data is JSON-serializable.
		bodyBytes = []byte("{}")
	}
	message := fmt.Sprintf("%d.%s", timestamp, string(bodyBytes))
	signature := GenerateHMACSHA256(message, apiKey)

	return SignedPayload{
		Data:      data,
		Signature: signature,
		Timestamp: timestamp,
		KeyID:     GetKeyID(apiKey),
	}
}
