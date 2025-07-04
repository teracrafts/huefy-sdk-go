package huefy

import (
	"github.com/teracrafts/huefy-sdk/core/kernel"
)

// ErrorCode re-exports the core ErrorCode type
type ErrorCode = core.ErrorCode

const (
	ErrorCodeAuthenticationFailed  = core.ErrorCodeAuthenticationFailed
	ErrorCodeTemplateNotFound      = core.ErrorCodeTemplateNotFound
	ErrorCodeInvalidTemplateData   = core.ErrorCodeInvalidTemplateData
	ErrorCodeInvalidRecipient      = core.ErrorCodeInvalidRecipient
	ErrorCodeProviderError         = core.ErrorCodeProviderError
	ErrorCodeRateLimitExceeded     = core.ErrorCodeRateLimitExceeded
	ErrorCodeValidationFailed      = core.ErrorCodeValidationFailed
	ErrorCodeInternalServerError   = core.ErrorCodeInternalServerError
	ErrorCodeServiceUnavailable    = core.ErrorCodeServiceUnavailable
	ErrorCodeBadGateway           = core.ErrorCodeBadGateway
	ErrorCodeTimeout              = core.ErrorCodeTimeout
	ErrorCodeNetworkError         = core.ErrorCodeNetworkError
)

// HuefyError re-exports the core HuefyError type
type HuefyError = core.HuefyError

// AuthenticationError re-exports the core AuthenticationError type
type AuthenticationError = core.AuthenticationError

// NewAuthenticationError re-exports the core constructor
var NewAuthenticationError = core.NewAuthenticationError

// TemplateNotFoundError re-exports the core TemplateNotFoundError type
type TemplateNotFoundError = core.TemplateNotFoundError

// NewTemplateNotFoundError re-exports the core constructor
var NewTemplateNotFoundError = core.NewTemplateNotFoundError

// InvalidTemplateDataError re-exports the core InvalidTemplateDataError type
type InvalidTemplateDataError = core.InvalidTemplateDataError

// NewInvalidTemplateDataError re-exports the core constructor
var NewInvalidTemplateDataError = core.NewInvalidTemplateDataError

// InvalidRecipientError re-exports the core InvalidRecipientError type
type InvalidRecipientError = core.InvalidRecipientError

// NewInvalidRecipientError re-exports the core constructor
var NewInvalidRecipientError = core.NewInvalidRecipientError

// ProviderError re-exports the core ProviderError type
type ProviderError = core.ProviderError

// NewProviderError re-exports the core constructor
var NewProviderError = core.NewProviderError

// RateLimitError re-exports the core RateLimitError type
type RateLimitError = core.RateLimitError

// NewRateLimitError re-exports the core constructor
var NewRateLimitError = core.NewRateLimitError

// NetworkError re-exports the core NetworkError type
type NetworkError = core.NetworkError

// NewNetworkError re-exports the core constructor
var NewNetworkError = core.NewNetworkError

// TimeoutError re-exports the core TimeoutError type
type TimeoutError = core.TimeoutError

// NewTimeoutError re-exports the core constructor
var NewTimeoutError = core.NewTimeoutError

// ValidationError re-exports the core ValidationError type
type ValidationError = core.ValidationError

// NewValidationError re-exports the core constructor
var NewValidationError = core.NewValidationError

// ErrorDetail re-exports the core ErrorDetail type
type ErrorDetail = core.ErrorDetail

// ErrorResponse re-exports the core ErrorResponse type
type ErrorResponse = core.ErrorResponse


