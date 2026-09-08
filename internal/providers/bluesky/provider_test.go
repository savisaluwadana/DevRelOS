package bluesky

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/savisaluwadana/DevRelOS/internal/connectors"
)

func TestValidateConfigRequiresQuery(t *testing.T) {
	p := New()
	if err := p.ValidateConfig(map[string]any{}); err == nil {
		t.Fatal("missing query was accepted")
	}
	if err := p.ValidateConfig(map[string]any{"query": "   "}); err == nil {
		t.Fatal("blank query was accepted")
	}
	// A non-string query must be rejected, not coerced: Fetch reads this key.
	if err := p.ValidateConfig(map[string]any{"query": 42}); err == nil {
		t.Fatal("non-string query was accepted")
	}
	if err := p.ValidateConfig(map[string]any{"query": "platform engineering"}); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
}

func TestValidateConfigRejectsUnsafeBaseURL(t *testing.T) {
	p := New()
	for _, base := range []string{
		"http://public.api.bsky.app",
		"https://169.254.169.254",
		"https://127.0.0.1",
		"https://localhost",
		"https://10.1.2.3",
		"https://user:pass@public.api.bsky.app",
	} {
		err := p.ValidateConfig(map[string]any{"query": "k8s", "base_url": base})
		if err == nil {
			t.Errorf("base_url %q was accepted", base)
		}
	}
	if err := p.ValidateConfig(map[string]any{"query": "k8s", "base_url": "https://public.api.bsky.app"}); err != nil {
		t.Fatalf("legitimate base_url rejected: %v", err)
	}
}

func TestFetchNormalizesPosts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("q"); got != "platform engineering" {
			t.Errorf("query forwarded as %q", got)
		}
		if got := r.URL.Query().Get("sort"); got != "latest" {
			t.Errorf("sort = %q, want latest", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
		  "cursor": "next-page",
		  "posts": [{
		    "uri": "at://did:plc:abc/app.bsky.feed.post/xyz",
		    "cid": "cid-1",
		    "author": {"did": "did:plc:abc", "handle": "dev.example.com", "displayName": "Dev"},
		    "record": {"text": "Kubernetes upgrades are painful", "createdAt": "2026-09-01T10:00:00Z"},
		    "replyCount": 2, "repostCount": 3, "likeCount": 5, "quoteCount": 1
		  }]
		}`)
	}))
	defer server.Close()

	p := New()
	p.allowUnsafeLocal = true
	p.client = server.Client()

	result, err := p.Fetch(context.Background(), map[string]any{
		"query":    "platform engineering",
		"base_url": server.URL,
		"topics":   []any{"kubernetes"},
	}, connectors.FetchRequest{PageLimit: 10})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if result.NextCursor != "next-page" || result.RequestsMade != 1 || len(result.Records) != 1 {
		t.Fatalf("cursor=%q requests=%d records=%d", result.NextCursor, result.RequestsMade, len(result.Records))
	}

	record := result.Records[0]
	if record.ExternalID != "at://did:plc:abc/app.bsky.feed.post/xyz" {
		t.Errorf("ExternalID = %q", record.ExternalID)
	}
	if !strings.Contains(record.Normalized.Body, "Kubernetes upgrades are painful") {
		t.Errorf("Body = %q", record.Normalized.Body)
	}
	if record.Normalized.AuthorHandle != "dev.example.com" {
		t.Errorf("AuthorHandle = %q", record.Normalized.AuthorHandle)
	}
	// likes + 2*(reposts + replies + quotes) = 5 + 2*(3+2+1) = 17
	if record.Normalized.EngagementScore != 17 {
		t.Errorf("EngagementScore = %d, want 17", record.Normalized.EngagementScore)
	}
	if record.SourceTimestamp == nil {
		t.Error("expected the record createdAt to be parsed")
	}
	// The raw payload must not carry post text; only provenance.
	if _, leaked := record.Payload["text"]; leaked {
		t.Error("payload should hold provenance, not redistributed post text")
	}
}

func TestFetchRejectsUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	p := New()
	p.allowUnsafeLocal = true
	p.client = server.Client()
	if _, err := p.Fetch(context.Background(), map[string]any{"query": "k8s", "base_url": server.URL}, connectors.FetchRequest{}); err == nil {
		t.Fatal("expected an error for a 429 response")
	}
}
