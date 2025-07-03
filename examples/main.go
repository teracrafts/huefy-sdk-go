package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/teracrafts/huefy-sdk-go"
)

// Example user data structure
type User struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Company string `json:"company,omitempty"`
}

// WelcomeEmailData represents the data for welcome emails
type WelcomeEmailData struct {
	Name           string `json:"name"`
	Company        string `json:"company"`
	ActivationLink string `json:"activation_link"`
	SupportEmail   string `json:"support_email"`
}

// NewsletterData represents the data for newsletter emails
type NewsletterData struct {
	SubscriberName   string `json:"subscriber_name"`
	NewsletterTitle  string `json:"newsletter_title"`
	UnsubscribeLink  string `json:"unsubscribe_link"`
	Articles         []Article `json:"articles"`
}

type Article struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	URL     string `json:"url"`
}

func main() {
	// Get API key from environment variable or use default
	apiKey := os.Getenv("HUEFY_API_KEY")
	if apiKey == "" {
		apiKey = "your-huefy-api-key"
		fmt.Println("Warning: Using default API key. Set HUEFY_API_KEY environment variable.")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Example 1: Basic client creation and single email
	fmt.Println("=== Basic Email Sending ===")
	
	config := &huefy.Config{
		BaseURL: "https://api.huefy.com",
		Timeout: 30 * time.Second,
		RetryConfig: &huefy.RetryConfig{
			MaxRetries: 3,
			BaseDelay:  1 * time.Second,
			MaxDelay:   10 * time.Second,
		},
	}

	client := huefy.NewClient(apiKey, config)

	// Send a welcome email
	welcomeData := WelcomeEmailData{
		Name:           "John Doe",
		Company:        "Acme Corporation",
		ActivationLink: "https://app.example.com/activate/abc123",
		SupportEmail:   "support@example.com",
	}

	emailRequest := &huefy.SendEmailRequest{
		TemplateKey: "welcome-email",
		Recipient:   "john@example.com",
		Data:        welcomeData,
		Provider:    huefy.ProviderSendGrid,
	}

	response, err := client.SendEmail(ctx, emailRequest)
	if err != nil {
		log.Printf("❌ Failed to send email: %v", err)
	} else {
		fmt.Printf("✅ Email sent successfully!\n")
		fmt.Printf("Message ID: %s\n", response.MessageID)
		fmt.Printf("Provider: %s\n", response.Provider)
		fmt.Printf("Status: %s\n\n", response.Status)
	}

	// Example 2: Bulk email sending
	fmt.Println("=== Bulk Email Sending ===")
	
	users := []User{
		{Name: "Alice Johnson", Email: "alice@example.com", Company: "Tech Corp"},
		{Name: "Bob Smith", Email: "bob@example.com", Company: "Startup Inc"},
		{Name: "Carol Davis", Email: "carol@example.com"},
	}

	var bulkRequests []*huefy.SendEmailRequest
	
	for _, user := range users {
		company := user.Company
		if company == "" {
			company = "Your Organization"
		}

		userData := WelcomeEmailData{
			Name:           user.Name,
			Company:        company,
			ActivationLink: fmt.Sprintf("https://app.example.com/activate/%d", time.Now().UnixNano()),
			SupportEmail:   "support@example.com",
		}

		bulkRequests = append(bulkRequests, &huefy.SendEmailRequest{
			TemplateKey: "welcome-email",
			Recipient:   user.Email,
			Data:        userData,
			Provider:    huefy.ProviderSendGrid,
		})
	}

	bulkResponse, err := client.SendBulkEmails(ctx, bulkRequests)
	if err != nil {
		log.Printf("❌ Failed to send bulk emails: %v", err)
	} else {
		fmt.Printf("✅ Bulk email operation completed!\n")
		fmt.Printf("Total emails: %d\n", bulkResponse.TotalEmails)
		fmt.Printf("Successful: %d\n", bulkResponse.SuccessfulEmails)
		fmt.Printf("Failed: %d\n", bulkResponse.FailedEmails)
		fmt.Printf("Success rate: %.1f%%\n\n", bulkResponse.SuccessRate)

		if bulkResponse.FailedEmails > 0 {
			fmt.Println("❌ Failed emails:")
			for _, result := range bulkResponse.Results {
				if result.Status != "sent" {
					fmt.Printf("  - %s: %s\n", result.Recipient, result.Error)
				}
			}
		}
	}

	// Example 3: Newsletter sending
	fmt.Println("=== Newsletter Sending ===")
	
	subscribers := []User{
		{Name: "Newsletter Subscriber 1", Email: "subscriber1@example.com"},
		{Name: "Newsletter Subscriber 2", Email: "subscriber2@example.com"},
		{Name: "Newsletter Subscriber 3", Email: "subscriber3@example.com"},
	}

	newsletterData := NewsletterData{
		NewsletterTitle: "Weekly Tech Updates",
		UnsubscribeLink: "https://app.example.com/unsubscribe",
		Articles: []Article{
			{
				Title:   "New Features Released",
				Summary: "Discover the latest features in our platform",
				URL:     "https://blog.example.com/new-features",
			},
			{
				Title:   "Performance Improvements",
				Summary: "Learn about our latest performance optimizations",
				URL:     "https://blog.example.com/performance",
			},
		},
	}

	var newsletterRequests []*huefy.SendEmailRequest
	
	for _, subscriber := range subscribers {
		data := newsletterData
		data.SubscriberName = subscriber.Name

		newsletterRequests = append(newsletterRequests, &huefy.SendEmailRequest{
			TemplateKey: "newsletter",
			Recipient:   subscriber.Email,
			Data:        data,
			Provider:    huefy.ProviderMailgun,
		})
	}

	newsletterResponse, err := client.SendBulkEmails(ctx, newsletterRequests)
	if err != nil {
		log.Printf("❌ Failed to send newsletter: %v", err)
	} else {
		fmt.Printf("✅ Newsletter sent to %d/%d subscribers\n\n", 
			newsletterResponse.SuccessfulEmails, newsletterResponse.TotalEmails)
	}

	// Example 4: Health check
	fmt.Println("=== API Health Check ===")
	
	healthResponse, err := client.HealthCheck(ctx)
	if err != nil {
		log.Printf("❌ Health check failed: %v", err)
	} else {
		switch healthResponse.Status {
		case "healthy":
			fmt.Println("✅ API is healthy")
		case "degraded":
			fmt.Println("⚠️  API is degraded")
		default:
			fmt.Println("❌ API is unhealthy")
		}
		
		fmt.Printf("Version: %s\n", healthResponse.Version)
		fmt.Printf("Uptime: %d seconds\n", healthResponse.Uptime)
		fmt.Printf("Timestamp: %s\n\n", healthResponse.Timestamp)
	}

	// Example 5: Using different email providers
	fmt.Println("=== Multiple Email Providers ===")
	
	providers := []huefy.EmailProvider{
		huefy.ProviderSendGrid,
		huefy.ProviderMailgun,
		huefy.ProviderSES,
		huefy.ProviderMailchimp,
	}

	testData := map[string]interface{}{
		"message": "Testing provider functionality",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	for _, provider := range providers {
		providerRequest := &huefy.SendEmailRequest{
			TemplateKey: "test-template",
			Recipient:   "test@example.com",
			Data:        testData,
			Provider:    provider,
		}

		providerResponse, err := client.SendEmail(ctx, providerRequest)
		if err != nil {
			fmt.Printf("❌ %s: %v\n", provider, err)
		} else {
			fmt.Printf("✅ %s: %s\n", provider, providerResponse.MessageID)
		}
	}

	// Example 6: Error handling demonstration
	fmt.Println("\n=== Error Handling Examples ===")
	
	// This will fail due to invalid email
	invalidRequest := &huefy.SendEmailRequest{
		TemplateKey: "test-template",
		Recipient:   "invalid-email-address",
		Data:        map[string]interface{}{"message": "Test"},
		Provider:    huefy.ProviderSES,
	}

	_, err = client.SendEmail(ctx, invalidRequest)
	if err != nil {
		switch e := err.(type) {
		case *huefy.ValidationError:
			fmt.Printf("Validation Error: %s (Field: %s)\n", e.Message, e.Field)
		case *huefy.AuthenticationError:
			fmt.Printf("Authentication Error: %s\n", e.Message)
		case *huefy.NetworkError:
			fmt.Printf("Network Error: %s\n", e.Message)
		case *huefy.TimeoutError:
			fmt.Printf("Timeout Error: %s\n", e.Message)
		default:
			fmt.Printf("Unknown Error: %v\n", err)
		}
	}

	// Example 7: Custom timeout context
	fmt.Println("\n=== Custom Timeout Example ===")
	
	// Create a context with a very short timeout to demonstrate timeout handling
	shortCtx, shortCancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer shortCancel()

	timeoutRequest := &huefy.SendEmailRequest{
		TemplateKey: "test-template",
		Recipient:   "timeout-test@example.com",
		Data:        map[string]interface{}{"message": "This will timeout"},
		Provider:    huefy.ProviderSES,
	}

	_, err = client.SendEmail(shortCtx, timeoutRequest)
	if err != nil {
		if timeoutErr, ok := err.(*huefy.TimeoutError); ok {
			fmt.Printf("Expected timeout occurred: %s\n", timeoutErr.Message)
		} else {
			fmt.Printf("Unexpected error: %v\n", err)
		}
	}

	fmt.Println("\n=== Go example completed ===")
}

// Helper function to demonstrate error handling patterns
func handleEmailError(err error, operation string) {
	if err == nil {
		return
	}

	fmt.Printf("❌ %s failed: ", operation)

	switch e := err.(type) {
	case *huefy.ValidationError:
		fmt.Printf("Validation error - %s", e.Message)
		if e.Field != "" {
			fmt.Printf(" (Field: %s)", e.Field)
		}
	case *huefy.AuthenticationError:
		fmt.Printf("Authentication error - %s", e.Message)
		fmt.Println("\n💡 Please check your API key configuration.")
	case *huefy.NetworkError:
		fmt.Printf("Network error - %s", e.Message)
		fmt.Println("\n💡 Please check your network connection.")
	case *huefy.TimeoutError:
		fmt.Printf("Timeout error - %s", e.Message)
		fmt.Println("\n💡 Consider increasing timeout settings.")
	default:
		fmt.Printf("Unknown error - %v", err)
	}
	fmt.Println()
}

// Example of using the client with custom configuration
func createCustomClient(apiKey string) *huefy.Client {
	config := &huefy.Config{
		BaseURL: "https://api.huefy.com",
		Timeout: 45 * time.Second,
		RetryConfig: &huefy.RetryConfig{
			MaxRetries: 5,
			BaseDelay:  2 * time.Second,
			MaxDelay:   60 * time.Second,
		},
		UserAgent: "MyApp/1.0 (contact@myapp.com)",
	}

	return huefy.NewClient(apiKey, config)
}