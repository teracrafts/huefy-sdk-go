package huefy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		opts     []ClientOption
		expected *Client
	}{
		{
			name:   "default client",
			apiKey: "test-key",
			expected: &Client{
				apiKey:      "test-key",
				baseURL:     "https://api.huefy.com",
				retryConfig: DefaultRetryConfig,
			},
		},
		{
			name:   "custom base URL",
			apiKey: "test-key",
			opts:   []ClientOption{WithBaseURL("https://custom.api.com/")},
			expected: &Client{
				apiKey:      "test-key",
				baseURL:     "https://custom.api.com",
				retryConfig: DefaultRetryConfig,
			},
		},
		{
			name:   "custom retry config",
			apiKey: "test-key",
			opts: []ClientOption{WithRetryConfig(&RetryConfig{
				MaxRetries: 5,
				BaseDelay:  2 * time.Second,
			})},
			expected: &Client{
				apiKey:  "test-key",
				baseURL: "https://api.huefy.com",
				retryConfig: &RetryConfig{
					MaxRetries: 5,
					BaseDelay:  2 * time.Second,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.apiKey, tt.opts...)
			assert.Equal(t, tt.expected.apiKey, client.apiKey)
			assert.Equal(t, tt.expected.baseURL, client.baseURL)
			assert.Equal(t, tt.expected.retryConfig, client.retryConfig)
			assert.NotNil(t, client.httpClient)
		})
	}
}

func TestClient_SendEmail(t *testing.T) {
	tests := []struct {
		name           string
		request        *SendEmailRequest
		responseStatus int
		responseBody   interface{}
		expectedError  string
		expected       *SendEmailResponse
	}{
		{
			name: "successful send",
			request: &SendEmailRequest{
				TemplateKey: "welcome-email",
				Data:        map[string]interface{}{"name": "John"},
				Recipient:   "john@example.com",
			},
			responseStatus: http.StatusOK,
			responseBody: &SendEmailResponse{
				MessageID: "msg-123",
				Status:    "sent",
				Provider:  ProviderSES,
				Timestamp: time.Now(),
			},
			expected: &SendEmailResponse{
				MessageID: "msg-123",
				Status:    "sent",
				Provider:  ProviderSES,
			},
		},
		{
			name:           "nil request",
			request:        nil,
			expectedError:  "request cannot be nil",
		},
		{
			name: "missing template key",
			request: &SendEmailRequest{
				Data:      map[string]interface{}{"name": "John"},
				Recipient: "john@example.com",
			},
			expectedError: "templateKey is required",
		},
		{
			name: "missing recipient",
			request: &SendEmailRequest{
				TemplateKey: "welcome-email",
				Data:        map[string]interface{}{"name": "John"},
			},
			expectedError: "recipient is required",
		},
		{
			name: "invalid email",
			request: &SendEmailRequest{
				TemplateKey: "welcome-email",
				Data:        map[string]interface{}{"name": "John"},
				Recipient:   "invalid-email",
			},
			expectedError: "invalid email address: invalid-email",
		},
		{
			name: "missing data",
			request: &SendEmailRequest{
				TemplateKey: "welcome-email",
				Recipient:   "john@example.com",
			},
			expectedError: "data is required",
		},
		{
			name: "invalid provider",
			request: &SendEmailRequest{
				TemplateKey: "welcome-email",
				Data:        map[string]interface{}{"name": "John"},
				Recipient:   "john@example.com",
				Provider:    (*EmailProvider)(stringPtr("invalid")),
			},
			expectedError: "invalid provider: invalid",
		},
		{
			name: "authentication error",
			request: &SendEmailRequest{
				TemplateKey: "welcome-email",
				Data:        map[string]interface{}{"name": "John"},
				Recipient:   "john@example.com",
			},
			responseStatus: http.StatusUnauthorized,
			responseBody: &ErrorResponse{
				Error: ErrorDetail{
					Code:    string(ErrorCodeAuthenticationFailed),
					Message: "Invalid API key",
				},
			},
			expectedError: "Huefy API error [AUTHENTICATION_FAILED]: Invalid API key",
		},
		{
			name: "template not found",
			request: &SendEmailRequest{
				TemplateKey: "nonexistent-template",
				Data:        map[string]interface{}{"name": "John"},
				Recipient:   "john@example.com",
			},
			responseStatus: http.StatusNotFound,
			responseBody: &ErrorResponse{
				Error: ErrorDetail{
					Code:    string(ErrorCodeTemplateNotFound),
					Message: "Template not found: nonexistent-template",
					Details: map[string]interface{}{"templateKey": "nonexistent-template"},
				},
			},
			expectedError: "Huefy API error [TEMPLATE_NOT_FOUND]: Template not found: nonexistent-template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request headers
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Equal(t, "application/json", r.Header.Get("Accept"))
				assert.Equal(t, "test-api-key", r.Header.Get("X-API-Key"))
				assert.Contains(t, r.Header.Get("User-Agent"), "Huefy-Go-SDK")

				// Set response status
				w.WriteHeader(tt.responseStatus)

				// Write response body
				if tt.responseBody != nil {
					jsonData, _ := json.Marshal(tt.responseBody)
					w.Write(jsonData)
				}
			}))
			defer server.Close()

			// Create client
			client := NewClient("test-api-key", WithBaseURL(server.URL))

			// Call SendEmail
			response, err := client.SendEmail(context.Background(), tt.request)

			// Check error
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, response)
			} else {
				require.NoError(t, err)
				require.NotNil(t, response)
				assert.Equal(t, tt.expected.MessageID, response.MessageID)
				assert.Equal(t, tt.expected.Status, response.Status)
				assert.Equal(t, tt.expected.Provider, response.Provider)
			}
		})
	}
}

