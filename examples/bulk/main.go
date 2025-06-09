// Bulk email example demonstrating how to send multiple emails using the Huefy Go SDK
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	huefy "github.com/huefy/huefy-sdk/go"
)

func main() {
	// Get API key from environment variable
	apiKey := os.Getenv("HUEFY_API_KEY")
	if apiKey == "" {
		log.Fatal("HUEFY_API_KEY environment variable is required")
	}

	// Create a new Huefy client
	client := huefy.NewClient(apiKey)

	// Prepare multiple emails
	emails := []huefy.SendEmailRequest{
		{
			TemplateKey: "welcome-email",
			Data: map[string]interface{}{
				"name":    "John Doe",
				"company": "Acme Corp",
			},
			Recipient: "john.doe@example.com",
		},
		{
			TemplateKey: "welcome-email",
			Data: map[string]interface{}{
				"name":    "Jane Smith",
				"company": "Tech Solutions Inc",
			},
			Recipient: "jane.smith@techsolutions.com",
			Provider:  &huefy.ProviderSendGrid, // Use different provider for this email
		},
		{
			TemplateKey: "newsletter",
			Data: map[string]interface{}{
				"name":        "Bob Johnson",
				"newsletter":  "Weekly Tech Update",
				"unsubscribe": "https://example.com/unsubscribe?token=abc123",
			},
			Recipient: "bob.johnson@startup.io",
		},
	}

	// Send bulk emails
	response, err := client.SendBulkEmails(context.Background(), emails)
	if err != nil {
		log.Fatalf("Failed to send bulk emails: %v", err)
	}

	// Process results
	fmt.Printf("Bulk email operation completed. Results:\n\n")
	
	successCount := 0
	failureCount := 0

	for i, result := range response.Results {
		fmt.Printf("Email %d (%s):\n", i+1, emails[i].Recipient)
		
		if result.Success {
			successCount++
			fmt.Printf("  ✅ SUCCESS\n")
			fmt.Printf("     Message ID: %s\n", result.Result.MessageID)
			fmt.Printf("     Status: %s\n", result.Result.Status)
			fmt.Printf("     Provider: %s\n", result.Result.Provider)
			fmt.Printf("     Timestamp: %s\n", result.Result.Timestamp.Format("2006-01-02 15:04:05"))
		} else {
			failureCount++
			fmt.Printf("  ❌ FAILED\n")
			fmt.Printf("     Error: %s\n", result.Error.Error.Message)
			fmt.Printf("     Code: %s\n", result.Error.Error.Code)
		}
		fmt.Println()
	}

	// Summary
	fmt.Printf("Summary:\n")
	fmt.Printf("  Total emails: %d\n", len(emails))
	fmt.Printf("  Successful: %d\n", successCount)
	fmt.Printf("  Failed: %d\n", failureCount)
	fmt.Printf("  Success rate: %.1f%%\n", float64(successCount)/float64(len(emails))*100)
}