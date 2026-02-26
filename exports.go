// Package huefy provides the Huefy Go SDK for interacting with the
// Huefy API. It offers a high-level client with built-in retry logic,
// circuit breaking, request signing, and error sanitization.
//
// Basic usage:
//
//	c := client.NewClient("your-api-key")
//	defer c.Close()
//
//	health, err := c.HealthCheck(context.Background())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(health.Status)
package huefy

import (
	"github.com/teracrafts/huefy-go/client"
	"github.com/teracrafts/huefy-go/config"
	"github.com/teracrafts/huefy-go/errors"
	"github.com/teracrafts/huefy-go/models"
	"github.com/teracrafts/huefy-go/security"
	"github.com/teracrafts/huefy-go/types"
)

// Client re-exports.
type Client = client.Client

// NewClient creates a new Huefy API client with the given API key and options.
var NewClient = client.NewClient

// Config re-exports.
type (
	Config              = config.Config
	Option              = config.Option
	RetryConfig         = config.RetryConfig
	CircuitBreakerConfig = config.CircuitBreakerConfig
)

// Option function re-exports.
var (
	WithBaseURL              = config.WithBaseURL
	WithTimeout              = config.WithTimeout
	WithRetryConfig          = config.WithRetryConfig
	WithCircuitBreakerConfig = config.WithCircuitBreakerConfig
	WithLogger               = config.WithLogger
	WithSecondaryAPIKey      = config.WithSecondaryAPIKey
	WithRequestSigning       = config.WithRequestSigning
	WithErrorSanitization    = config.WithErrorSanitization
)

// Error re-exports.
type HuefyError = errors.HuefyError

var (
	NewError          = errors.NewError
	NewErrorWithCause = errors.NewErrorWithCause
	NetworkError      = errors.NetworkError
	AuthenticationError = errors.AuthenticationError
	SecurityError     = errors.SecurityError
	IsRecoverable     = errors.IsRecoverable
)

// Security re-exports.
var (
	DetectPotentialPII      = security.DetectPotentialPII
	WarnIfPotentialPII      = security.WarnIfPotentialPII
	GenerateHMACSHA256      = security.GenerateHMACSHA256
	CreateRequestSignature  = security.CreateRequestSignature
	VerifyRequestSignature  = security.VerifyRequestSignature
	SignPayload             = security.SignPayload
)

type (
	SignedPayload    = security.SignedPayload
	RequestSignature = security.RequestSignature
)

// Email model re-exports.
type (
	EmailProvider     = models.EmailProvider
	SendEmailRequest  = models.SendEmailRequest
	SendEmailResponse = models.SendEmailResponse
	BulkEmailResult   = models.BulkEmailResult
	BulkEmailError    = models.BulkEmailError
	BulkEmailResponse = models.BulkEmailResponse
)

const (
	ProviderSES       = models.ProviderSES
	ProviderSendGrid  = models.ProviderSendGrid
	ProviderMailgun   = models.ProviderMailgun
	ProviderMailchimp = models.ProviderMailchimp
)

// Types re-exports.
type (
	Logger         = types.Logger
	HealthResponse = types.HealthResponse
)

var (
	NewConsoleLogger = types.NewConsoleLogger
	NewNoopLogger    = types.NewNoopLogger
)