func TestClient_SendBulkEmails(t *testing.T) {
	tests := []struct {
		name           string
		emails         []SendEmailRequest
		responseStatus int
		responseBody   interface{}
		expectedError  string
		expected       *BulkEmailResponse
	}{
		{
			name: "successful bulk send",
			emails: []SendEmailRequest{
				{
					TemplateKey: "welcome-email",
					Data:        map[string]interface{}{"name": "John"},
					Recipient:   "john@example.com",
				},
				{
					TemplateKey: "welcome-email",
					Data:        map[string]interface{}{"name": "Jane"},
					Recipient:   "jane@example.com",
				},
			},
			responseStatus: http.StatusOK,
			responseBody: &BulkEmailResponse{
				Results: []BulkEmailResult{
					{
						Success: true,
						Result: &SendEmailResponse{
							MessageID: "msg-123",
							Status:    "sent",
							Provider:  ProviderSES,
						},
					},
					{
						Success: true,
						Result: &SendEmailResponse{
							MessageID: "msg-456",
							Status:    "sent",
							Provider:  ProviderSES,
						},
					},
				},
			},
			expected: &BulkEmailResponse{
				Results: []BulkEmailResult{
					{Success: true},
					{Success: true},
				},
			},
		},
		{
			name:          "empty emails slice",
			emails:        []SendEmailRequest{},
			expectedError: "emails slice cannot be empty",
		},
		{
			name: "invalid email in slice",
			emails: []SendEmailRequest{
				{
					TemplateKey: "welcome-email",
					Data:        map[string]interface{}{"name": "John"},
					Recipient:   "john@example.com",
				},
				{
					TemplateKey: "welcome-email",
					Data:        map[string]interface{}{"name": "Jane"},
					Recipient:   "invalid-email",
				},
			},
			expectedError: "validation failed for email 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseStatus)
				if tt.responseBody != nil {
					jsonData, _ := json.Marshal(tt.responseBody)
					w.Write(jsonData)
				}
			}))
			defer server.Close()

			// Create client
			client := NewClient("test-api-key", WithBaseURL(server.URL))

			// Call SendBulkEmails
			response, err := client.SendBulkEmails(context.Background(), tt.emails)

			// Check error
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, response)
			} else {
				require.NoError(t, err)
				require.NotNil(t, response)
				assert.Len(t, response.Results, len(tt.expected.Results))
				for i, result := range response.Results {
					assert.Equal(t, tt.expected.Results[i].Success, result.Success)
				}
			}
		})
	}
}

