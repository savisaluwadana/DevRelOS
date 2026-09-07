package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	outreachdomain "github.com/savisaluwadana/DevRelOS/internal/domain/outreach"
	"github.com/savisaluwadana/DevRelOS/internal/storage"
)

type smtpConfig struct {
	Host       string
	Port       int
	Username   string
	Password   string
	From       string
	FromName   string
	StartTLS   bool
	RequireTLS bool
}

func processAvailableOutreach(ctx context.Context, store *storage.Store) {
	if !envBool("DEVRELOS_SMTP_ENABLED", false) {
		return
	}
	cfg, err := loadSMTPConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "outreach delivery disabled: %v\n", err)
		return
	}
	maxAttempts := envInt("DEVRELOS_OUTREACH_MAX_ATTEMPTS", 5, 1, 10)
	for {
		item, err := store.ClaimNextOutreachDelivery(ctx, maxAttempts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "outreach queue error: %v\n", err)
			return
		}
		if item == nil {
			return
		}
		messageID, sendErr := sendSMTP(ctx, cfg, *item)
		item.MessageID = messageID
		if finishErr := store.FinishOutreachDelivery(ctx, *item, sendErr, maxAttempts); finishErr != nil {
			fmt.Fprintf(os.Stderr, "outreach delivery completion error id=%s: %v\n", item.ID, finishErr)
			return
		}
		if sendErr != nil {
			fmt.Fprintf(os.Stderr, "outreach delivery failed id=%s attempt=%d: %v\n", item.ID, item.AttemptCount, sendErr)
			continue
		}
		fmt.Printf("outreach delivery sent id=%s outreach=%s recipient=%s\n", item.ID, item.OutreachID, item.RecipientEmail)
	}
}

func loadSMTPConfig() (smtpConfig, error) {
	cfg := smtpConfig{
		Host:       strings.TrimSpace(os.Getenv("DEVRELOS_SMTP_HOST")),
		Port:       envInt("DEVRELOS_SMTP_PORT", 587, 1, 65535),
		Username:   strings.TrimSpace(os.Getenv("DEVRELOS_SMTP_USERNAME")),
		Password:   os.Getenv("DEVRELOS_SMTP_PASSWORD"),
		From:       strings.TrimSpace(os.Getenv("DEVRELOS_SMTP_FROM")),
		FromName:   strings.TrimSpace(os.Getenv("DEVRELOS_SMTP_FROM_NAME")),
		StartTLS:   envBool("DEVRELOS_SMTP_STARTTLS", true),
		RequireTLS: envBool("DEVRELOS_SMTP_REQUIRE_TLS", true),
	}
	if cfg.Host == "" {
		return cfg, errors.New("DEVRELOS_SMTP_HOST is required")
	}
	if cfg.From == "" || strings.ContainsAny(cfg.From, "\r\n") || !strings.Contains(cfg.From, "@") {
		return cfg, errors.New("valid DEVRELOS_SMTP_FROM is required")
	}
	if cfg.Username != "" && cfg.Password == "" {
		return cfg, errors.New("DEVRELOS_SMTP_PASSWORD is required when username is set")
	}
	return cfg, nil
}

func sendSMTP(ctx context.Context, cfg smtpConfig, item outreachdomain.Delivery) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	recipient := strings.TrimSpace(item.RecipientEmail)
	if recipient == "" || strings.ContainsAny(recipient, "\r\n") || !strings.Contains(recipient, "@") {
		return "", errors.New("invalid outreach recipient email")
	}

	address := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return "", err
	}
	defer client.Close()

	isTLS := false
	if cfg.StartTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsCfg := &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}
			if err := client.StartTLS(tlsCfg); err != nil {
				return "", err
			}
			isTLS = true
		} else if cfg.RequireTLS {
			return "", errors.New("SMTP server does not advertise STARTTLS")
		}
	} else if cfg.RequireTLS {
		return "", errors.New("TLS is required but STARTTLS is disabled")
	}

	if cfg.Username != "" {
		if cfg.RequireTLS && !isTLS {
			return "", errors.New("refusing SMTP authentication without TLS")
		}
		auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		if err := client.Auth(auth); err != nil {
			return "", err
		}
	}
	if err := client.Mail(cfg.From); err != nil {
		return "", err
	}
	if err := client.Rcpt(recipient); err != nil {
		return "", err
	}
	writer, err := client.Data()
	if err != nil {
		return "", err
	}
	messageID, message := buildSMTPMessage(cfg, item)
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	if err := client.Quit(); err != nil {
		return "", err
	}
	return messageID, nil
}

func buildSMTPMessage(cfg smtpConfig, item outreachdomain.Delivery) (string, []byte) {
	idBytes := make([]byte, 12)
	_, _ = rand.Read(idBytes)
	messageID := fmt.Sprintf("<%s.%s@%s>", strings.ReplaceAll(item.OutreachID, "-", ""), hex.EncodeToString(idBytes), safeMessageIDHost(cfg.Host))
	fromName := sanitizeHeader(cfg.FromName)
	from := cfg.From
	if fromName != "" {
		from = mime.QEncoding.Encode("UTF-8", fromName) + " <" + cfg.From + ">"
	}
	toName := sanitizeHeader(item.RecipientName)
	to := item.RecipientEmail
	if toName != "" {
		to = mime.QEncoding.Encode("UTF-8", toName) + " <" + item.RecipientEmail + ">"
	}
	subject := mime.QEncoding.Encode("UTF-8", sanitizeHeader(item.Subject))
	body := normalizeCRLF(item.Body)
	message := strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
		"Message-ID: " + messageID,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
		"",
	}, "\r\n")
	return messageID, []byte(message)
}

func sanitizeHeader(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}

func normalizeCRLF(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.ReplaceAll(value, "\n", "\r\n")
}

func safeMessageIDHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" || strings.ContainsAny(host, "<>@ \r\n") {
		return "devrelos.local"
	}
	return host
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "true" || value == "1" || value == "yes" || value == "on"
}

func envInt(key string, fallback, minValue, maxValue int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil || value < minValue || value > maxValue {
		return fallback
	}
	return value
}
