package githubdiscussions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/savisaluwadana/DevRelOS/internal/connectors"
)

func TestValidateConfigRequiresTokenEnv(t *testing.T) {
	p := New()
	if err := p.ValidateConfig(map[string]any{"repository": "owner/repo"}); err == nil {
		t.Fatal("expected token_env validation error")
	}
	const envName = "DEVRELOS_TEST_DISCUSSION_VALIDATE_TOKEN"
	_ = os.Setenv(envName, "token")
	defer os.Unsetenv(envName)
	if err := p.ValidateConfig(map[string]any{"repository": "owner/repo", "token_env": envName}); err != nil {
		t.Fatalf("expected valid config: %v", err)
	}
}

func TestFetchNormalizesDiscussionAndCursor(t *testing.T) {
	const envName = "DEVRELOS_TEST_DISCUSSION_TOKEN"
	_ = os.Setenv(envName, "discussion-secret")
	defer os.Unsetenv(envName)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer discussion-secret" {
			t.Fatalf("authorization=%q", got)
		}
		var body graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.Variables["owner"] != "acme" || body.Variables["name"] != "tool" {
			t.Fatalf("variables=%v", body.Variables)
		}
		if body.Variables["after"] != "cursor-1" {
			t.Fatalf("after=%v", body.Variables["after"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {"repository": {"discussions": {
				"pageInfo": {"hasNextPage": true, "endCursor": "cursor-2"},
				"nodes": [{
					"number": 7,
					"title": "Platform engineering onboarding pain",
					"bodyText": "Kubernetes setup is confusing for new developers.",
					"url": "https://github.com/acme/tool/discussions/7",
					"createdAt": "2026-09-01T10:00:00Z",
					"updatedAt": "2026-09-07T10:00:00Z",
					"closed": false,
					"isAnswered": true,
					"upvoteCount": 8,
					"comments": {"totalCount": 6},
					"category": {"name": "Q&A"},
					"author": {"login": "alice"}
				}]}}
			}
		}`))
	}))
	defer server.Close()

	p := New()
	p.baseURL = server.URL
	result, err := p.Fetch(context.Background(), map[string]any{
		"repository": "acme/tool",
		"token_env":  envName,
		"query":      "platform engineering",
		"topics":     []any{"developer experience"},
	}, connectors.FetchRequest{PageLimit: 10, Cursor: "cursor-1"})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if result.NextCursor != "cursor-2" {
		t.Fatalf("next cursor=%q", result.NextCursor)
	}
	if len(result.Records) != 1 {
		t.Fatalf("records=%d, want 1", len(result.Records))
	}
	record := result.Records[0]
	if record.ExternalID != "acme/tool:discussion:7" {
		t.Fatalf("external id=%q", record.ExternalID)
	}
	if record.Normalized.EngagementScore != 20 {
		t.Fatalf("engagement=%d, want 20", record.Normalized.EngagementScore)
	}
	if record.Normalized.AuthorHandle != "alice" {
		t.Fatalf("author=%q", record.Normalized.AuthorHandle)
	}
	joined := strings.Join(record.Normalized.Topics, ",")
	if !strings.Contains(joined, "developer-experience") || !strings.Contains(joined, "q&a") {
		t.Fatalf("topics=%v", record.Normalized.Topics)
	}
}

func TestFetchSurfacesGraphQLErrors(t *testing.T) {
	const envName = "DEVRELOS_TEST_DISCUSSION_ERROR_TOKEN"
	_ = os.Setenv(envName, "discussion-secret")
	defer os.Unsetenv(envName)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"Discussions are disabled"}]}`))
	}))
	defer server.Close()

	p := New()
	p.baseURL = server.URL
	_, err := p.Fetch(context.Background(), map[string]any{
		"repository": "acme/tool",
		"token_env":  envName,
	}, connectors.FetchRequest{PageLimit: 5})
	if err == nil || !strings.Contains(err.Error(), "Discussions are disabled") {
		t.Fatalf("expected GraphQL error, got %v", err)
	}
}
