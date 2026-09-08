package ocg

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/savisaluwadana/DevRelOS/internal/connectors"
)

func TestValidateConfigRejectsUnsafeBaseURL(t *testing.T) {
	p := New()
	for _, base := range []string{
		"http://ocgroups.dev",
		"https://169.254.169.254",
		"https://127.0.0.1:9000",
		"https://localhost",
		"https://172.16.5.4",
		"https://user:pass@ocgroups.dev",
	} {
		if err := p.ValidateConfig(map[string]any{"base_url": base}); err == nil {
			t.Errorf("base_url %q was accepted", base)
		}
	}
	if err := p.ValidateConfig(map[string]any{}); err != nil {
		t.Fatalf("absent base_url should fall back to the default: %v", err)
	}
	if err := p.ValidateConfig(map[string]any{"base_url": "https://ocgroups.dev"}); err != nil {
		t.Fatalf("legitimate base_url rejected: %v", err)
	}
}

func TestFetchSendsPagingAndFilterParams(t *testing.T) {
	var seen url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"total": 2, "groups": [
		  {
		    "active": true,
		    "community_display_name": "CNCF",
		    "community_name": "cncf",
		    "group_id": "g-1",
		    "name": "Cloud Native Colombo",
		    "slug": "cloud-native-colombo",
		    "city": "Colombo",
		    "country_name": "Sri Lanka",
		    "description_short": "Kubernetes and platform engineering meetup",
		    "created_at": 1750000000
		  },
		  {
		    "active": false,
		    "community_name": "cncf",
		    "group_id": "g-2",
		    "name": "Dormant Group",
		    "slug": "dormant"
		  }
		]}`)
	}))
	defer server.Close()

	p := New()
	p.allowUnsafeLocal = true
	p.client = server.Client()

	result, err := p.Fetch(context.Background(), map[string]any{
		"base_url":       server.URL,
		"community":      "cncf",
		"query":          "platform",
		"region":         "asia",
		"group_category": "meetup",
	}, connectors.FetchRequest{PageLimit: 25, Cursor: "50"})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if result.RequestsMade != 1 {
		t.Errorf("RequestsMade = %d, want 1", result.RequestsMade)
	}
	// The inactive group must be dropped, not ingested as a live community.
	if len(result.Records) != 1 {
		t.Fatalf("got %d records, want 1 (the inactive group should be skipped)", len(result.Records))
	}
	record := result.Records[0]
	if record.ExternalID != "g-1" {
		t.Errorf("ExternalID = %q", record.ExternalID)
	}
	if record.Normalized.Name != "Cloud Native Colombo" || record.Normalized.Platform != "ocg" {
		t.Errorf("normalized identity = %+v", record.Normalized)
	}
	if record.Normalized.City != "Colombo" || record.Normalized.Country != "Sri Lanka" {
		t.Errorf("location = %q / %q", record.Normalized.City, record.Normalized.Country)
	}
	if record.CanonicalURL != server.URL+"/cncf/group/cloud-native-colombo" {
		t.Errorf("CanonicalURL = %q", record.CanonicalURL)
	}
	if record.SourceTimestamp == nil {
		t.Error("created_at should be parsed into a source timestamp")
	}

	// The cursor is an offset; a wrong translation silently re-reads page one.
	if got := seen.Get("offset"); got != "50" {
		t.Errorf("offset = %q, want the cursor value 50", got)
	}
	if got := seen.Get("limit"); got != "25" {
		t.Errorf("limit = %q, want 25", got)
	}
	if got := seen.Get("community[0]"); got != "cncf" {
		t.Errorf("community[0] = %q", got)
	}
	if got := seen.Get("ts_query"); got != "platform" {
		t.Errorf("ts_query = %q", got)
	}
	if got := seen.Get("region[0]"); got != "asia" {
		t.Errorf("region[0] = %q", got)
	}
	if got := seen.Get("group_category[0]"); got != "meetup" {
		t.Errorf("group_category[0] = %q", got)
	}
}

func TestFetchClampsPageLimit(t *testing.T) {
	var seen string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Query().Get("limit")
		fmt.Fprint(w, `{"total":0,"groups":[]}`)
	}))
	defer server.Close()

	p := New()
	p.allowUnsafeLocal = true
	p.client = server.Client()
	if _, err := p.Fetch(context.Background(), map[string]any{"base_url": server.URL},
		connectors.FetchRequest{PageLimit: 5000}); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	limit, err := strconv.Atoi(seen)
	if err != nil {
		t.Fatalf("limit = %q, not numeric", seen)
	}
	if limit > 100 {
		t.Fatalf("limit = %d, want it clamped to at most 100", limit)
	}
}

func TestFetchRejectsUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	p := New()
	p.allowUnsafeLocal = true
	p.client = server.Client()
	if _, err := p.Fetch(context.Background(), map[string]any{"base_url": server.URL}, connectors.FetchRequest{}); err == nil {
		t.Fatal("expected an error for a 502 response")
	}
}
