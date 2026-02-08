// Package huefy provides a Go SDK for the Huefy email sending platform.
//
// The Huefy Go SDK allows you to send template-based emails through the Huefy API
// with support for multiple email providers, retry logic, and comprehensive error handling.
//
// Basic usage:
//
//	client := huefy.NewClient("your-api-key")
//	resp, err := client.SendEmail(context.Background(), &huefy.SendEmailRequest{
//		TemplateKey: "welcome-email",
//		Data: map[string]interface{}{
//			"name":    "John Doe",
//			"company": "Acme Corp",
//		},
//		Recipient: "john@example.com",
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Email sent: %s\n", resp.MessageID)
package huefy

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/teracrafts/huefy-sdk-go/v2/internal/pb/sdk/v1"
)

// EmailProvider represents supported email providers
type EmailProvider string

const (
	ProviderSES       EmailProvider = "ses"
	ProviderSendGrid  EmailProvider = "sendgrid"
	ProviderMailgun   EmailProvider = "mailgun"
	ProviderMailchimp EmailProvider = "mailchimp"
)

// Production and local gRPC endpoints
const (
	ProductionGRPCEndpoint = "api.huefy.dev:50051"
	LocalGRPCEndpoint      = "localhost:50051"
)

// Client represents the Huefy SDK client
type Client struct {
	apiKey      string
	endpoint    string
	conn        *grpc.ClientConn
	grpcClient  pb.SDKServiceClient
	retryConfig *RetryConfig
}

// RetryConfig configures retry behavior for failed requests
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Multiplier float64
}

// DefaultRetryConfig provides sensible defaults for retry behavior
var DefaultRetryConfig = &RetryConfig{
	MaxRetries: 3,
	BaseDelay:  time.Second,
	MaxDelay:   30 * time.Second,
	Multiplier: 2.0,
}

// ClientOption represents an option for configuring the client
type ClientOption func(*clientOptions)

type clientOptions struct {
	endpoint    string
	retryConfig *RetryConfig
	local       bool
}

// WithEndpoint sets a custom gRPC endpoint for the API (overrides local setting)
func WithEndpoint(endpoint string) ClientOption {
	return func(o *clientOptions) {
		o.endpoint = endpoint
	}
}

// WithRetryConfig sets custom retry configuration
func WithRetryConfig(config *RetryConfig) ClientOption {
	return func(o *clientOptions) {
		o.retryConfig = config
	}
}

// WithLocal uses local development endpoints instead of production
func WithLocal() ClientOption {
	return func(o *clientOptions) {
		o.local = true
	}
}

// NewClient creates a new Huefy SDK client
func NewClient(apiKey string, opts ...ClientOption) *Client {
	options := &clientOptions{
		retryConfig: DefaultRetryConfig,
	}

	for _, opt := range opts {
		opt(options)
	}

	// Determine endpoint: custom > local > production
	endpoint := options.endpoint
	if endpoint == "" {
		if options.local {
			endpoint = LocalGRPCEndpoint
		} else {
			endpoint = ProductionGRPCEndpoint
		}
	}

	c := &Client{
		apiKey:      apiKey,
		endpoint:    endpoint,
		retryConfig: options.retryConfig,
	}

	// Establish gRPC connection
	if err := c.connect(); err != nil {
		panic(fmt.Sprintf("Failed to create Huefy client: %v", err))
	}

	return c
}

// connect establishes the gRPC connection
func (c *Client) connect() error {
	var opts []grpc.DialOption

	// Configure TLS for production endpoints
	if !strings.Contains(c.endpoint, "localhost") {
		tlsConfig := &tls.Config{}
		creds := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Configure keepalive parameters
	opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             time.Second,
		PermitWithoutStream: true,
	}))

	// Configure default call options
	opts = append(opts, grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(4*1024*1024), // 4MB
		grpc.MaxCallSendMsgSize(4*1024*1024), // 4MB
	))

	// Establish connection
	conn, err := grpc.NewClient(c.endpoint, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to gRPC server at %s: %w", c.endpoint, err)
	}

	c.conn = conn
	c.grpcClient = pb.NewSDKServiceClient(conn)

	return nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SendEmailRequest represents a request to send an email
type SendEmailRequest struct {
	TemplateKey string                 `json:"templateKey"`
	Data        map[string]interface{} `json:"data"`
	Recipient   string                 `json:"recipient"`
	Provider    *EmailProvider         `json:"providerType,omitempty"`
}

