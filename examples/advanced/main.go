// Advanced example demonstrating custom configuration, error handling, and health checks
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	huefy "github.com/huefy/huefy-sdk/go"
)

func main() {
	// Get API key from environment variable
	apiKey := os.Getenv("HUEFY_API_KEY")
	if apiKey == "" {
		log.Fatal("HUEFY_API_KEY environment variable is required")
	}

	// Create custom HTTP client with timeout
	customHTTPClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Create client with custom configuration
	client := huefy.NewClient(apiKey,
		huefy.WithBaseURL("https://api.huefy.com"),
		huefy.WithHTTPClient(customHTTPClient),
		huefy.WithRetryConfig(&huefy.RetryConfig{
			MaxRetries: 5,                   // Retry up to 5 times
			BaseDelay:  500 * time.Millisecond, // Start with 500ms delay
			MaxDelay:   10 * time.Second,    // Cap at 10 seconds
			Multiplier: 2.5,                 // Increase delay by 2.5x each retry
		}),
	)

	// First, check API health
	fmt.Println("Checking API health...")
	healthResp, err := client.HealthCheck(context.Background())
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}
	fmt.Printf("✅ API is healthy: %s (checked at %s)\n\n", 
		healthResp.Status, healthResp.Timestamp.Format("2006-01-02 15:04:05"))

	// Example 1: Send email with comprehensive error handling
	fmt.Println("=== Example 1: Single Email with Error Handling ===")
	sendEmailWithErrorHandling(client)

	// Example 2: Send email with context timeout
	fmt.Println("\n=== Example 2: Email with Context Timeout ===")
	sendEmailWithTimeout(client)

	// Example 3: Send email with specific provider
	fmt.Println("\n=== Example 3: Email with Specific Provider ===")
	sendEmailWithProvider(client)

	fmt.Println("\n✅ All examples completed successfully!")
}

func sendEmailWithErrorHandling(client *huefy.Client) {
	request := &huefy.SendEmailRequest{
		TemplateKey: "welcome-email",
		Data: map[string]interface{}{
			"name":         "Alice Johnson",
			"company":      "Innovation Labs",
			"signupDate":   time.Now().Format("January 2, 2006"),
			"supportEmail": "support@example.com",
		},
		Recipient: "alice.johnson@innovationlabs.com",
	}

	response, err := client.SendEmail(context.Background(), request)
	if err != nil {
		// Detailed error handling
		switch {
		case huefy.IsAuthenticationError(err):
			log.Printf("❌ Authentication Error: %v", err)
			return
			
		case huefy.IsTemplateNotFoundError(err):
			if templateErr, ok := err.(*huefy.TemplateNotFoundError); ok {
				log.Printf("❌ Template '%s' not found: %v", templateErr.TemplateKey, err)
			}
			return
			
		case huefy.IsInvalidTemplateDataError(err):
			if dataErr, ok := err.(*huefy.InvalidTemplateDataError); ok {
				log.Printf("❌ Invalid template data: %v", err)
				log.Printf("Validation errors: %v", dataErr.ValidationErrors)
			}
			return
			
		case huefy.IsRateLimitError(err):
			if rateLimitErr, ok := err.(*huefy.RateLimitError); ok {
				log.Printf("❌ Rate limited. Retry after %d seconds: %v", 
					rateLimitErr.RetryAfter, err)
			}
			return
			
		case huefy.IsProviderError(err):
			if providerErr, ok := err.(*huefy.ProviderError); ok {
				log.Printf("❌ Provider %s error [%s]: %v", 
					providerErr.Provider, providerErr.ProviderCode, err)
			}
			return
			
		case huefy.IsNetworkError(err):
			log.Printf("❌ Network error (may retry automatically): %v", err)
			return
			
		case huefy.IsValidationError(err):
			log.Printf("❌ Validation error: %v", err)
			return
			
		default:
			log.Printf("❌ Unknown error: %v", err)
			return
		}
	}

	fmt.Printf("✅ Email sent successfully!\n")
	fmt.Printf("   Message ID: %s\n", response.MessageID)
	fmt.Printf("   Provider: %s\n", response.Provider)
	fmt.Printf("   Status: %s\n", response.Status)
}

func sendEmailWithTimeout(client *huefy.Client) {
	// Create context with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	request := &huefy.SendEmailRequest{
		TemplateKey: "password-reset",
		Data: map[string]interface{}{
			"name":      "Bob Wilson",
			"resetLink": "https://example.com/reset?token=abc123",
			"expiresIn": "24 hours",
		},
		Recipient: "bob.wilson@example.com",
	}

	start := time.Now()
	response, err := client.SendEmail(ctx, request)
	duration := time.Since(start)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("❌ Request timed out after %v", duration)
		} else {
			log.Printf("❌ Failed to send email: %v", err)
		}
		return
	}

	fmt.Printf("✅ Email sent in %v\n", duration)
	fmt.Printf("   Message ID: %s\n", response.MessageID)
	fmt.Printf("   Provider: %s\n", response.Provider)
}

func sendEmailWithProvider(client *huefy.Client) {
	// Send email using specific provider
	provider := huefy.ProviderSendGrid
	request := &huefy.SendEmailRequest{
		TemplateKey: "marketing-newsletter",
		Data: map[string]interface{}{
			"name":           "Carol Davis",
			"newsletterTitle": "Weekly Product Updates",
			"featuredProduct": "Smart Home Assistant",
			"discount":        "20%",
			"unsubscribeUrl":  "https://example.com/unsubscribe?token=xyz789",
		},
		Recipient: "carol.davis@example.com",
		Provider:  &provider,
	}

	response, err := client.SendEmail(context.Background(), request)
	if err != nil {
		log.Printf("❌ Failed to send email via %s: %v", provider, err)
		return
	}

	fmt.Printf("✅ Email sent via %s\n", provider)
	fmt.Printf("   Message ID: %s\n", response.MessageID)
	fmt.Printf("   Actual Provider: %s\n", response.Provider)
	fmt.Printf("   Status: %s\n", response.Status)
	fmt.Printf("   Timestamp: %s\n", response.Timestamp.Format("2006-01-02 15:04:05"))
}