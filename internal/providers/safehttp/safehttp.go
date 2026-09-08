// Package safehttp provides the outbound HTTP policy shared by every connector
// provider: an SSRF-resistant client and a bounded JSON decoder.
//
// Only the RSS provider originally had either. The other seven decoded response
// bodies with no size limit, and the two that accept an operator-configured
// base URL (bluesky, ocg) did no destination validation at all - so a connector
// could be pointed at cloud instance metadata or an internal service. Centralising
// the policy means a new provider gets it by default instead of by remembering.
package safehttp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MaxResponseBytes bounds how much of a response body a provider will read.
// Feeds and API pages are small; anything larger is a fault or an attack.
const MaxResponseBytes = 8 << 20 // 8 MiB

const maxRedirects = 5

// IsPublicIP reports whether an address is safe to connect to from a connector.
func IsPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() && !ip.IsUnspecified() && !ip.IsMulticast()
}

// ValidateURL checks a connector destination before any connection is made.
// field names the config key in error messages. When allowUnsafeLocal is true,
// plain http and local addresses are permitted; that exists for tests and local
// fixtures only and must never be enabled from operator config.
func ValidateURL(parsed *url.URL, field string, allowUnsafeLocal bool) error {
	if field == "" {
		field = "url"
	}
	if parsed == nil || parsed.Hostname() == "" {
		return fmt.Errorf("%s must include a hostname", field)
	}
	if !allowUnsafeLocal && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use https", field)
	}
	if allowUnsafeLocal && parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use http or https", field)
	}
	if parsed.User != nil {
		return fmt.Errorf("%s must not include user credentials", field)
	}

	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if !allowUnsafeLocal && (host == "localhost" || strings.HasSuffix(host, ".localhost")) {
		return fmt.Errorf("%s hostname is not allowed", field)
	}
	if ip := net.ParseIP(host); ip != nil && !allowUnsafeLocal && !IsPublicIP(ip) {
		return fmt.Errorf("%s IP address is not public", field)
	}
	return nil
}

// NewClient returns a client that re-checks the destination after DNS
// resolution and again on every redirect, closing the window where a hostname
// passes validation and then resolves to a private address.
func NewClient(timeout time.Duration, field string) *http.Client {
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy:               nil,
		TLSHandshakeTimeout: 10 * time.Second,
		IdleConnTimeout:     30 * time.Second,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			if len(ips) == 0 {
				return nil, fmt.Errorf("%s host resolved to no addresses", field)
			}
			for _, resolved := range ips {
				if !IsPublicIP(resolved.IP) {
					return nil, fmt.Errorf("%s host resolved to blocked address %s", field, resolved.IP)
				}
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		},
	}
	client := &http.Client{Transport: transport, Timeout: timeout}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return fmt.Errorf("too many %s redirects", field)
		}
		return ValidateURL(req.URL, field, false)
	}
	return client
}

// NewUnrestrictedClient returns a plain client with a timeout, for providers
// whose endpoint is a compile-time constant rather than operator config. It
// still pairs with DecodeJSON for the response size bound.
func NewUnrestrictedClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &http.Client{Timeout: timeout}
}

// DecodeJSON decodes a response body into out, reading at most
// MaxResponseBytes. It also rejects trailing garbage so a truncated or
// concatenated body is an error rather than a silently partial decode.
func DecodeJSON(body io.Reader, out any) error {
	decoder := json.NewDecoder(io.LimitReader(body, MaxResponseBytes))
	if err := decoder.Decode(out); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return fmt.Errorf("response exceeded %d bytes or was truncated", MaxResponseBytes)
		}
		return err
	}
	return nil
}

// ReadAll reads at most MaxResponseBytes from body.
func ReadAll(body io.Reader) ([]byte, error) {
	return io.ReadAll(io.LimitReader(body, MaxResponseBytes))
}
