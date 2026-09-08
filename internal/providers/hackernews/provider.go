package hackernews

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/savisaluwadana/DevRelOS/internal/connectors"
)

const defaultBaseURL = "https://hacker-news.firebaseio.com/v0"

var htmlTag = regexp.MustCompile(`<[^>]+>`)

type Provider struct {
	client  *http.Client
	baseURL string
}

func New() *Provider {
	return &Provider{client: &http.Client{Timeout: 20 * time.Second}, baseURL: defaultBaseURL}
}

func (p *Provider) ID() string { return "hackernews" }

func (p *Provider) Capabilities() []connectors.Capability {
	return []connectors.Capability{connectors.CapabilityTimeline}
}

func (p *Provider) ValidateConfig(config map[string]any) error {
	query, _ := config["query"].(string)
	if strings.TrimSpace(query) == "" {
		return errors.New("query is required")
	}
	feed, _ := config["feed"].(string)
	if feed != "" && feed != "new" && feed != "top" && feed != "best" && feed != "ask" && feed != "show" {
		return errors.New("feed must be new, top, best, ask or show")
	}
	return nil
}

func (p *Provider) Policy(config map[string]any) connectors.Policy {
	return connectors.Policy{
		RequestsPerMinute: 30,
		DailyRequestLimit: 3000,
		MonthlyBudgetUSD:  0,
		StoreRawPayload:   false,
		CommercialUseOK:   true,
		Notes:             "Uses the official public Hacker News Firebase API. The API currently documents no rate limit, but DevRelOS bounds story scans and stores normalized evidence/canonical HN links rather than a redistribution corpus.",
	}
}

type item struct {
	ID          int    `json:"id"`
	Deleted     bool   `json:"deleted"`
	Dead        bool   `json:"dead"`
	Type        string `json:"type"`
	By          string `json:"by"`
	Time        int64  `json:"time"`
	Text        string `json:"text"`
	URL         string `json:"url"`
	Score       int    `json:"score"`
	Title       string `json:"title"`
	Descendants int    `json:"descendants"`
}

func (p *Provider) Fetch(ctx context.Context, config map[string]any, request connectors.FetchRequest) (connectors.FetchResult, error) {
	if err := p.ValidateConfig(config); err != nil {
		return connectors.FetchResult{}, err
	}
	limit := request.PageLimit
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	scanLimit := limit * 3
	if scanLimit < 20 {
		scanLimit = 20
	}
	if scanLimit > 60 {
		scanLimit = 60
	}
	if configured := intConfig(config, "scan_limit", 0); configured > 0 {
		scanLimit = configured
		if scanLimit > 100 {
			scanLimit = 100
		}
	}

	feed := stringConfig(config, "feed", "new")
	feedPath := map[string]string{
		"new": "newstories", "top": "topstories", "best": "beststories",
		"ask": "askstories", "show": "showstories",
	}[feed]
	ids, err := p.fetchIDs(ctx, feedPath)
	if err != nil {
		return connectors.FetchResult{}, err
	}
	if len(ids) > scanLimit {
		ids = ids[:scanLimit]
	}

	queryTerms := splitQuery(stringConfig(config, "query", ""))
	topics := topicsFromConfig(config)
	result := connectors.FetchResult{Records: make([]connectors.RawRecord, 0, limit), RequestsMade: 1}
	for _, id := range ids {
		if len(result.Records) >= limit {
			break
		}
		story, fetchErr := p.fetchItem(ctx, id)
		result.RequestsMade++
		if fetchErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to fetch HN item %d", id))
			continue
		}
		if story.Deleted || story.Dead || story.Type != "story" {
			continue
		}
		body := cleanHTML(story.Text)
		if !matchesAll(queryTerms, strings.ToLower(story.Title+"\n"+body)) {
			continue
		}
		engagement := story.Score + 2*story.Descendants
		if engagement < 0 {
			engagement = 0
		}
		if engagement > 1000 {
			engagement = 1000
		}
		timestamp := time.Unix(story.Time, 0).UTC()
		result.Records = append(result.Records, connectors.RawRecord{
			ExternalID:      strconv.Itoa(story.ID),
			CanonicalURL:    fmt.Sprintf("https://news.ycombinator.com/item?id=%d", story.ID),
			SourceTimestamp: &timestamp,
			Payload: map[string]any{
				"id":           story.ID,
				"score":        story.Score,
				"descendants":  story.Descendants,
				"external_url": story.URL,
				"feed":         feed,
			},
			Normalized: connectors.NormalizedRecord{
				Kind:            "signal",
				Title:           html.UnescapeString(story.Title),
				Body:            body,
				AuthorHandle:    story.By,
				AuthorName:      story.By,
				Topics:          topics,
				EngagementScore: engagement,
			},
		})
	}
	return result, nil
}

func (p *Provider) fetchIDs(ctx context.Context, feed string) ([]int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(p.baseURL, "/")+"/"+feed+".json", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "DevRelOS/0.4 (+https://github.com/savisaluwadana/DevRelOS)")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("hacker news feed returned %s", resp.Status)
	}
	var ids []int
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func (p *Provider) fetchItem(ctx context.Context, id int) (item, error) {
	var out item
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/item/%d.json", strings.TrimRight(p.baseURL, "/"), id), nil)
	if err != nil {
		return out, err
	}
	req.Header.Set("User-Agent", "DevRelOS/0.4 (+https://github.com/savisaluwadana/DevRelOS)")
	resp, err := p.client.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out, fmt.Errorf("hacker news item returned %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, err
	}
	return out, nil
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

func cleanHTML(value string) string {
	value = html.UnescapeString(value)
	value = htmlTag.ReplaceAllString(value, " ")
	return strings.Join(strings.Fields(value), " ")
}

func stringConfig(config map[string]any, key, fallback string) string {
	if value, ok := config[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func intConfig(config map[string]any, key string, fallback int) int {
	value, ok := config[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return int(parsed)
		}
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
		item = strings.ReplaceAll(item, " ", "-")
		if item != "" && !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	return out
}
