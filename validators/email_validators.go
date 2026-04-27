package validators

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/teracrafts/huefy-go/models"
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
var validRecipientTypes = map[string]struct{}{
	"to":  {},
	"cc":  {},
	"bcc": {},
}

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

func ValidateRecipient(recipient any) error {
	switch value := recipient.(type) {
	case string:
		return ValidateEmail(value)
	case *string:
		if value == nil {
			return fmt.Errorf("recipient email is required")
		}
		return ValidateEmail(*value)
	case map[string]any:
		email, _ := value["email"].(string)
		if err := ValidateEmail(email); err != nil {
			return err
		}
		if err := validateRecipientType(value["type"]); err != nil {
			return err
		}
		if err := validateRecipientData(value["data"]); err != nil {
			return err
		}
		return nil
	case map[string]string:
		if err := ValidateEmail(value["email"]); err != nil {
			return err
		}
		return validateRecipientType(value["type"])
	case models.SendEmailRecipient:
		if err := ValidateEmail(value.Email); err != nil {
			return err
		}
		if err := validateRecipientType(value.Type); err != nil {
			return err
		}
		return validateRecipientData(value.Data)
	case *models.SendEmailRecipient:
		if value == nil {
			return fmt.Errorf("recipient email is required")
		}
		if err := ValidateEmail(value.Email); err != nil {
			return err
		}
		if err := validateRecipientType(value.Type); err != nil {
			return err
		}
		return validateRecipientData(value.Data)
	}

	return fmt.Errorf("recipient must be a string or recipient object")
}

func ValidateBulkRecipient(recipient models.BulkRecipient) error {
	if err := ValidateEmail(recipient.Email); err != nil {
		return err
	}

	if err := validateRecipientType(recipient.Type); err != nil {
		return err
	}

	return validateRecipientData(recipient.Data)
}

func validateRecipientType(value any) error {
	switch recipientType := value.(type) {
	case nil:
		return nil
	case string:
		normalized := strings.ToLower(strings.TrimSpace(recipientType))
		if normalized == "" {
			return nil
		}
		if _, ok := validRecipientTypes[normalized]; ok {
			return nil
		}
		return fmt.Errorf("recipient type must be one of: to, cc, bcc")
	default:
		return fmt.Errorf("recipient type must be one of: to, cc, bcc")
	}
}

func validateRecipientData(value any) error {
	switch value.(type) {
	case nil, map[string]any, map[string]string:
		return nil
	default:
		return fmt.Errorf("recipient data must be an object")
	}
}

func ValidateSendEmailInput(templateKey string, data map[string]any, recipient any) []error {
	var errs []error
	if err := ValidateTemplateKey(templateKey); err != nil {
		errs = append(errs, err)
	}
	if err := ValidateEmailData(data); err != nil {
		errs = append(errs, err)
	}
	if err := ValidateRecipient(recipient); err != nil {
		errs = append(errs, err)
	}
	return errs
}
