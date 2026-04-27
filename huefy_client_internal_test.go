package huefy

import "testing"

func TestNormalizeSendEmailRecipientMapStringString(t *testing.T) {
	normalized, err := normalizeSendEmailRecipient(map[string]string{
		"email": " user@example.com ",
		"type":  " CC ",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	recipientMap, ok := normalized.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", normalized)
	}

	if recipientMap["email"] != "user@example.com" {
		t.Fatalf("expected trimmed email, got %#v", recipientMap["email"])
	}

	if recipientMap["type"] != "cc" {
		t.Fatalf("expected normalized type, got %#v", recipientMap["type"])
	}
}
