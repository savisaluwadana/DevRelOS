package rss

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/savisaluwadana/DevRelOS/internal/connectors"
	"github.com/savisaluwadana/DevRelOS/internal/providers/safehttp"
)

const maxFeedBytes = 2 << 20

var htmlTag = regexp.MustCompile(`<[^>]+>`)

type Provider struct {
	client           *http.Client
	allowUnsafeLocal bool
}

func New() *Provider {
	return &Provider{client: safehttp.NewClient(20*time.Second, "feed_url")}
}

func (p *Provider) ID() string { return "rss" }

func (p *Provider) Capabilities() []connectors.Capability {
	return []connectors.Capability{connectors.CapabilityTimeline}
}

func (p *Provider) ValidateConfig(config map[string]any) error {
	feedURL, _ := config["feed_url"].(string)
	feedURL = strings.TrimSpace(feedURL)
	if feedURL == "" {
		return errors.New("feed_url is required")
	}
	parsed, err := url.Parse(feedURL)
	if err != nil {
		return errors.New("feed_url must be a valid URL")
	}
	return safehttp.ValidateURL(parsed, "feed_url", p.allowUnsafeLocal)
}

func (p *Provider) Policy(config map[string]any) connectors.Policy {
	return connectors.Policy{
		RequestsPerMinute: 10,
		DailyRequestLimit: 1000,
		MonthlyBudgetUSD:  0,
		StoreRawPayload:   false,
		CommercialUseOK:   true,
		Notes:             "Fetches HTTPS RSS 2.0 or Atom feeds through an SSRF-safe client. Private, loopback, link-local and otherwise non-public destinations are blocked after DNS resolution and again on redirects. DevRelOS stores normalized evidence and canonical source links by default.",
	}
}

