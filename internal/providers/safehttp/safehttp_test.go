package safehttp

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestIsPublicIPRejectsInternalRanges(t *testing.T) {
	blocked := []string{
		"127.0.0.1",       // loopback
		"::1",             // loopback v6
		"10.0.0.5",        // private
		"172.16.0.5",      // private
		"192.168.1.10",    // private
		"169.254.169.254", // link-local: cloud instance metadata
		"fe80::1",         // link-local v6
		"0.0.0.0",         // unspecified
		"224.0.0.1",       // multicast
	}
	for _, raw := range blocked {
		ip := net.ParseIP(raw)
		if ip == nil {
			t.Fatalf("test fixture %q is not a valid IP", raw)
		}
		if IsPublicIP(ip) {
			t.Errorf("IsPublicIP(%s) = true, want false", raw)
		}
	}
	if IsPublicIP(nil) {
		t.Error("IsPublicIP(nil) = true, want false")
	}

	for _, raw := range []string{"93.184.216.34", "8.8.8.8", "2606:2800:220:1::1"} {
		if !IsPublicIP(net.ParseIP(raw)) {
			t.Errorf("IsPublicIP(%s) = false, want true", raw)
		}
	}
}

func TestValidateURLRejectsUnsafeDestinations(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"http://example.com/feed", "must use https"},
		{"https://user:pass@example.com/feed", "must not include user credentials"},
		{"https://localhost/feed", "hostname is not allowed"},
		{"https://api.localhost/feed", "hostname is not allowed"},
		{"https://127.0.0.1/feed", "IP address is not public"},
		{"https://169.254.169.254/latest/meta-data/", "IP address is not public"},
		{"https://10.0.0.1/feed", "IP address is not public"},
		{"https:///feed", "must include a hostname"},
	}
	for _, tc := range cases {
		parsed, err := url.Parse(tc.raw)
		if err != nil {
			t.Fatalf("fixture %q: %v", tc.raw, err)
		}
		err = ValidateURL(parsed, "base_url", false)
		if err == nil {
			t.Errorf("ValidateURL(%q) = nil, want an error", tc.raw)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("ValidateURL(%q) = %q, want it to mention %q", tc.raw, err, tc.want)
		}
		if !strings.Contains(err.Error(), "base_url") {
			t.Errorf("ValidateURL(%q) = %q, want it to name the config field", tc.raw, err)
		}
	}
}

func TestValidateURLAcceptsPublicHTTPS(t *testing.T) {
	parsed, _ := url.Parse("https://public.api.bsky.app/xrpc/app.bsky.feed.searchPosts")
	if err := ValidateURL(parsed, "base_url", false); err != nil {
		t.Fatalf("rejected a legitimate destination: %v", err)
	}
	// A trailing-dot FQDN must not sneak past the localhost check.
	parsed, _ = url.Parse("https://localhost./feed")
	if err := ValidateURL(parsed, "base_url", false); err == nil {
		t.Fatal("localhost with a trailing dot was allowed")
	}
}

func TestValidateURLUnsafeLocalPermitsFixtures(t *testing.T) {
	for _, raw := range []string{"http://127.0.0.1:8080/feed", "http://localhost:3000/feed"} {
		parsed, _ := url.Parse(raw)
		if err := ValidateURL(parsed, "feed_url", true); err != nil {
			t.Errorf("allowUnsafeLocal should permit %q, got %v", raw, err)
		}
	}
	// Even relaxed, a non-HTTP scheme stays rejected.
	parsed, _ := url.Parse("file:///etc/passwd")
	if err := ValidateURL(parsed, "feed_url", true); err == nil {
		t.Error("file:// was allowed even with allowUnsafeLocal")
	}
}

func TestDecodeJSONReadsValidBody(t *testing.T) {
	var out struct{ Name string }
	if err := DecodeJSON(strings.NewReader(`{"Name":"devrelos"}`), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Name != "devrelos" {
		t.Fatalf("Name = %q", out.Name)
	}
}

func TestDecodeJSONBoundsOversizedBody(t *testing.T) {
	// A body larger than the cap must error rather than be read into memory.
	// The payload is a valid JSON array that never terminates within the limit.
	var out []int
	body := strings.NewReader("[" + strings.Repeat("1,", MaxResponseBytes))
	err := DecodeJSON(body, &out)
	if err == nil {
		t.Fatal("oversized body decoded without error")
	}
}

func TestNewClientRedirectGuard(t *testing.T) {
	client := NewClient(0, "base_url")
	if client.CheckRedirect == nil {
		t.Fatal("safe client has no redirect guard")
	}

	// A destination can pass validation and then redirect inward; the guard
	// re-validates every hop, so metadata and private addresses stay blocked.
	inward, _ := url.Parse("http://169.254.169.254/latest/meta-data/")
	if err := client.CheckRedirect(&http.Request{URL: inward}, nil); err == nil {
		t.Fatal("redirect to link-local metadata was allowed")
	}

	onward, _ := url.Parse("https://example.com/feed")
	if err := client.CheckRedirect(&http.Request{URL: onward}, nil); err != nil {
		t.Fatalf("redirect to a public https host was blocked: %v", err)
	}

	// Redirect chains are bounded.
	via := make([]*http.Request, maxRedirects)
	if err := client.CheckRedirect(&http.Request{URL: onward}, via); err == nil {
		t.Fatal("unbounded redirect chain was allowed")
	}
}

func TestNewClientRefusesLoopbackDial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()

	_, err := NewClient(0, "feed_url").Get(server.URL)
	if err == nil {
		t.Fatal("expected the dialer to refuse a loopback destination")
	}
	if !strings.Contains(err.Error(), "blocked address") {
		t.Fatalf("error should explain the block, got %v", err)
	}
}
