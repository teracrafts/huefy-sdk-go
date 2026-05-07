package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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

type stubServer struct {
	server   *httptest.Server
	mu       sync.Mutex
	requests []capturedRequest
}

func newStubServer() *stubServer {
	stub := &stubServer{}

	stub.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var body map[string]any
		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(rawBody) > 0 {
			if err := json.Unmarshal(rawBody, &body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"message":"invalid json"}`))
				return
			}
		}

		stub.mu.Lock()
		stub.requests = append(stub.requests, capturedRequest{
			Method: r.Method,
			Path:   r.URL.Path,
			Header: r.Header.Clone(),
			Body:   body,
		})
		stub.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/emails/send":
			_, _ = w.Write([]byte(`{
				"success": true,
				"data": {
					"emailId": "email_123",
					"status": "queued",
					"recipients": [
						{"email": "user@example.com", "status": "queued", "messageId": "msg_123"}
					]
				},
				"correlationId": "corr_send"
			}`))
		case "/emails/send-bulk":
			_, _ = w.Write([]byte(`{
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
			}`))
		case "/health":
			_, _ = w.Write([]byte(`{
				"success": true,
				"data": {
					"status": "healthy",
					"timestamp": "2026-01-01T00:00:00Z",
					"version": "test"
				},
				"correlationId": "corr_health"
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"not found"}`))
		}
	}))

	return stub
}

func (s *stubServer) Close() {
	s.server.Close()
}

func (s *stubServer) URL() string {
	return s.server.URL
}

func (s *stubServer) RequestCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

func (s *stubServer) RequestAt(index int) (capturedRequest, bool) {
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

	stub := newStubServer()
	defer stub.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client, err := huefy.NewEmailClient(
		"sdk_lab_test_key",
		huefy.WithBaseURL(stub.URL()),
		huefy.WithTimeout(2*time.Second),
	)
	if err != nil {
		fail("Initialization", err.Error())
	} else {
		pass("Initialization")
	}

	if client != nil {
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
				break
			}

			recipient, ok := req.Body["recipient"].(map[string]any)
			if !ok {
				fail("Single-send contract shaping", "recipient was not serialized as an object")
				break
			}

			switch {
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
				break
			}

			recipients, ok := req.Body["recipients"].([]any)
			if !ok || len(recipients) != 2 {
				fail("Bulk-send contract shaping", "expected two serialized recipients")
				break
			}

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
