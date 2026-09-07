package rss

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/savisaluwadana/DevRelOS/internal/connectors"
)

func TestValidateConfigBlocksUnsafeDestinations(t *testing.T) {
	p := New()
	blocked := []string{
		"http://example.com/feed.xml",
		"https://localhost/feed.xml",
		"https://127.0.0.1/feed.xml",
		"https://10.0.0.10/feed.xml",
		"https://user:pass@example.com/feed.xml",
	}
	for _, feedURL := range blocked {
		if err := p.ValidateConfig(map[string]any{"feed_url": feedURL}); err == nil {
			t.Fatalf("expected %q to be blocked", feedURL)
		}
	}
	if err := p.ValidateConfig(map[string]any{"feed_url": "https://example.com/feed.xml"}); err != nil {
		t.Fatalf("expected public HTTPS URL to validate: %v", err)
	}
}

func TestParseRSSAndAtom(t *testing.T) {
	rssBody := []byte(`<?xml version="1.0"?>
	<rss version="2.0"><channel><item>
		<guid>post-1</guid><title>Platform engineering update</title>
		<link>https://example.com/post-1</link>
		<description><![CDATA[<p>Kubernetes &amp; developer experience.</p>]]></description>
		<pubDate>Mon, 07 Sep 2026 10:00:00 +0000</pubDate><author>Alice</author>
	</item></channel></rss>`)
	entries, err := parseFeed(rssBody)
	if err != nil {
		t.Fatalf("parse RSS: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != "post-1" || entries[0].Author != "Alice" {
		t.Fatalf("unexpected RSS entries: %+v", entries)
	}

	atomBody := []byte(`<?xml version="1.0"?>
	<feed xmlns="http://www.w3.org/2005/Atom"><entry>
		<id>tag:example.com,2026:2</id><title>DevTools release</title>
		<link rel="alternate" href="https://example.com/post-2" />
		<summary>Developer workflow improvements</summary>
		<updated>2026-09-07T10:00:00Z</updated><author><name>Bob</name></author>
	</entry></feed>`)
	entries, err = parseFeed(atomBody)
	if err != nil {
		t.Fatalf("parse Atom: %v", err)
	}
	if len(entries) != 1 || entries[0].Link != "https://example.com/post-2" || entries[0].Author != "Bob" {
		t.Fatalf("unexpected Atom entries: %+v", entries)
	}
}

func TestFetchNormalizesAndFiltersFeedItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
		<rss version="2.0"><channel>
			<item><guid>1</guid><title>Platform engineering guide</title><link>https://example.com/1</link><description><![CDATA[<p>Kubernetes developer experience pain.</p>]]></description><pubDate>Mon, 07 Sep 2026 10:00:00 +0000</pubDate><author>Alice</author></item>
			<item><guid>2</guid><title>Database tuning</title><link>https://example.com/2</link><description>Indexes and storage.</description><pubDate>Mon, 07 Sep 2026 09:00:00 +0000</pubDate></item>
		</channel></rss>`))
	}))
	defer server.Close()

	p := New()
	p.allowUnsafeLocal = true
	p.client = server.Client()
	result, err := p.Fetch(context.Background(), map[string]any{
		"feed_url": server.URL,
		"query":    "platform engineering",
		"topics":   []any{"developer experience", "kubernetes"},
	}, connectors.FetchRequest{PageLimit: 10})
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if result.RequestsMade != 1 || len(result.Records) != 1 {
		t.Fatalf("requests=%d records=%d", result.RequestsMade, len(result.Records))
	}
	record := result.Records[0]
	if record.ExternalID != "1" || record.CanonicalURL != "https://example.com/1" {
		t.Fatalf("unexpected record identity: %+v", record)
	}
	if strings.Contains(record.Normalized.Body, "<p>") || !strings.Contains(record.Normalized.Body, "Kubernetes developer experience pain") {
		t.Fatalf("body not normalized: %q", record.Normalized.Body)
	}
	if got := strings.Join(record.Normalized.Topics, ","); got != "developer-experience,kubernetes" {
		t.Fatalf("topics=%q", got)
	}
	if record.SourceTimestamp == nil {
		t.Fatal("expected parsed source timestamp")
	}
}
