package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	huefy "github.com/teracrafts/huefy-go"
)

const (
	green = "\033[32m"
	red   = "\033[31m"
	reset = "\033[0m"
)

var (
	passed int
	failed int
)

type capturedRequest struct {
	Method string
	Path   string
	Header http.Header
	Body   map[string]any
}

type liveConfig struct {
	APIKey      string
	BaseURL     string
	TemplateKey string
	Recipient   string
	Provider    string
}

type stubTransport struct {
	mu       sync.Mutex
	requests []capturedRequest
}

func newStubTransport() *stubTransport {
	return &stubTransport{}
}

func (s *stubTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var body map[string]any
	if r.Body != nil {
		defer r.Body.Close()
		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			return s.jsonResponse(http.StatusInternalServerError, `{"message":"failed to read body"}`), nil
		}
		if len(rawBody) > 0 {
			if err := json.Unmarshal(rawBody, &body); err != nil {
				return s.jsonResponse(http.StatusBadRequest, `{"message":"invalid json"}`), nil
			}
		}
	}

	s.mu.Lock()
	s.requests = append(s.requests, capturedRequest{
		Method: r.Method,
		Path:   r.URL.Path,
		Header: r.Header.Clone(),
		Body:   body,
	})
	s.mu.Unlock()

	switch r.URL.Path {
	case "/emails/send":
		return s.jsonResponse(http.StatusOK, `{
				"success": true,
				"data": {
					"emailId": "email_123",
					"status": "queued",
					"recipients": [
						{"email": "user@example.com", "status": "queued", "messageId": "msg_123"}
					]
				},
				"correlationId": "corr_send"
			}`), nil
	case "/emails/send-bulk":
		return s.jsonResponse(http.StatusOK, `{
				"success": true,
				"data": {
					"batchId": "batch_123",
					"status": "queued",
					"templateKey": "welcome-email",
					"templateVersion": 1,
					"senderUsed": "ses",
					"senderVerified": true,
					"totalRecipients": 2,
					"processedCount": 2,
					"successCount": 2,
					"failureCount": 0,
					"suppressedCount": 0,
					"startedAt": "2026-01-01T00:00:00Z",
					"recipients": [
						{"email": "first@example.com", "status": "queued"},
						{"email": "second@example.com", "status": "queued"}
					]
				},
				"correlationId": "corr_bulk"
			}`), nil
	case "/health":
		return s.jsonResponse(http.StatusOK, `{
				"success": true,
				"data": {
					"status": "healthy",
					"timestamp": "2026-01-01T00:00:00Z",
					"version": "test"
				},
				"correlationId": "corr_health"
			}`), nil
	default:
		return s.jsonResponse(http.StatusNotFound, `{"message":"not found"}`), nil
	}
}

func (s *stubTransport) jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}
}

func (s *stubTransport) RequestCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

func (s *stubTransport) RequestAt(index int) (capturedRequest, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.requests) {
		return capturedRequest{}, false
	}
	return s.requests[index], true
}

func pass(name string) {
	fmt.Printf("%s[PASS]%s %s\n", green, reset, name)
	passed++
}

func fail(name, reason string) {
	fmt.Printf("%s[FAIL]%s %s: %s\n", red, reset, name, reason)
	failed++
}

func main() {
	fmt.Println("=== Huefy Go SDK Lab ===")
	fmt.Println()

	mode := strings.ToLower(strings.TrimSpace(os.Getenv("HUEFY_SDK_LAB_MODE")))
	fmt.Printf("Mode: %s\n\n", map[bool]string{true: "live", false: "contract"}[mode == "live"])

	if mode == "live" {
		runLive()
	} else {
		runContract()
	}

	fmt.Println()
	fmt.Println("========================================")
	fmt.Printf("Results: %d passed, %d failed\n", passed, failed)
	fmt.Println("========================================")
	fmt.Println()

	if failed > 0 {
		os.Exit(1)
	}

	fmt.Println("All verifications passed!")
}

