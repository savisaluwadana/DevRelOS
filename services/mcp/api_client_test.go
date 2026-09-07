package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestAPIClientListForwardsAuthAndQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/signals" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("q") != "kubernetes" {
			t.Fatalf("query not forwarded: %s", r.URL.RawQuery)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("authorization header not forwarded")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "signal-1", "title": "Deployment issue"}})
	}))
	defer server.Close()

	client, err := newAPIClient(server.URL, "test-token")
	if err != nil {
		t.Fatal(err)
	}
	items, err := client.list(context.Background(), "/api/v1/signals", url.Values{"q": {"kubernetes"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0]["id"] != "signal-1" {
		t.Fatalf("unexpected items: %#v", items)
	}
}

func TestAPIClientCalendarUnwrapsItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/calendar" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("from") != "2026-09-01T00:00:00Z" {
			t.Fatalf("calendar query not forwarded: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"from": "2026-09-01T00:00:00Z",
			"to":   "2026-10-01T00:00:00Z",
			"items": []map[string]any{{"id": "cfp:1", "kind": "cfp"}},
		})
	}))
	defer server.Close()

	client, err := newAPIClient(server.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	items, err := client.calendar(context.Background(), url.Values{"from": {"2026-09-01T00:00:00Z"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0]["kind"] != "cfp" {
		t.Fatalf("unexpected calendar items: %#v", items)
	}
}

func TestAPIClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/campaigns/campaign-1/report" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"outcomeScore": 82})
	}))
	defer server.Close()

	client, err := newAPIClient(server.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	item, err := client.get(context.Background(), "/api/v1/campaigns/campaign-1/report", nil)
	if err != nil {
		t.Fatal(err)
	}
	if item["outcomeScore"] != float64(82) {
		t.Fatalf("unexpected campaign report: %#v", item)
	}
}

func TestAPIClientCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/submissions" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["cfpId"] != "cfp-1" {
			t.Fatalf("unexpected body: %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "submission-1", "status": "draft"})
	}))
	defer server.Close()

	client, err := newAPIClient(server.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	item, err := client.create(context.Background(), "/api/v1/submissions", map[string]any{"cfpId": "cfp-1"})
	if err != nil {
		t.Fatal(err)
	}
	if item["id"] != "submission-1" {
		t.Fatalf("unexpected item: %#v", item)
	}
}

func TestAPIClientRejectsNonDomainPath(t *testing.T) {
	client, err := newAPIClient("http://127.0.0.1:8080", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.doJSON(context.Background(), http.MethodGet, "/internal/secrets", nil, nil, nil); err == nil {
		t.Fatal("expected non-domain API path to be rejected")
	}
}

func TestBoundedLimit(t *testing.T) {
	if got := boundedLimit(0, 50, 200); got != 50 {
		t.Fatalf("fallback=%d", got)
	}
	if got := boundedLimit(1000, 50, 200); got != 200 {
		t.Fatalf("cap=%d", got)
	}
	if got := boundedLimit(25, 50, 200); got != 25 {
		t.Fatalf("value=%d", got)
	}
}
