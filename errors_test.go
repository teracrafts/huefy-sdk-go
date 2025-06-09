package huefy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name        string
		error       error
		expectedMsg string
		checkFunc   func(error) bool
	}{
		{
			name:        "HuefyError",
			error:       &HuefyError{Code: "TEST_ERROR", Message: "Test message"},
			expectedMsg: "Huefy API error [TEST_ERROR]: Test message",
			checkFunc:   IsHuefyError,
		},
		{
			name:        "AuthenticationError",
			error:       NewAuthenticationError("Invalid API key"),
			expectedMsg: "Huefy API error [AUTHENTICATION_FAILED]: Invalid API key",
			checkFunc:   IsAuthenticationError,
		},
		{
			name:        "TemplateNotFoundError",
			error:       NewTemplateNotFoundError("welcome-email"),
			expectedMsg: "Huefy API error [TEMPLATE_NOT_FOUND]: Template not found: welcome-email",
			checkFunc:   IsTemplateNotFoundError,
		},
		{
			name:        "InvalidTemplateDataError",
			error:       NewInvalidTemplateDataError("Missing required fields", []string{"name", "email"}),
			expectedMsg: "Huefy API error [INVALID_TEMPLATE_DATA]: Missing required fields",
			checkFunc:   IsInvalidTemplateDataError,
		},
		{
			name:        "InvalidRecipientError",
			error:       NewInvalidRecipientError("Invalid email format"),
			expectedMsg: "Huefy API error [INVALID_RECIPIENT]: Invalid email format",
			checkFunc:   IsInvalidRecipientError,
		},
		{
			name:        "ProviderError",
			error:       NewProviderError(ProviderSES, "MessageRejected", "Email rejected by provider"),
			expectedMsg: "Huefy API error [PROVIDER_ERROR]: Email rejected by provider",
			checkFunc:   IsProviderError,
		},
		{
			name:        "RateLimitError",
			error:       NewRateLimitError("Rate limit exceeded", 60),
			expectedMsg: "Huefy API error [RATE_LIMIT_EXCEEDED]: Rate limit exceeded",
			checkFunc:   IsRateLimitError,
		},
		{
			name:        "NetworkError",
			error:       NewNetworkError("Connection failed"),
			expectedMsg: "Huefy API error [NETWORK_ERROR]: Connection failed",
			checkFunc:   IsNetworkError,
		},
		{
			name:        "TimeoutError",
			error:       NewTimeoutError("Request timeout"),
			expectedMsg: "Huefy API error [TIMEOUT]: Request timeout",
			checkFunc:   IsTimeoutError,
		},
		{
			name:        "ValidationError",
			error:       NewValidationError("Invalid request"),
			expectedMsg: "Huefy API error [VALIDATION_FAILED]: Invalid request",
			checkFunc:   IsValidationError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test error message
			assert.Equal(t, tt.expectedMsg, tt.error.Error())

			// Test type checking function
			assert.True(t, tt.checkFunc(tt.error))

			// Test that other type checking functions return false
			allCheckers := []func(error) bool{
				IsHuefyError,
				IsAuthenticationError,
				IsTemplateNotFoundError,
				IsInvalidTemplateDataError,
				IsInvalidRecipientError,
				IsProviderError,
				IsRateLimitError,
				IsNetworkError,
				IsTimeoutError,
				IsValidationError,
			}

			for _, checker := range allCheckers {
				if checker != tt.checkFunc && checker != IsHuefyError {
					assert.False(t, checker(tt.error), "Should not match other error type checkers")
				}
			}
		})
	}
}

