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
	"github.com/teracrafts/huefy-go/types"
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

// SendEmail sends a single email using a template.
func (c *EmailClient) SendEmail(ctx context.Context, req *models.SendEmailRequest) (*models.SendEmailResponse, error) {
	security.WarnIfPotentialPII(req.Data, "email template data", c.GetLogger())
	warnIfPotentialRecipientPII(req.Recipient, c.GetLogger())

	errs := validators.ValidateSendEmailInput(req.TemplateKey, req.Data, req.Recipient)
	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return nil, fmt.Errorf("validation failed: %s", strings.Join(msgs, "; "))
	}

	req.TemplateKey = strings.TrimSpace(req.TemplateKey)
	normalizedRecipient, err := normalizeSendEmailRecipient(req.Recipient)
	if err != nil {
		return nil, err
	}
	req.Recipient = normalizedRecipient

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

func normalizeSendEmailRecipient(recipient any) (any, error) {
	switch value := recipient.(type) {
	case string:
		return strings.TrimSpace(value), nil
	case *string:
		if value == nil {
			return nil, fmt.Errorf("recipient email is required")
		}
		return strings.TrimSpace(*value), nil
	case models.SendEmailRecipient:
		value.Email = strings.TrimSpace(value.Email)
		value.Type = strings.ToLower(strings.TrimSpace(value.Type))
		return value, nil
	case *models.SendEmailRecipient:
		if value == nil {
			return nil, fmt.Errorf("recipient email is required")
		}
		normalized := *value
		normalized.Email = strings.TrimSpace(normalized.Email)
		normalized.Type = strings.ToLower(strings.TrimSpace(normalized.Type))
		return normalized, nil
	case map[string]any:
		normalized := make(map[string]any, len(value))
		for key, entry := range value {
			normalized[key] = entry
		}
		if email, ok := normalized["email"].(string); ok {
			normalized["email"] = strings.TrimSpace(email)
		}
		if recipientType, ok := normalized["type"].(string); ok {
			normalized["type"] = strings.ToLower(strings.TrimSpace(recipientType))
		}
		return normalized, nil
	case map[string]string:
		normalized := make(map[string]any, len(value))
		for key, entry := range value {
			if key == "email" {
				normalized[key] = strings.TrimSpace(entry)
				continue
			}
			if key == "type" {
				normalized[key] = strings.ToLower(strings.TrimSpace(entry))
				continue
			}
			normalized[key] = entry
		}
		return normalized, nil
	default:
		return nil, fmt.Errorf("recipient must be a string or recipient object")
	}
}

func warnIfPotentialRecipientPII(recipient any, logger types.Logger) {
	switch value := recipient.(type) {
	case models.SendEmailRecipient:
		if value.Data != nil {
			security.WarnIfPotentialPII(value.Data, "recipient data", logger)
		}
	case *models.SendEmailRecipient:
		if value != nil && value.Data != nil {
			security.WarnIfPotentialPII(value.Data, "recipient data", logger)
		}
	case map[string]any:
		if data, ok := value["data"].(map[string]any); ok {
			security.WarnIfPotentialPII(data, "recipient data", logger)
		}
	case map[string]string:
		data := map[string]any{}
		for key, entry := range value {
			if key == "data" {
				data[key] = entry
			}
		}
		if len(data) > 0 {
			security.WarnIfPotentialPII(data, "recipient data", logger)
		}
	}
}

// SendBulkEmails sends emails to multiple recipients using a single template.
func (c *EmailClient) SendBulkEmails(ctx context.Context, req *models.SendBulkEmailsRequest) (*models.SendBulkEmailsResponse, error) {
	if err := validators.ValidateBulkCount(len(req.Recipients)); err != nil {
		return nil, err
	}

	if err := validators.ValidateTemplateKey(req.TemplateKey); err != nil {
		return nil, err
	}

	normalizedRecipients := make([]models.BulkRecipient, len(req.Recipients))
	for i, r := range req.Recipients {
		if err := validators.ValidateBulkRecipient(r); err != nil {
			return nil, fmt.Errorf("recipients[%d]: %w", i, err)
		}

		normalizedRecipients[i] = models.BulkRecipient{
			Email: strings.TrimSpace(r.Email),
			Type:  strings.ToLower(strings.TrimSpace(r.Type)),
			Data:  r.Data,
		}
	}

	req.TemplateKey = strings.TrimSpace(req.TemplateKey)
	req.Recipients = normalizedRecipients

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