func (p *Provider) Fetch(ctx context.Context, config map[string]any, request connectors.FetchRequest) (connectors.FetchResult, error) {
	if err := p.ValidateConfig(config); err != nil {
		return connectors.FetchResult{}, err
	}
	feedURL := strings.TrimSpace(config["feed_url"].(string))
	limit := request.PageLimit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return connectors.FetchResult{}, err
	}
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml;q=0.9")
	req.Header.Set("User-Agent", "DevRelOS/0.4 (+https://github.com/savisaluwadana/DevRelOS)")

	resp, err := p.client.Do(req)
	if err != nil {
		return connectors.FetchResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return connectors.FetchResult{}, fmt.Errorf("feed returned %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFeedBytes+1))
	if err != nil {
		return connectors.FetchResult{}, err
	}
	if len(body) > maxFeedBytes {
		return connectors.FetchResult{}, errors.New("feed exceeds 2 MiB limit")
	}

	entries, err := parseFeed(body)
	if err != nil {
		return connectors.FetchResult{}, err
	}
	queryTerms := splitQuery(stringConfig(config, "query", ""))
	topics := topicsFromConfig(config)
	result := connectors.FetchResult{Records: make([]connectors.RawRecord, 0, min(limit, len(entries))), RequestsMade: 1}
	for _, entry := range entries {
		if len(result.Records) >= limit {
			break
		}
		if !matchesAll(queryTerms, strings.ToLower(entry.Title+"\n"+entry.Body)) {
			continue
		}

		externalID := strings.TrimSpace(entry.ID)
		if externalID == "" {
			externalID = strings.TrimSpace(entry.Link)
		}
		if externalID == "" {
			hash := sha256.Sum256([]byte(entry.Title + "\n" + entry.Published))
			externalID = "feed:" + hex.EncodeToString(hash[:])
		}
		canonicalURL := strings.TrimSpace(entry.Link)
		if canonicalURL == "" {
			canonicalURL = feedURL
		}
		timestamp := parsePublished(entry.Published)

		result.Records = append(result.Records, connectors.RawRecord{
			ExternalID:      externalID,
			CanonicalURL:    canonicalURL,
			SourceTimestamp: timestamp,
			Payload: map[string]any{
				"feed_url":  feedURL,
				"entry_id":  entry.ID,
				"published": entry.Published,
			},
			Normalized: connectors.NormalizedRecord{
				Kind:         "signal",
				Shape:        connectors.SourceShapeAnnouncement,
				Title:        cleanText(entry.Title),
				Body:         cleanText(entry.Body),
				AuthorName:   cleanText(entry.Author),
				AuthorHandle: cleanText(entry.Author),
				Topics:       topics,
			},
		})
	}
	return result, nil
}

type feedEntry struct {
	ID        string
	Title     string
	Body      string
	Link      string
	Published string
	Author    string
}

type rssDocument struct {
	Channel struct {
		Items []struct {
			GUID        string `xml:"guid"`
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
			Content     string `xml:"encoded"`
			PubDate     string `xml:"pubDate"`
			Creator     string `xml:"creator"`
			Author      string `xml:"author"`
		} `xml:"item"`
	} `xml:"channel"`
}

type atomDocument struct {
	Entries []struct {
		ID        string `xml:"id"`
		Title     string `xml:"title"`
		Summary   string `xml:"summary"`
		Content   string `xml:"content"`
		Updated   string `xml:"updated"`
		Published string `xml:"published"`
		Links     []struct {
			Href string `xml:"href,attr"`
			Rel  string `xml:"rel,attr"`
		} `xml:"link"`
		Author struct {
			Name string `xml:"name"`
		} `xml:"author"`
	} `xml:"entry"`
}

func parseFeed(body []byte) ([]feedEntry, error) {
	var root struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("invalid XML feed: %w", err)
	}

	switch strings.ToLower(root.XMLName.Local) {
	case "rss":
		var doc rssDocument
		if err := xml.Unmarshal(body, &doc); err != nil {
			return nil, err
		}
		entries := make([]feedEntry, 0, len(doc.Channel.Items))
		for _, item := range doc.Channel.Items {
			bodyText := item.Content
			if strings.TrimSpace(bodyText) == "" {
				bodyText = item.Description
			}
			author := item.Creator
			if strings.TrimSpace(author) == "" {
				author = item.Author
			}
			entries = append(entries, feedEntry{
				ID: item.GUID, Title: item.Title, Body: bodyText, Link: item.Link,
				Published: item.PubDate, Author: author,
			})
		}
		return entries, nil

	case "feed":
		var doc atomDocument
		if err := xml.Unmarshal(body, &doc); err != nil {
			return nil, err
		}
		entries := make([]feedEntry, 0, len(doc.Entries))
		for _, item := range doc.Entries {
			bodyText := item.Content
			if strings.TrimSpace(bodyText) == "" {
				bodyText = item.Summary
			}
			published := item.Published
			if strings.TrimSpace(published) == "" {
				published = item.Updated
			}
			link := ""
			for _, candidate := range item.Links {
				if candidate.Rel == "" || candidate.Rel == "alternate" {
					link = candidate.Href
					break
				}
			}
			entries = append(entries, feedEntry{
				ID: item.ID, Title: item.Title, Body: bodyText, Link: link,
				Published: published, Author: item.Author.Name,
			})
		}
		return entries, nil

	default:
		return nil, fmt.Errorf("unsupported feed root %q; expected RSS or Atom", root.XMLName.Local)
	}
}

func parsePublished(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	layouts := []string{time.RFC3339Nano, time.RFC3339, time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC822}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			parsed = parsed.UTC()
			return &parsed
		}
	}
	return nil
}

func cleanText(value string) string {
	value = html.UnescapeString(value)
	value = htmlTag.ReplaceAllString(value, " ")
	return strings.Join(strings.Fields(value), " ")
}

func splitQuery(value string) []string {
	parts := strings.Fields(strings.ToLower(strings.TrimSpace(value)))
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(part, `"'`)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func matchesAll(terms []string, text string) bool {
	for _, term := range terms {
		if !strings.Contains(text, term) {
			return false
		}
	}
	return true
}

func stringConfig(config map[string]any, key, fallback string) string {
	if value, ok := config[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func topicsFromConfig(config map[string]any) []string {
	value, ok := config["topics"]
	if !ok {
		return []string{}
	}
	items := make([]string, 0)
	switch typed := value.(type) {
	case []string:
		items = append(items, typed...)
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok {
				items = append(items, text)
			}
		}
	case string:
		items = append(items, strings.Split(typed, ",")...)
	}
	out := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		item = strings.TrimSpace(strings.ToLower(item))
		item = strings.ReplaceAll(item, "_", "-")
		item = strings.ReplaceAll(item, " ", "-")
		if item != "" && !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	return out
}
