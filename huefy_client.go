package huefy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/teracrafts/huefy-go/client"
	"github.com/teracrafts/huefy-go/config"
	"github.com/teracrafts/huefy-go/models"
	"github.com/teracrafts/huefy-go/security"
	"github.com/teracrafts/huefy-go/validators"
)

// EmailClient extends the base client with email-specific operations.
type EmailClient struct {
	*client.Client
}

// NewEmailClient creates a new Huefy email client.
func NewEmailClient(apiKey string, opts ...config.Option) (*EmailClient, error) {
	c, err := client.NewClient(apiKey, opts...)
	if err != nil {
		return nil, err
	}
	return &EmailClient{
		Client: c,
	}, nil
}

// toAnyMap converts a map[string]string to map[string]any for PII detection.
func toAnyMap(m map[string]string) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// SendEmail sends a single email using a template.
func (c *EmailClient) SendEmail(ctx context.Context, req *models.SendEmailRequest) (*models.SendEmailResponse, error) {
	security.WarnIfPotentialPII(toAnyMap(req.Data), "email template data", c.GetLogger())

	errs := validators.ValidateSendEmailInput(req.TemplateKey, req.Data, req.Recipient)
	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return nil, fmt.Errorf("validation failed: %s", strings.Join(msgs, "; "))
	}

	req.TemplateKey = strings.TrimSpace(req.TemplateKey)
	req.Recipient = strings.TrimSpace(req.Recipient)

	data, err := c.Client.Request(ctx, "POST", "/emails/send", req)
	if err != nil {
		return nil, err
	}

	var resp models.SendEmailResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &resp, nil
}

// SendBulkEmails sends emails to multiple recipients using a single template.
func (c *EmailClient) SendBulkEmails(ctx context.Context, templateKey string, recipients []models.BulkRecipient, opts ...BulkEmailOption) (*models.SendBulkEmailsResponse, error) {
	if err := validators.ValidateBulkCount(len(recipients)); err != nil {
		return nil, err
	}

	for i, r := range recipients {
		if err := validators.ValidateEmail(r.Email); err != nil {
			return nil, fmt.Errorf("recipients[%d]: %w", i, err)
		}
	}

	req := models.SendBulkEmailsRequest{
		TemplateKey: strings.TrimSpace(templateKey),
		Recipients:  recipients,
	}
	for _, opt := range opts {
		opt(&req)
	}

	data, err := c.Client.Request(ctx, "POST", "/emails/send-bulk", req)
	if err != nil {
		return nil, err
	}

	var resp models.SendBulkEmailsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &resp, nil
}

// BulkEmailOption is a functional option for SendBulkEmails.
type BulkEmailOption func(*models.SendBulkEmailsRequest)

// WithFromEmail sets the fromEmail field on a bulk email request.
func WithFromEmail(email string) BulkEmailOption {
	return func(r *models.SendBulkEmailsRequest) {
		r.FromEmail = email
	}
}

// WithFromName sets the fromName field on a bulk email request.
func WithFromName(name string) BulkEmailOption {
	return func(r *models.SendBulkEmailsRequest) {
		r.FromName = name
	}
}

// WithBulkProviderType sets the providerType field on a bulk email request.
func WithBulkProviderType(providerType string) BulkEmailOption {
	return func(r *models.SendBulkEmailsRequest) {
		r.ProviderType = providerType
	}
}

// WithBatchSize sets the batchSize field on a bulk email request.
func WithBatchSize(size int) BulkEmailOption {
	return func(r *models.SendBulkEmailsRequest) {
		r.BatchSize = size
	}
}

// WithBulkMetadata sets the metadata field on a bulk email request.
func WithBulkMetadata(metadata map[string]interface{}) BulkEmailOption {
	return func(r *models.SendBulkEmailsRequest) {
		r.Metadata = metadata
	}
}