// SendEmailResponse represents the response from sending an email
type SendEmailResponse struct {
	Success   bool          `json:"success"`
	Message   string        `json:"message"`
	MessageID string        `json:"messageId"`
	Provider  EmailProvider `json:"provider"`
}

// BulkEmailRequest represents a request to send multiple emails
type BulkEmailRequest struct {
	Emails []SendEmailRequest `json:"emails"`
}

// BulkEmailResult represents the result of a single email in a bulk operation
type BulkEmailResult struct {
	Success bool               `json:"success"`
	Result  *SendEmailResponse `json:"result,omitempty"`
	Error   *ErrorResponse     `json:"error,omitempty"`
}

// BulkEmailResponse represents the response from sending multiple emails
type BulkEmailResponse struct {
	Results []BulkEmailResult `json:"results"`
}

// HealthResponse represents the API health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version,omitempty"`
}

// SendEmail sends a single email using a template
func (c *Client) SendEmail(ctx context.Context, req *SendEmailRequest) (*SendEmailResponse, error) {
	if req == nil {
		return nil, NewValidationError("request cannot be nil")
	}

	if err := c.validateSendEmailRequest(req); err != nil {
		return nil, err
	}

	pbReq, err := c.convertToProtoSendEmailRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	pbResp, err := c.doGRPCRequest(ctx, func(ctx context.Context) (interface{}, error) {
		return c.grpcClient.SendEmail(ctx, pbReq)
	})
	if err != nil {
		return nil, err
	}

	return c.convertFromProtoSendEmailResponse(pbResp.(*pb.SendEmailResponse)), nil
}

// SendBulkEmails sends multiple emails in a single request
func (c *Client) SendBulkEmails(ctx context.Context, emails []SendEmailRequest) (*BulkEmailResponse, error) {
	if len(emails) == 0 {
		return nil, NewValidationError("emails slice cannot be empty")
	}

	for i, email := range emails {
		if err := c.validateSendEmailRequest(&email); err != nil {
			return nil, fmt.Errorf("validation failed for email %d: %w", i, err)
		}
	}

	pbReqs := make([]*pb.SendEmailRequest, len(emails))
	for i, email := range emails {
		pbReq, err := c.convertToProtoSendEmailRequest(&email)
		if err != nil {
			return nil, fmt.Errorf("failed to convert email %d: %w", i, err)
		}
		pbReqs[i] = pbReq
	}

	bulkReq := &pb.SendBulkEmailRequest{
		Emails: pbReqs,
	}

	pbResp, err := c.doGRPCRequest(ctx, func(ctx context.Context) (interface{}, error) {
		return c.grpcClient.SendBulkEmail(ctx, bulkReq)
	})
	if err != nil {
		return nil, err
	}

	return c.convertFromProtoBulkEmailResponse(pbResp.(*pb.SendBulkEmailResponse)), nil
}

// HealthCheck checks the API health status
func (c *Client) HealthCheck(ctx context.Context) (*HealthResponse, error) {
	req := &pb.HealthCheckRequest{}

	pbResp, err := c.doGRPCRequest(ctx, func(ctx context.Context) (interface{}, error) {
		return c.grpcClient.HealthCheck(ctx, req)
	})
	if err != nil {
		return nil, err
	}

	return c.convertFromProtoHealthResponse(pbResp.(*pb.HealthCheckResponse)), nil
}

// doGRPCRequest performs a gRPC request with retry logic
func (c *Client) doGRPCRequest(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	var lastErr error

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(float64(c.retryConfig.BaseDelay) *
				math.Pow(c.retryConfig.Multiplier, float64(attempt-1)))
			if delay > c.retryConfig.MaxDelay {
				delay = c.retryConfig.MaxDelay
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		// Add API key to gRPC metadata
		ctxWithAuth := metadata.AppendToOutgoingContext(ctx, "x-api-key", c.apiKey)

		// Set deadline if not already set
		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctxWithAuth, cancel = context.WithTimeout(ctxWithAuth, 30*time.Second)
			defer cancel()
		}

		resp, err := fn(ctxWithAuth)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		if !isRetryableGRPCError(err) {
			return nil, convertGRPCError(err)
		}
	}

	return nil, convertGRPCError(lastErr)
}

