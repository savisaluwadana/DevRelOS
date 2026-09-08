package main

import (
	"strings"
	"testing"

	outreachdomain "github.com/savisaluwadana/DevRelOS/internal/domain/outreach"
)

func TestBuildSMTPMessageSanitizesHeaders(t *testing.T) {
	cfg := smtpConfig{Host: "smtp.example.com", From: "sender@example.com", FromName: "DevRel\nOS"}
	item := outreachdomain.Delivery{
		OutreachID:     "11111111-1111-1111-1111-111111111111",
		RecipientName:  "Alice\r\nBcc: attacker@example.com",
		RecipientEmail: "alice@example.com",
		Subject:        "Hello\r\nBcc: attacker@example.com",
		Body:           "line one\nline two",
	}
	messageID, message := buildSMTPMessage(cfg, item)
	text := string(message)
	if !strings.HasPrefix(messageID, "<11111111111111111111111111111111.") {
		t.Fatalf("unexpected message id: %s", messageID)
	}
	if strings.Contains(text, "\r\nBcc: attacker@example.com\r\n") {
		t.Fatalf("header injection survived: %s", text)
	}
	if !strings.Contains(text, "line one\r\nline two") {
		t.Fatalf("body was not normalized to CRLF: %q", text)
	}
	if !strings.Contains(text, "Message-ID: "+messageID) {
		t.Fatal("message id header missing")
	}
}

func TestLoadSMTPConfigRequiresTLSCompatibleSettings(t *testing.T) {
	t.Setenv("DEVRELOS_SMTP_HOST", "smtp.example.com")
	t.Setenv("DEVRELOS_SMTP_FROM", "sender@example.com")
	t.Setenv("DEVRELOS_SMTP_USERNAME", "user")
	t.Setenv("DEVRELOS_SMTP_PASSWORD", "password")
	t.Setenv("DEVRELOS_SMTP_REQUIRE_TLS", "true")
	t.Setenv("DEVRELOS_SMTP_STARTTLS", "true")
	cfg, err := loadSMTPConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Host != "smtp.example.com" || cfg.From != "sender@example.com" || !cfg.RequireTLS || !cfg.StartTLS {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadSMTPConfigRejectsAuthWithoutPassword(t *testing.T) {
	t.Setenv("DEVRELOS_SMTP_HOST", "smtp.example.com")
	t.Setenv("DEVRELOS_SMTP_FROM", "sender@example.com")
	t.Setenv("DEVRELOS_SMTP_USERNAME", "user")
	t.Setenv("DEVRELOS_SMTP_PASSWORD", "")
	if _, err := loadSMTPConfig(); err == nil {
		t.Fatal("expected missing password error")
	}
}
