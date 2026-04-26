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
	Data         map[string]any    `json:"data"`
	ProviderType *EmailProvider    `json:"providerType,omitempty"`
}

// RecipientStatus represents the delivery status of a single recipient.
type RecipientStatus struct {
	Email     string  `json:"email"`
	Status    string  `json:"status"`
	MessageID string  `json:"messageId,omitempty"`
	Error     string  `json:"error,omitempty"`
	SentAt    *string `json:"sentAt,omitempty"`
}

// SendEmailResponseData is the data payload from a send-email response.
type SendEmailResponseData struct {
	EmailID     string            `json:"emailId"`
	Status      string            `json:"status"`
	Recipients  []RecipientStatus `json:"recipients"`
	ScheduledAt *string           `json:"scheduledAt,omitempty"`
	SentAt      *string           `json:"sentAt,omitempty"`
}

// SendEmailResponse represents the response from sending an email.
type SendEmailResponse struct {
	Success       bool                  `json:"success"`
	Data          SendEmailResponseData `json:"data"`
	CorrelationID string                `json:"correlationId"`
}

// BulkRecipient represents a single recipient in a bulk email send.
type BulkRecipient struct {
	Email string                 `json:"email"`
	Type  string                 `json:"type,omitempty"`
	Data  map[string]interface{} `json:"data,omitempty"`
}

// SendBulkEmailsRequest represents a request to send bulk emails via a template.
type SendBulkEmailsRequest struct {
	TemplateKey          string                 `json:"templateKey"`
	Recipients           []BulkRecipient        `json:"recipients"`
	FromEmail            string                 `json:"fromEmail,omitempty"`
	FromName             string                 `json:"fromName,omitempty"`
	ProviderType         string                 `json:"providerType,omitempty"`
	BatchSize            int                    `json:"batchSize,omitempty"`
	CorrelationID        string                 `json:"correlationId,omitempty"`
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
}

// SendBulkEmailsResponseData is the data payload from a send-bulk-emails response.
type SendBulkEmailsResponseData struct {
	BatchID         string            `json:"batchId"`
	Status          string            `json:"status"`
	TemplateKey     string            `json:"templateKey"`
	TotalRecipients int               `json:"totalRecipients"`
	ProcessedCount  int               `json:"processedCount"`
	SuccessCount    int               `json:"successCount"`
	FailureCount    int               `json:"failureCount"`
	SuppressedCount int               `json:"suppressedCount"`
	StartedAt       string            `json:"startedAt"`
	CompletedAt     *string           `json:"completedAt,omitempty"`
	Recipients      []RecipientStatus `json:"recipients"`
}

// SendBulkEmailsResponse represents the response from a bulk email operation.
type SendBulkEmailsResponse struct {
	Success       bool                       `json:"success"`
	Data          SendBulkEmailsResponseData `json:"data"`
	CorrelationID string                     `json:"correlationId"`
}

// HealthResponseData is the data payload from a health check response.
type HealthResponseData struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

// HealthResponse represents the full health check API response.
type HealthResponse struct {
	Success       bool               `json:"success"`
	Data          HealthResponseData `json:"data"`
	CorrelationID string             `json:"correlationId"`
}