// validateSendEmailRequest validates a send email request
func (c *Client) validateSendEmailRequest(req *SendEmailRequest) error {
	if req.TemplateKey == "" {
		return NewValidationError("templateKey is required")
	}

	if req.Recipient == "" {
		return NewValidationError("recipient is required")
	}

	if !strings.Contains(req.Recipient, "@") || !strings.Contains(req.Recipient, ".") {
		return NewInvalidRecipientError(fmt.Sprintf("invalid email address: %s", req.Recipient))
	}

	if req.Data == nil {
		return NewValidationError("data is required")
	}

	if req.Provider != nil {
		switch *req.Provider {
		case ProviderSES, ProviderSendGrid, ProviderMailgun, ProviderMailchimp:
			// Valid provider
		default:
			return NewValidationError(fmt.Sprintf("invalid provider: %s", *req.Provider))
		}
	}

	return nil
}

// convertToProtoSendEmailRequest converts our request type to protobuf
func (c *Client) convertToProtoSendEmailRequest(req *SendEmailRequest) (*pb.SendEmailRequest, error) {
	data, err := structpb.NewStruct(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert data to protobuf struct: %w", err)
	}

	pbReq := &pb.SendEmailRequest{
		TemplateKey: req.TemplateKey,
		Data:        data,
		Recipient: &pb.EmailRecipient{
			Email: req.Recipient,
			Type:  pb.RecipientType_RECIPIENT_TYPE_TO,
		},
	}

	if req.Provider != nil {
		providerType := string(*req.Provider)
		pbReq.ProviderType = &providerType
	}

	return pbReq, nil
}

// convertFromProtoSendEmailResponse converts protobuf response to our type
func (c *Client) convertFromProtoSendEmailResponse(pbResp *pb.SendEmailResponse) *SendEmailResponse {
	return &SendEmailResponse{
		Success:   pbResp.Success,
		Message:   pbResp.Message,
		MessageID: pbResp.MessageId,
		Provider:  EmailProvider(pbResp.Provider),
	}
}

// convertFromProtoBulkEmailResponse converts protobuf bulk response to our type
func (c *Client) convertFromProtoBulkEmailResponse(pbResp *pb.SendBulkEmailResponse) *BulkEmailResponse {
	results := make([]BulkEmailResult, len(pbResp.Results))

	for i, pbResult := range pbResp.Results {
		result := BulkEmailResult{
			Success: pbResult.Success,
		}

		if pbResult.Response != nil {
			result.Result = c.convertFromProtoSendEmailResponse(pbResult.Response)
		}

		if pbResult.Error != nil {
			result.Error = &ErrorResponse{
				Error: ErrorDetail{
					Code:    pbResult.Error.Code,
					Message: pbResult.Error.Message,
				},
			}
		}

		results[i] = result
	}

	return &BulkEmailResponse{
		Results: results,
	}
}

// convertFromProtoHealthResponse converts protobuf health response to our type
func (c *Client) convertFromProtoHealthResponse(pbResp *pb.HealthCheckResponse) *HealthResponse {
	timestamp := time.Now()
	if pbResp.Timestamp != nil {
		timestamp = pbResp.Timestamp.AsTime()
	}

	return &HealthResponse{
		Status:    pbResp.Status,
		Timestamp: timestamp,
		Version:   pbResp.Version,
	}
}

// isRetryableGRPCError determines if a gRPC error should trigger a retry
func isRetryableGRPCError(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}

// convertGRPCError converts gRPC errors to our SDK error types
func convertGRPCError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return NewNetworkError(fmt.Sprintf("non-gRPC error: %v", err))
	}

	switch st.Code() {
	case codes.InvalidArgument:
		return NewValidationError(st.Message())
	case codes.Unauthenticated:
		return NewAuthenticationError(st.Message())
	case codes.PermissionDenied:
		return NewAuthenticationError(st.Message())
	case codes.NotFound:
		return NewTemplateNotFoundError(st.Message())
	case codes.ResourceExhausted:
		return NewRateLimitError(st.Message(), 0)
	case codes.Unavailable:
		return NewProviderError("", "", st.Message())
	case codes.DeadlineExceeded:
		return NewTimeoutError(st.Message())
	case codes.Internal:
		return NewHuefyError(st.Message(), "INTERNAL_ERROR")
	default:
		return NewHuefyError(st.Message(), fmt.Sprintf("GRPC_%s", st.Code().String()))
	}
}
