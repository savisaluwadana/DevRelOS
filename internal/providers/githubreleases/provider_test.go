package githubreleases

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/savisaluwadana/DevRelOS/internal/connectors"
)

func TestValidateConfig(t *testing.T) {
	p := New()
	if err := p.ValidateConfig(map[string]any{"repository": "owner/repo"}); err != nil {
		t.Fatalf("expected valid config: %v", err)
	}
	if err := p.ValidateConfig(map[string]any{"repository": "bad"}); err == nil {
		t.Fatal("expected owner/repo validation error")
	}
}

func TestFetchFiltersAndNormalizesReleases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/tool/releases" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"id": 100,
				"tag_name": "v1.2.0",
				"name": "Platform reliability release",
				"body": "Kubernetes deployment and developer experience improvements.",
				"html_url": "https://github.com/acme/tool/releases/tag/v1.2.0",
				"draft": false,
				"prerelease": false,
				"created_at": "2026-09-01T10:00:00Z",
				"published_at": "2026-09-02T10:00:00Z",
				"author": {"login": "release-bot"},
				"assets": [{"download_count": 350}, {"download_count": 800}]
			},
			{
				"id": 101,
				"tag_name": "v1.3.0-rc1",
				"name": "Platform engineering preview",
				"body": "Kubernetes preview.",
				"html_url": "https://github.com/acme/tool/releases/tag/v1.3.0-rc1",
				"draft": false,
				"prerelease": true,
				"created_at": "2026-09-03T10:00:00Z",
				"published_at": "2026-09-03T10:00:00Z",
				"author": {"login": "release-bot"},
				"assets": []
			}
		]`))
	}))
	defer server.Close()

	p := New()
	p.baseURL = server.URL
	result, err := p.Fetch(context.Background(), map[string]any{
		"repository": "acme/tool",
		"query":      "platform",
		"topics":     []any{"release intelligence", "kubernetes"},
	}, connectors.FetchRequest{PageLimit: 10})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("records=%d, want 1", len(result.Records))
	}
	record := result.Records[0]
	if record.ExternalID != "acme/tool:release:100" {
		t.Fatalf("external id=%q", record.ExternalID)
	}
	if record.Normalized.EngagementScore != 1000 {
		t.Fatalf("engagement=%d, want capped 1000", record.Normalized.EngagementScore)
	}
	if got := strings.Join(record.Normalized.Topics, ","); got != "release-intelligence,kubernetes" {
		t.Fatalf("topics=%q", got)
	}
	if record.SourceTimestamp == nil || record.SourceTimestamp.Format("2006-01-02") != "2026-09-02" {
		t.Fatalf("unexpected timestamp=%v", record.SourceTimestamp)
	}
}

func TestFetchUsesTokenEnvironmentVariable(t *testing.T) {
	const envName = "DEVRELOS_TEST_RELEASE_TOKEN"
	if err := os.Setenv(envName, "release-secret"); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv(envName)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer release-secret" {
			t.Fatalf("authorization=%q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	p := New()
	p.baseURL = server.URL
	_, err := p.Fetch(context.Background(), map[string]any{
		"repository": "acme/tool",
		"token_env":  envName,
	}, connectors.FetchRequest{PageLimit: 1})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
}
