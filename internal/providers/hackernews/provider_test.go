package hackernews

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/savisaluwadana/DevRelOS/internal/connectors"
)

func TestValidateConfig(t *testing.T) {
	p := New()
	if err := p.ValidateConfig(map[string]any{"query": "platform engineering", "feed": "new"}); err != nil {
		t.Fatalf("expected valid config: %v", err)
	}
	if err := p.ValidateConfig(map[string]any{"query": ""}); err == nil {
		t.Fatal("expected query validation error")
	}
	if err := p.ValidateConfig(map[string]any{"query": "kubernetes", "feed": "random"}); err == nil {
		t.Fatal("expected feed validation error")
	}
}

func TestFetchFiltersAndNormalizesStories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/newstories.json":
			_, _ = w.Write([]byte(`[101,102]`))
		case "/item/101.json":
			_, _ = w.Write([]byte(`{
				"id": 101,
				"type": "story",
				"by": "alice",
				"time": 1788775200,
				"title": "Platform engineering lessons",
				"text": "<p>Kubernetes developer experience is hard &amp; expensive.</p>",
				"url": "https://example.com/post",
				"score": 20,
				"descendants": 5
			}`))
		case "/item/102.json":
			_, _ = w.Write([]byte(`{
				"id": 102,
				"type": "story",
				"by": "bob",
				"time": 1788775200,
				"title": "Database indexing tips",
				"text": "Nothing about platform teams.",
				"score": 50,
				"descendants": 10
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	p := New()
	p.baseURL = server.URL
	result, err := p.Fetch(context.Background(), map[string]any{
		"query":  "platform engineering",
		"feed":   "new",
		"topics": []any{"developer experience", "kubernetes"},
	}, connectors.FetchRequest{PageLimit: 5})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if result.RequestsMade != 3 {
		t.Fatalf("requests made = %d, want 3", result.RequestsMade)
	}
	if len(result.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(result.Records))
	}
	record := result.Records[0]
	if record.ExternalID != "101" {
		t.Fatalf("external id = %q", record.ExternalID)
	}
	if record.Normalized.EngagementScore != 30 {
		t.Fatalf("engagement = %d, want 30", record.Normalized.EngagementScore)
	}
	if strings.Contains(record.Normalized.Body, "<p>") || !strings.Contains(record.Normalized.Body, "hard & expensive") {
		t.Fatalf("body not normalized: %q", record.Normalized.Body)
	}
	if got := strings.Join(record.Normalized.Topics, ","); got != "developer-experience,kubernetes" {
		t.Fatalf("topics = %q", got)
	}
	if !strings.Contains(record.CanonicalURL, "item?id=101") {
		t.Fatalf("canonical URL = %q", record.CanonicalURL)
	}
}
