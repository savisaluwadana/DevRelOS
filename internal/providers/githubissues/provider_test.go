package githubissues

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
	if err := p.ValidateConfig(map[string]any{"repository": "owner/repo", "state": "open"}); err != nil {
		t.Fatalf("expected valid config: %v", err)
	}
	if err := p.ValidateConfig(map[string]any{"repository": "not-a-repository"}); err == nil {
		t.Fatal("expected owner/repo validation error")
	}
	if err := p.ValidateConfig(map[string]any{"repository": "owner/repo", "state": "unknown"}); err == nil {
		t.Fatal("expected state validation error")
	}
}

func TestFetchNormalizesIssuesAndSkipsPullRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/tool/issues" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("state"); got != "open" {
			t.Fatalf("unexpected state %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{
				"number": 42,
				"title": "Kubernetes deployment is confusing",
				"body": "The install docs leave the platform setup unclear.",
				"html_url": "https://github.com/acme/tool/issues/42",
				"state": "open",
				"comments": 4,
				"created_at": "2026-09-01T10:00:00Z",
				"updated_at": "2026-09-07T10:00:00Z",
				"user": {"login": "dev-user", "html_url": "https://github.com/dev-user"},
				"labels": [{"name": "documentation"}, {"name": "good first issue"}],
				"reactions": {"total_count": 3}
			},
			{
				"number": 43,
				"title": "Kubernetes PR",
				"body": "implementation",
				"html_url": "https://github.com/acme/tool/pull/43",
				"state": "open",
				"comments": 1,
				"created_at": "2026-09-01T10:00:00Z",
				"updated_at": "2026-09-07T10:00:00Z",
				"user": {"login": "contributor"},
				"labels": [],
				"reactions": {"total_count": 0},
				"pull_request": {}
			}
		]`))
	}))
	defer server.Close()

	p := New()
	p.baseURL = server.URL
	result, err := p.Fetch(context.Background(), map[string]any{
		"repository": "acme/tool",
		"state":      "open",
		"query":      "kubernetes",
		"topics":     []any{"platform engineering"},
	}, connectors.FetchRequest{PageLimit: 10})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if result.RequestsMade != 1 {
		t.Fatalf("requests made = %d, want 1", result.RequestsMade)
	}
	if len(result.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(result.Records))
	}
	record := result.Records[0]
	if record.ExternalID != "acme/tool#42" {
		t.Fatalf("external id = %q", record.ExternalID)
	}
	if record.Normalized.EngagementScore != 15 {
		t.Fatalf("engagement = %d, want 15", record.Normalized.EngagementScore)
	}
	joined := strings.Join(record.Normalized.Topics, ",")
	for _, topic := range []string{"platform-engineering", "documentation", "good-first-issue"} {
		if !strings.Contains(joined, topic) {
			t.Fatalf("topics %v missing %q", record.Normalized.Topics, topic)
		}
	}
}

func TestFetchUsesTokenEnvironmentVariable(t *testing.T) {
	const envName = "DEVRELOS_TEST_GITHUB_TOKEN"
	if err := os.Setenv(envName, "secret-token"); err != nil {
		t.Fatal(err)
	}
	defer os.Unsetenv(envName)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Fatalf("authorization = %q", got)
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
