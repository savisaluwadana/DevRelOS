package main

import (
	"bufio"
	"context"
	"net"
	"strings"
	"sync/atomic"
	"testing"

	outreachdomain "github.com/savisaluwadana/DevRelOS/internal/domain/outreach"
)

// fakeSMTP speaks just enough SMTP to accept one message. quitBehaviour decides
// what happens after the message has been accepted.
type quitBehaviour int

const (
	quitOK             quitBehaviour = iota
	quitError                        // server replies 421 to QUIT
	quitDropConnection               // server closes the socket without replying
)

func startFakeSMTP(t *testing.T, behaviour quitBehaviour) (host string, port int, accepted *atomic.Int32) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	accepted = &atomic.Int32{}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				reader := bufio.NewReader(c)
				write := func(s string) { _, _ = c.Write([]byte(s + "\r\n")) }
				write("220 fake.test ESMTP")
				inData := false
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					trimmed := strings.TrimRight(line, "\r\n")

					if inData {
						if trimmed == "." {
							inData = false
							// This response is the acceptance of the message.
							accepted.Add(1)
							write("250 2.0.0 Ok: queued as FAKE1")
						}
						continue
					}

					upper := strings.ToUpper(trimmed)
					switch {
					case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
						write("250-fake.test")
						write("250 SIZE 10240000")
					case strings.HasPrefix(upper, "MAIL FROM"), strings.HasPrefix(upper, "RCPT TO"):
						write("250 2.1.0 Ok")
					case upper == "DATA":
						inData = true
						write("354 End data with <CR><LF>.<CR><LF>")
					case upper == "QUIT":
						switch behaviour {
						case quitOK:
							write("221 2.0.0 Bye")
						case quitError:
							write("421 4.7.0 Try again later")
						case quitDropConnection:
							// Say nothing and hang up.
						}
						return
					default:
						write("250 2.0.0 Ok")
					}
				}
			}(conn)
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)
	return "127.0.0.1", addr.Port, accepted
}

func testDelivery() outreachdomain.Delivery {
	return outreachdomain.Delivery{
		ID:             "delivery-1",
		OutreachID:     "outreach-1",
		RecipientName:  "Organizer",
		RecipientEmail: "organizer@example.invalid",
		Subject:        "Talk proposal",
		Body:           "Hello there.",
	}
}

func testConfig(host string, port int) smtpConfig {
	return smtpConfig{
		Host: host, Port: port,
		From: "devrel@example.invalid", FromName: "DevRel",
		StartTLS: false, RequireTLS: false,
	}
}

// A QUIT failure after the server accepted the message must not be reported as
// a send failure: the delivery would be retried and the same human-approved
// email would reach the organiser twice.
func TestSendSMTPTreatsQuitFailureAsDelivered(t *testing.T) {
	for _, tc := range []struct {
		name      string
		behaviour quitBehaviour
	}{
		{"server rejects QUIT", quitError},
		{"server drops the connection", quitDropConnection},
	} {
		t.Run(tc.name, func(t *testing.T) {
			host, port, accepted := startFakeSMTP(t, tc.behaviour)
			messageID, err := sendSMTP(context.Background(), testConfig(host, port), testDelivery())
			if err != nil {
				t.Fatalf("delivery reported failure after the message was accepted: %v", err)
			}
			if messageID == "" {
				t.Fatal("expected a message id for an accepted message")
			}
			if got := accepted.Load(); got != 1 {
				t.Fatalf("server accepted %d messages, want 1", got)
			}
		})
	}
}

func TestSendSMTPSucceedsOnCleanQuit(t *testing.T) {
	host, port, accepted := startFakeSMTP(t, quitOK)
	messageID, err := sendSMTP(context.Background(), testConfig(host, port), testDelivery())
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if messageID == "" || accepted.Load() != 1 {
		t.Fatalf("messageID=%q accepted=%d", messageID, accepted.Load())
	}
}

func TestSendSMTPRejectsBadRecipients(t *testing.T) {
	host, port, accepted := startFakeSMTP(t, quitOK)
	cfg := testConfig(host, port)
	for _, recipient := range []string{"", "   ", "no-at-sign", "a@b\r\nBcc: victim@example.invalid"} {
		item := testDelivery()
		item.RecipientEmail = recipient
		if _, err := sendSMTP(context.Background(), cfg, item); err == nil {
			t.Errorf("recipient %q was accepted", recipient)
		}
	}
	if got := accepted.Load(); got != 0 {
		t.Fatalf("server accepted %d messages for invalid recipients, want 0", got)
	}
}

func TestSendSMTPRefusesWhenTLSRequiredButUnavailable(t *testing.T) {
	host, port, accepted := startFakeSMTP(t, quitOK)
	cfg := testConfig(host, port)
	cfg.StartTLS = true
	cfg.RequireTLS = true
	// The fake server never advertises STARTTLS.
	if _, err := sendSMTP(context.Background(), cfg, testDelivery()); err == nil {
		t.Fatal("sent without TLS while TLS was required")
	}
	if got := accepted.Load(); got != 0 {
		t.Fatalf("a message was delivered without required TLS (%d accepted)", got)
	}
}
