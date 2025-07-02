// Basic example demonstrating how to send a single email using the Huefy Go SDK
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	huefy "github.com/teracrafts/teracrafts-huefy-sdk-go"
)

func main() {
	// Get API key from environment variable
	apiKey := os.Getenv("HUEFY_API_KEY")
	if apiKey == "" {
		log.Fatal("HUEFY_API_KEY environment variable is required")
	}

	// Create a new Huefy client
	client := huefy.NewClient(apiKey)

	// Alternatively, create client with custom configuration
	// client := huefy.NewClient(apiKey,
	//     huefy.WithBaseURL("https://api.huefy.com"),
	//     huefy.WithRetryConfig(&huefy.RetryConfig{
	//         MaxRetries: 3,
	//         BaseDelay:  time.Second,
	//         MaxDelay:   30 * time.Second,
	//         Multiplier: 2.0,
	//     }),
	// )

	// Create email request
	request := &huefy.SendEmailRequest{
		TemplateKey: "welcome-email",
		Data: map[string]interface{}{
			"name":    "John Doe",
			"company": "Acme Corp",
			"product": "Amazing Widget",
		},
		Recipient: "john.doe@example.com",
		// Optional: specify email provider
		// Provider: &huefy.ProviderSendGrid,
	}

	// Send the email
	response, err := client.SendEmail(context.Background(), request)
	if err != nil {
		// Handle different error types
		switch {
		case huefy.IsAuthenticationError(err):
			log.Fatalf("Authentication failed: %v", err)
		case huefy.IsTemplateNotFoundError(err):
			log.Fatalf("Template not found: %v", err)
		case huefy.IsInvalidTemplateDataError(err):
			log.Fatalf("Invalid template data: %v", err)
		case huefy.IsRateLimitError(err):
			if rateLimitErr, ok := err.(*huefy.RateLimitError); ok {
				log.Fatalf("Rate limited. Retry after %d seconds: %v", 
					rateLimitErr.RetryAfter, err)
			}
		case huefy.IsProviderError(err):
			if providerErr, ok := err.(*huefy.ProviderError); ok {
				log.Fatalf("Provider %s error [%s]: %v", 
					providerErr.Provider, providerErr.ProviderCode, err)
			}
		default:
			log.Fatalf("Failed to send email: %v", err)
		}
	}

	// Success!
	fmt.Printf("Email sent successfully!\n")
	fmt.Printf("Message ID: %s\n", response.MessageID)
	fmt.Printf("Status: %s\n", response.Status)
	fmt.Printf("Provider: %s\n", response.Provider)
	fmt.Printf("Timestamp: %s\n", response.Timestamp.Format("2006-01-02 15:04:05"))
}