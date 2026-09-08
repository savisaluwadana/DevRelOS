package developersevents

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/savisaluwadana/DevRelOS/internal/connectors"
)

func TestValidateConfigRejectsUnsafeFeedURL(t *testing.T) {
	p := New()
	// feed_url was previously only checked for being a string, so any of these
	// would have been fetched and stored.
	for _, feed := range []string{
		"http://developers.events/all-events.json",
		"https://169.254.169.254/latest/meta-data/",
		"https://127.0.0.1:8080/all-events.json",
		"https://localhost/all-events.json",
		"https://192.168.0.1/all-events.json",
		"https://user:pass@developers.events/all-events.json",
	} {
		if err := p.ValidateConfig(map[string]any{"feed_url": feed}); err == nil {
			t.Errorf("feed_url %q was accepted", feed)
		}
	}
}

func TestValidateConfigAcceptsDefaultsAndPublicHTTPS(t *testing.T) {
	p := New()
	if err := p.ValidateConfig(map[string]any{}); err != nil {
		t.Fatalf("absent feed_url should fall back to the default: %v", err)
	}
	if err := p.ValidateConfig(map[string]any{"feed_url": ""}); err != nil {
		t.Fatalf("empty feed_url should fall back to the default: %v", err)
	}
	if err := p.ValidateConfig(map[string]any{"feed_url": 7}); err == nil {
		t.Fatal("non-string feed_url was accepted")
	}
	if err := p.ValidateConfig(map[string]any{"feed_url": "https://developers.events/all-events.json"}); err != nil {
		t.Fatalf("legitimate feed_url rejected: %v", err)
	}
}

func TestFetchNormalizesAndRespectsPageLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
		  {"name":"KubeCon EU","date":"2027-03-01","hyperlink":"https://example.com/kubecon"},
		  {"name":"PlatformCon","date":"2027-06-01","hyperlink":"https://example.com/platformcon"},
		  {"name":"SREday","date":"2027-09-01","hyperlink":"https://example.com/sreday"}
		]`)
	}))
	defer server.Close()

	p := New()
	p.allowUnsafeLocal = true
	p.client = server.Client()

	result, err := p.Fetch(context.Background(), map[string]any{"feed_url": server.URL},
		connectors.FetchRequest{PageLimit: 2})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if result.RequestsMade != 1 {
		t.Errorf("RequestsMade = %d, want 1", result.RequestsMade)
	}
	if len(result.Records) != 2 {
		t.Fatalf("PageLimit not honoured: got %d records, want 2", len(result.Records))
	}
	if result.Records[0].CanonicalURL != "https://example.com/kubecon" {
		t.Errorf("CanonicalURL = %q", result.Records[0].CanonicalURL)
	}
	// ExternalID is a content hash, so it must be stable and distinct per event.
	if result.Records[0].ExternalID == result.Records[1].ExternalID {
		t.Error("distinct events produced the same ExternalID")
	}
	if len(result.Records[0].ExternalID) != 32 {
		t.Errorf("ExternalID = %q, want a 32-char hex digest", result.Records[0].ExternalID)
	}

	// The same event must hash identically across runs, or every poll
	// re-ingests it as new.
	again, err := p.Fetch(context.Background(), map[string]any{"feed_url": server.URL},
		connectors.FetchRequest{PageLimit: 2})
	if err != nil {
		t.Fatalf("second fetch: %v", err)
	}
	if again.Records[0].ExternalID != result.Records[0].ExternalID {
		t.Error("ExternalID is not stable across fetches")
	}
}

func TestFetchHandlesEmptyFeed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `[]`)
	}))
	defer server.Close()

	p := New()
	p.allowUnsafeLocal = true
	p.client = server.Client()
	result, err := p.Fetch(context.Background(), map[string]any{"feed_url": server.URL}, connectors.FetchRequest{})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(result.Records) != 0 {
		t.Fatalf("expected no records, got %d", len(result.Records))
	}
}
