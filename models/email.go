package models

// EmailProvider represents supported email providers.
type EmailProvider string

const (
	ProviderSES       EmailProvider = "ses"
	ProviderSendGrid  EmailProvider = "sendgrid"
	ProviderMailgun   EmailProvider = "mailgun"
	ProviderMailchimp EmailProvider = "mailchimp"
)

// SendEmailRequest represents a request to send a single email.
type SendEmailRequest struct {
	TemplateKey  string            `json:"templateKey"`
	Recipient    string            `json:"recipient"`
	Data         map[string]string `json:"data"`
	ProviderType *EmailProvider    `json:"providerType,omitempty"`
}

// SendEmailResponse represents the response from sending an email.
type SendEmailResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	MessageID string `json:"messageId"`
	Provider  string `json:"provider"`
}

// BulkEmailResult represents the result of a single email in a bulk operation.
type BulkEmailResult struct {
	Email   string             `json:"email"`
	Success bool               `json:"success"`
	Result  *SendEmailResponse `json:"result,omitempty"`
	Error   *BulkEmailError    `json:"error,omitempty"`
}

// BulkEmailError represents an error for a single email in a bulk operation.
type BulkEmailError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

// BulkEmailResponse represents the response from a bulk email operation.
type BulkEmailResponse struct {
	Results []BulkEmailResult `json:"results"`
}