func runContract() {
	stub := newStubTransport()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client, err := huefy.NewEmailClient(
		"sdk_lab_test_key",
		huefy.WithBaseURL("https://sdk-lab.invalid"),
		huefy.WithHTTPTransport(stub),
		huefy.WithTimeout(2*time.Second),
	)
	if err != nil {
		fail("Initialization", err.Error())
	} else {
		pass("Initialization")
	}

	if client == nil {
		return
	}

	provider := huefy.ProviderSES
	sendResp, sendErr := client.SendEmail(ctx, &huefy.SendEmailRequest{
		TemplateKey: "  welcome-email  ",
		Data: map[string]any{
			"name": "Jane",
		},
		Recipient: huefy.SendEmailRecipient{
			Email: "  user@example.com  ",
			Type:  " CC ",
			Data: map[string]any{
				"segment": "vip",
			},
		},
		ProviderType: &provider,
	})

	switch {
	case sendErr != nil:
		fail("Single-send contract shaping", sendErr.Error())
	case sendResp == nil || !sendResp.Success:
		fail("Single-send contract shaping", "expected successful stub response")
	default:
		req, ok := stub.RequestAt(0)
		if !ok {
			fail("Single-send contract shaping", "expected captured request")
		} else {
			recipient, ok := req.Body["recipient"].(map[string]any)
			switch {
			case !ok:
				fail("Single-send contract shaping", "recipient was not serialized as an object")
			case req.Method != http.MethodPost:
				fail("Single-send contract shaping", fmt.Sprintf("expected POST, got %s", req.Method))
			case req.Path != "/emails/send":
				fail("Single-send contract shaping", fmt.Sprintf("expected /emails/send, got %s", req.Path))
			case req.Header.Get("X-API-Key") != "sdk_lab_test_key":
				fail("Single-send contract shaping", "missing X-API-Key header")
			case req.Body["templateKey"] != "welcome-email":
				fail("Single-send contract shaping", fmt.Sprintf("expected trimmed templateKey, got %#v", req.Body["templateKey"]))
			case recipient["email"] != "user@example.com":
				fail("Single-send contract shaping", fmt.Sprintf("expected trimmed recipient email, got %#v", recipient["email"]))
			case recipient["type"] != "cc":
				fail("Single-send contract shaping", fmt.Sprintf("expected normalized recipient type, got %#v", recipient["type"]))
			case req.Body["providerType"] != "ses":
				fail("Single-send contract shaping", fmt.Sprintf("expected providerType ses, got %#v", req.Body["providerType"]))
			default:
				pass("Single-send contract shaping")
			}
		}
	}

	bulkResp, bulkErr := client.SendBulkEmails(ctx, &huefy.SendBulkEmailsRequest{
		TemplateKey: "  welcome-email  ",
		Recipients: []huefy.BulkRecipient{
			{
				Email: "  first@example.com  ",
				Type:  " TO ",
				Data: map[string]any{
					"tier": "gold",
				},
			},
			{
				Email: " second@example.com ",
				Type:  " BCC ",
			},
		},
		ProviderType: "ses",
	})

	switch {
	case bulkErr != nil:
		fail("Bulk-send contract shaping", bulkErr.Error())
	case bulkResp == nil || !bulkResp.Success:
		fail("Bulk-send contract shaping", "expected successful stub response")
	default:
		req, ok := stub.RequestAt(1)
		if !ok {
			fail("Bulk-send contract shaping", "expected captured bulk request")
		} else {
			recipients, ok := req.Body["recipients"].([]any)
			if !ok || len(recipients) != 2 {
				fail("Bulk-send contract shaping", "expected two serialized recipients")
			} else {
				first, _ := recipients[0].(map[string]any)
				second, _ := recipients[1].(map[string]any)

				switch {
				case req.Method != http.MethodPost:
					fail("Bulk-send contract shaping", fmt.Sprintf("expected POST, got %s", req.Method))
				case req.Path != "/emails/send-bulk":
					fail("Bulk-send contract shaping", fmt.Sprintf("expected /emails/send-bulk, got %s", req.Path))
				case req.Body["templateKey"] != "welcome-email":
					fail("Bulk-send contract shaping", fmt.Sprintf("expected trimmed templateKey, got %#v", req.Body["templateKey"]))
				case first["email"] != "first@example.com":
					fail("Bulk-send contract shaping", fmt.Sprintf("expected trimmed first email, got %#v", first["email"]))
				case first["type"] != "to":
					fail("Bulk-send contract shaping", fmt.Sprintf("expected normalized first type, got %#v", first["type"]))
				case second["email"] != "second@example.com":
					fail("Bulk-send contract shaping", fmt.Sprintf("expected trimmed second email, got %#v", second["email"]))
				case second["type"] != "bcc":
					fail("Bulk-send contract shaping", fmt.Sprintf("expected normalized second type, got %#v", second["type"]))
				default:
					pass("Bulk-send contract shaping")
				}
			}
		}
	}

	beforeSingle := stub.RequestCount()
	_, sendErr = client.SendEmail(ctx, &huefy.SendEmailRequest{
		TemplateKey: "",
		Data:        nil,
		Recipient:   "not-an-email",
	})
	switch {
	case sendErr == nil:
		fail("Validation rejection for invalid single input", "expected validation error")
	case stub.RequestCount() != beforeSingle:
		fail("Validation rejection for invalid single input", "invalid request reached the transport")
	default:
		pass("Validation rejection for invalid single input")
	}

	beforeBulk := stub.RequestCount()
	_, bulkErr = client.SendBulkEmails(ctx, &huefy.SendBulkEmailsRequest{
		TemplateKey: "welcome-email",
		Recipients: []huefy.BulkRecipient{
			{Email: "bad-email", Type: "reply-to"},
		},
	})
	switch {
	case bulkErr == nil:
		fail("Validation rejection for invalid bulk input", "expected validation error")
	case stub.RequestCount() != beforeBulk:
		fail("Validation rejection for invalid bulk input", "invalid bulk request reached the transport")
	default:
		pass("Validation rejection for invalid bulk input")
	}

	health, healthErr := client.HealthCheck(ctx)
	switch {
	case healthErr != nil:
		fail("SDK health path behavior", healthErr.Error())
	case health == nil || health.Data.Status != "healthy":
		fail("SDK health path behavior", "expected decoded healthy response")
	default:
		req, ok := stub.RequestAt(2)
		if !ok {
			fail("SDK health path behavior", "expected captured health request")
		} else if req.Method != http.MethodGet || req.Path != "/health" {
			fail("SDK health path behavior", fmt.Sprintf("expected GET /health, got %s %s", req.Method, req.Path))
		} else {
			pass("SDK health path behavior")
		}
	}

	client.Close()
	pass("Cleanup")
}

