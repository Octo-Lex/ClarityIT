package email

import (
	"testing"
)

func TestDevModeDoesNotSend(t *testing.T) {
	svc := NewService("dev", "", 0, "", "", "", "")
	if err := svc.Send("user@example.com", "Test", "body"); err != nil {
		t.Errorf("dev mode send failed: %v", err)
	}
}

func TestDisabledModeDoesNothing(t *testing.T) {
	svc := NewService("disabled", "", 0, "", "", "", "")
	if err := svc.Send("user@example.com", "Test", "body"); err != nil {
		t.Errorf("disabled mode send failed: %v", err)
	}
}

func TestSMTPModeRequiresConfig(t *testing.T) {
	svc := NewService("smtp", "", 0, "", "", "", "")
	if err := svc.Validate(); err == nil {
		t.Error("expected validation error for missing SMTP config")
	}
}

func TestSMTPModeValidWithConfig(t *testing.T) {
	svc := NewService("smtp", "smtp.example.com", 587, "user", "pass", "noreply@example.com", "starttls")
	if err := svc.Validate(); err != nil {
		t.Errorf("valid SMTP config should pass: %v", err)
	}
}

func TestDevModeValidWithoutSMTP(t *testing.T) {
	svc := NewService("dev", "", 0, "", "", "", "")
	if err := svc.Validate(); err != nil {
		t.Errorf("dev mode should not require SMTP config: %v", err)
	}
}

func TestPreviewURLReturnsEmptyInSMTP(t *testing.T) {
	svc := NewService("smtp", "smtp.example.com", 587, "user", "pass", "noreply@example.com", "starttls")
	if svc.PreviewURL("hash123") != "" {
		t.Error("preview URL should be empty in SMTP mode")
	}
}

func TestPreviewURLReturnsValueInDev(t *testing.T) {
	svc := NewService("dev", "", 0, "", "", "", "")
	url := svc.PreviewURL("hash123")
	if url == "" {
		t.Error("preview URL should not be empty in dev mode")
	}
}

func TestIsConfigured(t *testing.T) {
	if !NewService("dev", "", 0, "", "", "", "").IsConfigured() {
		t.Error("dev mode should be configured")
	}
	if !NewService("smtp", "smtp.example.com", 587, "user", "pass", "noreply@example.com", "starttls").IsConfigured() {
		t.Error("smtp mode should be configured")
	}
	if NewService("disabled", "", 0, "", "", "", "").IsConfigured() {
		t.Error("disabled mode should not be configured")
	}
}

func TestUnknownModeReturnsError(t *testing.T) {
	svc := NewService("unknown", "", 0, "", "", "", "")
	if err := svc.Send("user@example.com", "Test", "body"); err == nil {
		t.Error("expected error for unknown mode")
	}
}
