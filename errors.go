package huefy

import (
	"fmt"
	"net/http"
)

// ErrorCode represents Huefy API error codes
type ErrorCode string

const (
	ErrorCodeAuthenticationFailed  ErrorCode = "AUTHENTICATION_FAILED"
	ErrorCodeTemplateNotFound      ErrorCode = "TEMPLATE_NOT_FOUND"
	ErrorCodeInvalidTemplateData   ErrorCode = "INVALID_TEMPLATE_DATA"
	ErrorCodeInvalidRecipient      ErrorCode = "INVALID_RECIPIENT"
	ErrorCodeProviderError         ErrorCode = "PROVIDER_ERROR"
	ErrorCodeRateLimitExceeded     ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrorCodeValidationFailed      ErrorCode = "VALIDATION_FAILED"
	ErrorCodeInternalServerError   ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrorCodeServiceUnavailable    ErrorCode = "SERVICE_UNAVAILABLE"
	ErrorCodeBadGateway           ErrorCode = "BAD_GATEWAY"
	ErrorCodeTimeout              ErrorCode = "TIMEOUT"
	ErrorCodeNetworkError         ErrorCode = "NETWORK_ERROR"
)

// HuefyError represents the base error type for all Huefy SDK errors
type HuefyError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e *HuefyError) Error() string {
	return fmt.Sprintf("Huefy API error [%s]: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for error wrapping
func (e *HuefyError) Unwrap() error {
	return nil
}

// AuthenticationError represents authentication failures
type AuthenticationError struct {
	*HuefyError
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(message string) *AuthenticationError {
	return &AuthenticationError{
		HuefyError: &HuefyError{
			Code:    ErrorCodeAuthenticationFailed,
			Message: message,
		},
	}
}

// TemplateNotFoundError represents template not found errors
type TemplateNotFoundError struct {
	*HuefyError
	TemplateKey string
}

// NewTemplateNotFoundError creates a new template not found error
func NewTemplateNotFoundError(templateKey string) *TemplateNotFoundError {
	return &TemplateNotFoundError{
		HuefyError: &HuefyError{
			Code:    ErrorCodeTemplateNotFound,
			Message: fmt.Sprintf("Template not found: %s", templateKey),
			Details: map[string]interface{}{"templateKey": templateKey},
		},
		TemplateKey: templateKey,
	}
}

// InvalidTemplateDataError represents invalid template data errors
type InvalidTemplateDataError struct {
	*HuefyError
	ValidationErrors []string
}

// NewInvalidTemplateDataError creates a new invalid template data error
func NewInvalidTemplateDataError(message string, validationErrors []string) *InvalidTemplateDataError {
	return &InvalidTemplateDataError{
		HuefyError: &HuefyError{
			Code:    ErrorCodeInvalidTemplateData,
			Message: message,
			Details: map[string]interface{}{"validationErrors": validationErrors},
		},
		ValidationErrors: validationErrors,
	}
}

// InvalidRecipientError represents invalid recipient errors
type InvalidRecipientError struct {
	*HuefyError
	Recipient string
}

// NewInvalidRecipientError creates a new invalid recipient error
func NewInvalidRecipientError(message string) *InvalidRecipientError {
	return &InvalidRecipientError{
		HuefyError: &HuefyError{
			Code:    ErrorCodeInvalidRecipient,
			Message: message,
		},
	}
}

// ProviderError represents email provider errors
type ProviderError struct {
	*HuefyError
	Provider     EmailProvider
	ProviderCode string
}

// NewProviderError creates a new provider error
func NewProviderError(provider EmailProvider, providerCode, message string) *ProviderError {
	return &ProviderError{
		HuefyError: &HuefyError{
			Code:    ErrorCodeProviderError,
			Message: message,
			Details: map[string]interface{}{
				"provider":     provider,
				"providerCode": providerCode,
			},
		},
		Provider:     provider,
		ProviderCode: providerCode,
	}
}

// RateLimitError represents rate limiting errors
type RateLimitError struct {
	*HuefyError
	RetryAfter int
}

// NewRateLimitError creates a new rate limit error
func NewRateLimitError(message string, retryAfter int) *RateLimitError {
	return &RateLimitError{
		HuefyError: &HuefyError{
			Code:    ErrorCodeRateLimitExceeded,
			Message: message,
			Details: map[string]interface{}{"retryAfter": retryAfter},
		},
		RetryAfter: retryAfter,
	}
}

// NetworkError represents network-related errors
type NetworkError struct {
	*HuefyError
}

// NewNetworkError creates a new network error
func NewNetworkError(message string) *NetworkError {
	return &NetworkError{
		HuefyError: &HuefyError{
			Code:    ErrorCodeNetworkError,
			Message: message,
		},
	}
}

// TimeoutError represents timeout errors
type TimeoutError struct {
	*HuefyError
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(message string) *TimeoutError {
	return &TimeoutError{
		HuefyError: &HuefyError{
			Code:    ErrorCodeTimeout,
			Message: message,
		},
	}
}

// ValidationError represents validation errors
type ValidationError struct {
	*HuefyError
}

// NewValidationError creates a new validation error
func NewValidationError(message string) *ValidationError {
	return &ValidationError{
		HuefyError: &HuefyError{
			Code:    ErrorCodeValidationFailed,
			Message: message,
		},
	}
}

// ErrorDetail represents error details from API responses
type ErrorDetail struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// createErrorFromResponse creates appropriate error types from API responses
func createErrorFromResponse(errorResp *ErrorResponse, statusCode int) error {
	errorCode := ErrorCode(errorResp.Error.Code)
	message := errorResp.Error.Message
	details := errorResp.Error.Details

	// Create specific error types based on error code
	switch errorCode {
	case ErrorCodeAuthenticationFailed:
		return &AuthenticationError{
			HuefyError: &HuefyError{
				Code:    errorCode,
				Message: message,
				Details: details,
			},
		}

	case ErrorCodeTemplateNotFound:
		templateKey := ""
		if tk, ok := details["templateKey"].(string); ok {
			templateKey = tk
		}
		return &TemplateNotFoundError{
			HuefyError: &HuefyError{
				Code:    errorCode,
				Message: message,
				Details: details,
			},
			TemplateKey: templateKey,
		}

	case ErrorCodeInvalidTemplateData:
		var validationErrors []string
		if ve, ok := details["validationErrors"].([]interface{}); ok {
			for _, v := range ve {
				if s, ok := v.(string); ok {
					validationErrors = append(validationErrors, s)
				}
			}
		}
		return &InvalidTemplateDataError{
			HuefyError: &HuefyError{
				Code:    errorCode,
				Message: message,
				Details: details,
			},
			ValidationErrors: validationErrors,
		}

	case ErrorCodeInvalidRecipient:
		return &InvalidRecipientError{
			HuefyError: &HuefyError{
				Code:    errorCode,
				Message: message,
				Details: details,
			},
		}

	case ErrorCodeProviderError:
		provider := EmailProvider("")
		providerCode := ""
		if p, ok := details["provider"].(string); ok {
			provider = EmailProvider(p)
		}
		if pc, ok := details["providerCode"].(string); ok {
			providerCode = pc
		}
		return &ProviderError{
			HuefyError: &HuefyError{
				Code:    errorCode,
				Message: message,
				Details: details,
			},
			Provider:     provider,
			ProviderCode: providerCode,
		}

	case ErrorCodeRateLimitExceeded:
		retryAfter := 0
		if ra, ok := details["retryAfter"].(float64); ok {
			retryAfter = int(ra)
		}
		return &RateLimitError{
			HuefyError: &HuefyError{
				Code:    errorCode,
				Message: message,
				Details: details,
			},
			RetryAfter: retryAfter,
		}

	case ErrorCodeValidationFailed:
		return &ValidationError{
			HuefyError: &HuefyError{
				Code:    errorCode,
				Message: message,
				Details: details,
			},
		}

	case ErrorCodeTimeout:
		return &TimeoutError{
			HuefyError: &HuefyError{
				Code:    errorCode,
				Message: message,
				Details: details,
			},
		}

	case ErrorCodeNetworkError:
		return &NetworkError{
			HuefyError: &HuefyError{
				Code:    errorCode,
				Message: message,
				Details: details,
			},
		}

	default:
		// Return generic HuefyError for unknown error codes
		return &HuefyError{
			Code:    errorCode,
			Message: message,
			Details: details,
		}
	}
}

// IsHuefyError checks if an error is a Huefy error
func IsHuefyError(err error) bool {
	_, ok := err.(*HuefyError)
	return ok
}

// IsAuthenticationError checks if an error is an authentication error
func IsAuthenticationError(err error) bool {
	_, ok := err.(*AuthenticationError)
	return ok
}

// IsTemplateNotFoundError checks if an error is a template not found error
func IsTemplateNotFoundError(err error) bool {
	_, ok := err.(*TemplateNotFoundError)
	return ok
}

// IsInvalidTemplateDataError checks if an error is an invalid template data error
func IsInvalidTemplateDataError(err error) bool {
	_, ok := err.(*InvalidTemplateDataError)
	return ok
}

// IsInvalidRecipientError checks if an error is an invalid recipient error
func IsInvalidRecipientError(err error) bool {
	_, ok := err.(*InvalidRecipientError)
	return ok
}

// IsProviderError checks if an error is a provider error
func IsProviderError(err error) bool {
	_, ok := err.(*ProviderError)
	return ok
}

// IsRateLimitError checks if an error is a rate limit error
func IsRateLimitError(err error) bool {
	_, ok := err.(*RateLimitError)
	return ok
}

// IsNetworkError checks if an error is a network error
func IsNetworkError(err error) bool {
	_, ok := err.(*NetworkError)
	return ok
}

// IsTimeoutError checks if an error is a timeout error
func IsTimeoutError(err error) bool {
	_, ok := err.(*TimeoutError)
	return ok
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}