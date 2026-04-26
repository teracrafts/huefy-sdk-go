package validators

import (
	"fmt"
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

const (
	MaxEmailLength    = 254
	MaxTemplateKeyLen = 100
	MaxBulkEmails     = 1000
)

func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("recipient email is required")
	}
	trimmed := strings.TrimSpace(email)
	if len(trimmed) > MaxEmailLength {
		return fmt.Errorf("email exceeds maximum length of %d characters", MaxEmailLength)
	}
	if !emailRegex.MatchString(trimmed) {
		return fmt.Errorf("invalid email address: %s", trimmed)
	}
	return nil
}

func ValidateTemplateKey(key string) error {
	if key == "" {
		return fmt.Errorf("template key is required")
	}
	trimmed := strings.TrimSpace(key)
	if len(trimmed) == 0 {
		return fmt.Errorf("template key cannot be empty")
	}
	if len(trimmed) > MaxTemplateKeyLen {
		return fmt.Errorf("template key exceeds maximum length of %d characters", MaxTemplateKeyLen)
	}
	return nil
}

func ValidateEmailData(data map[string]any) error {
	if data == nil {
		return fmt.Errorf("template data is required")
	}
	return nil
}

func ValidateBulkCount(count int) error {
	if count <= 0 {
		return fmt.Errorf("at least one email is required")
	}
	if count > MaxBulkEmails {
		return fmt.Errorf("maximum of %d emails per bulk request", MaxBulkEmails)
	}
	return nil
}

func ValidateSendEmailInput(templateKey string, data map[string]any, recipient string) []error {
	var errs []error
	if err := ValidateTemplateKey(templateKey); err != nil {
		errs = append(errs, err)
	}
	if err := ValidateEmailData(data); err != nil {
		errs = append(errs, err)
	}
	if err := ValidateEmail(recipient); err != nil {
		errs = append(errs, err)
	}
	return errs
}