func TestCreateErrorFromResponse(t *testing.T) {
	tests := []struct {
		name         string
		errorResp    *ErrorResponse
		statusCode   int
		expectedType interface{}
		expectedCode ErrorCode
	}{
		{
			name: "AuthenticationError",
			errorResp: &ErrorResponse{
				Error: ErrorDetail{
					Code:    string(ErrorCodeAuthenticationFailed),
					Message: "Invalid API key",
				},
			},
			statusCode:   401,
			expectedType: &AuthenticationError{},
			expectedCode: ErrorCodeAuthenticationFailed,
		},
		{
			name: "TemplateNotFoundError",
			errorResp: &ErrorResponse{
				Error: ErrorDetail{
					Code:    string(ErrorCodeTemplateNotFound),
					Message: "Template not found",
					Details: map[string]interface{}{"templateKey": "welcome-email"},
				},
			},
			statusCode:   404,
			expectedType: &TemplateNotFoundError{},
			expectedCode: ErrorCodeTemplateNotFound,
		},
		{
			name: "InvalidTemplateDataError",
			errorResp: &ErrorResponse{
				Error: ErrorDetail{
					Code:    string(ErrorCodeInvalidTemplateData),
					Message: "Invalid template data",
					Details: map[string]interface{}{
						"validationErrors": []interface{}{"name is required", "email is required"},
					},
				},
			},
			statusCode:   400,
			expectedType: &InvalidTemplateDataError{},
			expectedCode: ErrorCodeInvalidTemplateData,
		},
		{
			name: "ProviderError",
			errorResp: &ErrorResponse{
				Error: ErrorDetail{
					Code:    string(ErrorCodeProviderError),
					Message: "Provider rejected email",
					Details: map[string]interface{}{
						"provider":     "ses",
						"providerCode": "MessageRejected",
					},
				},
			},
			statusCode:   400,
			expectedType: &ProviderError{},
			expectedCode: ErrorCodeProviderError,
		},
		{
			name: "RateLimitError",
			errorResp: &ErrorResponse{
				Error: ErrorDetail{
					Code:    string(ErrorCodeRateLimitExceeded),
					Message: "Rate limit exceeded",
					Details: map[string]interface{}{"retryAfter": float64(60)},
				},
			},
			statusCode:   429,
			expectedType: &RateLimitError{},
			expectedCode: ErrorCodeRateLimitExceeded,
		},
		{
			name: "Generic HuefyError for unknown code",
			errorResp: &ErrorResponse{
				Error: ErrorDetail{
					Code:    "UNKNOWN_ERROR",
					Message: "Unknown error occurred",
				},
			},
			statusCode:   500,
			expectedType: &HuefyError{},
			expectedCode: "UNKNOWN_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := createErrorFromResponse(tt.errorResp, tt.statusCode)

			// Check error type
			assert.IsType(t, tt.expectedType, err)

			// Check error code
			if huefyErr, ok := err.(*HuefyError); ok {
				assert.Equal(t, tt.expectedCode, huefyErr.Code)
			} else {
				// For specific error types, extract the embedded HuefyError
				switch e := err.(type) {
				case *AuthenticationError:
					assert.Equal(t, tt.expectedCode, e.HuefyError.Code)
				case *TemplateNotFoundError:
					assert.Equal(t, tt.expectedCode, e.HuefyError.Code)
					assert.Equal(t, "welcome-email", e.TemplateKey)
				case *InvalidTemplateDataError:
					assert.Equal(t, tt.expectedCode, e.HuefyError.Code)
					assert.Len(t, e.ValidationErrors, 2)
					assert.Contains(t, e.ValidationErrors, "name is required")
					assert.Contains(t, e.ValidationErrors, "email is required")
				case *ProviderError:
					assert.Equal(t, tt.expectedCode, e.HuefyError.Code)
					assert.Equal(t, ProviderSES, e.Provider)
					assert.Equal(t, "MessageRejected", e.ProviderCode)
				case *RateLimitError:
					assert.Equal(t, tt.expectedCode, e.HuefyError.Code)
					assert.Equal(t, 60, e.RetryAfter)
				}
			}
		})
	}
}

func TestErrorTypeCheckers(t *testing.T) {
	// Test with non-Huefy errors
	standardErr := assert.AnError

	assert.False(t, IsHuefyError(standardErr))
	assert.False(t, IsAuthenticationError(standardErr))
	assert.False(t, IsTemplateNotFoundError(standardErr))
	assert.False(t, IsInvalidTemplateDataError(standardErr))
	assert.False(t, IsInvalidRecipientError(standardErr))
	assert.False(t, IsProviderError(standardErr))
	assert.False(t, IsRateLimitError(standardErr))
	assert.False(t, IsNetworkError(standardErr))
	assert.False(t, IsTimeoutError(standardErr))
	assert.False(t, IsValidationError(standardErr))
}

func TestTemplateNotFoundErrorFields(t *testing.T) {
	err := NewTemplateNotFoundError("welcome-email")
	assert.Equal(t, "welcome-email", err.TemplateKey)
	assert.Equal(t, ErrorCodeTemplateNotFound, err.Code)
	assert.Contains(t, err.Details, "templateKey")
	assert.Equal(t, "welcome-email", err.Details["templateKey"])
}

func TestInvalidTemplateDataErrorFields(t *testing.T) {
	validationErrors := []string{"name is required", "email is invalid"}
	err := NewInvalidTemplateDataError("Validation failed", validationErrors)
	assert.Equal(t, validationErrors, err.ValidationErrors)
	assert.Equal(t, ErrorCodeInvalidTemplateData, err.Code)
	assert.Contains(t, err.Details, "validationErrors")
	assert.Equal(t, validationErrors, err.Details["validationErrors"])
}

func TestProviderErrorFields(t *testing.T) {
	err := NewProviderError(ProviderSendGrid, "REJECTED", "Message rejected")
	assert.Equal(t, ProviderSendGrid, err.Provider)
	assert.Equal(t, "REJECTED", err.ProviderCode)
	assert.Equal(t, ErrorCodeProviderError, err.Code)
	assert.Contains(t, err.Details, "provider")
	assert.Contains(t, err.Details, "providerCode")
	assert.Equal(t, ProviderSendGrid, err.Details["provider"])
	assert.Equal(t, "REJECTED", err.Details["providerCode"])
}

func TestRateLimitErrorFields(t *testing.T) {
	err := NewRateLimitError("Too many requests", 120)
	assert.Equal(t, 120, err.RetryAfter)
	assert.Equal(t, ErrorCodeRateLimitExceeded, err.Code)
	assert.Contains(t, err.Details, "retryAfter")
	assert.Equal(t, 120, err.Details["retryAfter"])
}