func runLive() {
	live, err := getLiveConfig()
	if err != nil {
		fail("Initialization", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var client *huefy.EmailClient
	client, err = huefy.NewEmailClient(
		live.APIKey,
		huefy.WithBaseURL(live.BaseURL),
		huefy.WithTimeout(10*time.Second),
	)
	if err != nil {
		fail("Initialization", err.Error())
		return
	}
	pass("Initialization")

	health, err := client.HealthCheck(ctx)
	switch {
	case err != nil:
		fail("Health check", err.Error())
	case health == nil || !health.Success || health.Data.Status != "healthy":
		fail("Health check", fmt.Sprintf("unexpected health response: %#v", health))
	default:
		pass("Health check")
	}

	singleReq := &huefy.SendEmailRequest{
		TemplateKey: live.TemplateKey,
		Data: map[string]any{
			"sdkLabMode": "live",
			"sdk":        "go",
			"operation":  "single",
		},
		Recipient: live.Recipient,
	}
	if live.Provider != "" {
		provider := huefy.EmailProvider(live.Provider)
		singleReq.ProviderType = &provider
	}

	sendResp, err := client.SendEmail(ctx, singleReq)
	switch {
	case err != nil:
		fail("Single send", err.Error())
	case sendResp == nil || !sendResp.Success || strings.TrimSpace(sendResp.Data.EmailID) == "":
		fail("Single send", fmt.Sprintf("unexpected send response: %#v", sendResp))
	default:
		pass("Single send")
	}

	bulkReq := &huefy.SendBulkEmailsRequest{
		TemplateKey: live.TemplateKey,
		Recipients: []huefy.BulkRecipient{
			{
				Email: live.Recipient,
				Type:  "to",
				Data: map[string]any{
					"sdkLabMode": "live",
					"sdk":        "go",
					"operation":  "bulk",
				},
			},
		},
	}
	if live.Provider != "" {
		bulkReq.ProviderType = live.Provider
	}

	bulkResp, err := client.SendBulkEmails(ctx, bulkReq)
	switch {
	case err != nil:
		fail("Bulk send", err.Error())
	case bulkResp == nil || !bulkResp.Success || strings.TrimSpace(bulkResp.Data.BatchID) == "" || bulkResp.Data.TotalRecipients < 1:
		fail("Bulk send", fmt.Sprintf("unexpected bulk response: %#v", bulkResp))
	default:
		pass("Bulk send")
	}

	_, err = client.SendEmail(ctx, &huefy.SendEmailRequest{
		TemplateKey: live.TemplateKey,
		Data: map[string]any{
			"sdkLabMode": "live",
			"sdk":        "go",
			"operation":  "invalid-single",
		},
		Recipient: "not-an-email",
	})
	switch {
	case err == nil:
		fail("Invalid single rejection", "expected validation error")
	case !strings.Contains(strings.ToLower(err.Error()), "invalid email"):
		fail("Invalid single rejection", err.Error())
	default:
		pass("Invalid single rejection")
	}

	_, err = client.SendBulkEmails(ctx, &huefy.SendBulkEmailsRequest{
		TemplateKey: live.TemplateKey,
		Recipients:  []huefy.BulkRecipient{},
	})
	switch {
	case err == nil:
		fail("Invalid bulk rejection", "expected validation error")
	case !strings.Contains(strings.ToLower(err.Error()), "at least one email"):
		fail("Invalid bulk rejection", err.Error())
	default:
		pass("Invalid bulk rejection")
	}

	client.Close()
	pass("Cleanup")
}

func getLiveConfig() (liveConfig, error) {
	apiKey, err := requireEnv("HUEFY_SDK_LIVE_API_KEY")
	if err != nil {
		return liveConfig{}, err
	}
	baseURL, err := requireEnv("HUEFY_SDK_LIVE_BASE_URL")
	if err != nil {
		return liveConfig{}, err
	}
	templateKey, err := requireEnv("HUEFY_SDK_LIVE_TEMPLATE_KEY")
	if err != nil {
		return liveConfig{}, err
	}
	recipient, err := requireEnv("HUEFY_SDK_LIVE_RECIPIENT")
	if err != nil {
		return liveConfig{}, err
	}

	provider := strings.ToLower(strings.TrimSpace(os.Getenv("HUEFY_SDK_LIVE_PROVIDER")))
	if provider != "" {
		switch provider {
		case "ses", "sendgrid", "mailgun", "mailchimp":
		default:
			return liveConfig{}, fmt.Errorf("HUEFY_SDK_LIVE_PROVIDER must be one of: ses, sendgrid, mailgun, mailchimp")
		}
	}

	return liveConfig{
		APIKey:      apiKey,
		BaseURL:     baseURL,
		TemplateKey: templateKey,
		Recipient:   recipient,
		Provider:    provider,
	}, nil
}

func requireEnv(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s is required in live mode", name)
	}
	return value, nil
}