func TestClient_HealthCheck(t *testing.T) {
	expectedResponse := &HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/sdk/health", r.URL.Path)

		w.WriteHeader(http.StatusOK)
		jsonData, _ := json.Marshal(expectedResponse)
		w.Write(jsonData)
	}))
	defer server.Close()

	// Create client
	client := NewClient("test-api-key", WithBaseURL(server.URL))

	// Call HealthCheck
	response, err := client.HealthCheck(context.Background())

	// Verify response
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, expectedResponse.Status, response.Status)
	assert.Equal(t, expectedResponse.Version, response.Version)
}

func TestClient_RetryLogic(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// Return server error for first 2 attempts
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":{"code":"INTERNAL_SERVER_ERROR","message":"Server error"}}`))
		} else {
			// Return success on 3rd attempt
			w.WriteHeader(http.StatusOK)
			jsonData, _ := json.Marshal(&SendEmailResponse{
				MessageID: "msg-123",
				Status:    "sent",
				Provider:  ProviderSES,
			})
			w.Write(jsonData)
		}
	}))
	defer server.Close()

	// Create client with custom retry config
	client := NewClient("test-api-key", 
		WithBaseURL(server.URL),
		WithRetryConfig(&RetryConfig{
			MaxRetries: 3,
			BaseDelay:  10 * time.Millisecond,
			MaxDelay:   100 * time.Millisecond,
			Multiplier: 2.0,
		}),
	)

	// Call SendEmail
	request := &SendEmailRequest{
		TemplateKey: "welcome-email",
		Data:        map[string]interface{}{"name": "John"},
		Recipient:   "john@example.com",
	}

	start := time.Now()
	response, err := client.SendEmail(context.Background(), request)
	duration := time.Since(start)

	// Verify success after retries
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "msg-123", response.MessageID)
	assert.Equal(t, 3, attempts)
	
	// Verify that retries took some time (delays)
	assert.Greater(t, duration, 10*time.Millisecond)
}

func TestClient_ContextCancellation(t *testing.T) {
	// Create server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client
	client := NewClient("test-api-key", WithBaseURL(server.URL))

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Call SendEmail with timeout context
	request := &SendEmailRequest{
		TemplateKey: "welcome-email",
		Data:        map[string]interface{}{"name": "John"},
		Recipient:   "john@example.com",
	}

	_, err := client.SendEmail(ctx, request)

	// Verify context cancellation
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "context deadline exceeded") || 
		strings.Contains(err.Error(), "request failed"))
}

func TestValidateEmailRequest(t *testing.T) {
	client := NewClient("test-key")

	tests := []struct {
		name    string
		request *SendEmailRequest
		wantErr string
	}{
		{
			name: "valid request",
			request: &SendEmailRequest{
				TemplateKey: "welcome",
				Data:        map[string]interface{}{"name": "John"},
				Recipient:   "john@example.com",
			},
			wantErr: "",
		},
		{
			name: "valid request with provider",
			request: &SendEmailRequest{
				TemplateKey: "welcome",
				Data:        map[string]interface{}{"name": "John"},
				Recipient:   "john@example.com",
				Provider:    (*EmailProvider)(stringPtr(string(ProviderSendGrid))),
			},
			wantErr: "",
		},
		{
			name: "empty template key",
			request: &SendEmailRequest{
				Data:      map[string]interface{}{"name": "John"},
				Recipient: "john@example.com",
			},
			wantErr: "templateKey is required",
		},
		{
			name: "empty recipient",
			request: &SendEmailRequest{
				TemplateKey: "welcome",
				Data:        map[string]interface{}{"name": "John"},
			},
			wantErr: "recipient is required",
		},
		{
			name: "invalid email format",
			request: &SendEmailRequest{
				TemplateKey: "welcome",
				Data:        map[string]interface{}{"name": "John"},
				Recipient:   "invalid-email",
			},
			wantErr: "invalid email address: invalid-email",
		},
		{
			name: "nil data",
			request: &SendEmailRequest{
				TemplateKey: "welcome",
				Recipient:   "john@example.com",
			},
			wantErr: "data is required",
		},
		{
			name: "invalid provider",
			request: &SendEmailRequest{
				TemplateKey: "welcome",
				Data:        map[string]interface{}{"name": "John"},
				Recipient:   "john@example.com",
				Provider:    (*EmailProvider)(stringPtr("invalid")),
			},
			wantErr: "invalid provider: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.validateSendEmailRequest(tt.request)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}