package bluesky

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/savisaluwadana/DevRelOS/internal/connectors"
	"github.com/savisaluwadana/DevRelOS/internal/providers/safehttp"
)

const defaultBaseURL = "https://public.api.bsky.app"

type Provider struct {
	client *http.Client
	// allowUnsafeLocal relaxes destination validation for tests and local
	// fixtures. It must never be set from operator configuration.
	allowUnsafeLocal bool
}

func New() *Provider {
	return &Provider{client: safehttp.NewClient(20*time.Second, "base_url")}
}

func (p *Provider) ID() string { return "bluesky" }

func (p *Provider) Capabilities() []connectors.Capability {
	return []connectors.Capability{connectors.CapabilitySearch}
}

func (p *Provider) ValidateConfig(config map[string]any) error {
	query, _ := config["query"].(string)
	if strings.TrimSpace(query) == "" {
		return errors.New("query is required")
	}
	if baseURL, ok := config["base_url"].(string); ok && baseURL != "" {
		parsed, err := url.Parse(baseURL)
		if err != nil {
			return errors.New("base_url must be a valid URL")
		}
		// base_url is operator-supplied, so it needs the same destination
		// validation the RSS feed URL gets: scheme, credentials, and private or
		// link-local addresses (cloud instance metadata lives on one).
		if err := safehttp.ValidateURL(parsed, "base_url", p.allowUnsafeLocal); err != nil {
			return err
		}
	}
	return nil
}

func (p *Provider) Policy(config map[string]any) connectors.Policy {
	return connectors.Policy{
		RequestsPerMinute: 30,
		DailyRequestLimit: 1000,
		MonthlyBudgetUSD:  0,
		StoreRawPayload:   false,
		CommercialUseOK:   true,
		Notes:             "Public AppView search is unauthenticated. User content remains owned by its authors; DevRelOS stores normalized evidence/provenance and canonical links by default, not a redistribution corpus.",
	}
}

type searchResponse struct {
	Cursor string     `json:"cursor"`
	Posts  []postView `json:"posts"`
}

type postView struct {
	URI         string         `json:"uri"`
	CID         string         `json:"cid"`
	Author      author         `json:"author"`
	Record      map[string]any `json:"record"`
	ReplyCount  int            `json:"replyCount"`
	RepostCount int            `json:"repostCount"`
	LikeCount   int            `json:"likeCount"`
	QuoteCount  int            `json:"quoteCount"`
	IndexedAt   string         `json:"indexedAt"`
}

type author struct {
	DID         string `json:"did"`
	Handle      string `json:"handle"`
	DisplayName string `json:"displayName"`
}

func (p *Provider) Fetch(ctx context.Context, config map[string]any, request connectors.FetchRequest) (connectors.FetchResult, error) {
	if err := p.ValidateConfig(config); err != nil {
		return connectors.FetchResult{}, err
	}
	baseURL := defaultBaseURL
	if configured, ok := config["base_url"].(string); ok && configured != "" {
		baseURL = strings.TrimRight(configured, "/")
	}

	limit := request.PageLimit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	endpoint, err := url.Parse(baseURL + "/xrpc/app.bsky.feed.searchPosts")
	if err != nil {
		return connectors.FetchResult{}, err
	}
	params := endpoint.Query()
	query, _ := config["query"].(string)
	params.Set("q", strings.TrimSpace(query))
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("sort", "latest")
	if request.Cursor != "" {
		params.Set("cursor", request.Cursor)
	}
	endpoint.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return connectors.FetchResult{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "DevRelOS/0.2 (+https://github.com/savisaluwadana/DevRelOS)")

	resp, err := p.client.Do(req)
	if err != nil {
		return connectors.FetchResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return connectors.FetchResult{}, fmt.Errorf("bluesky returned %s", resp.Status)
	}

	var payload searchResponse
	if err := safehttp.DecodeJSON(resp.Body, &payload); err != nil {
		return connectors.FetchResult{}, err
	}

	topics := topicsFromConfig(config)
	result := connectors.FetchResult{
		Records:      make([]connectors.RawRecord, 0, len(payload.Posts)),
		NextCursor:   payload.Cursor,
		RequestsMade: 1,
	}
	for _, post := range payload.Posts {
		text, _ := post.Record["text"].(string)
		createdAt, _ := post.Record["createdAt"].(string)
		timestamp := parseTime(createdAt)
		if timestamp == nil {
			timestamp = parseTime(post.IndexedAt)
		}

		canonicalURL := postURL(post.Author.Handle, post.URI)
		engagement := post.LikeCount + 2*post.RepostCount + 2*post.ReplyCount + 2*post.QuoteCount
		if engagement > 1000 {
			engagement = 1000
		}

		result.Records = append(result.Records, connectors.RawRecord{
			ExternalID:      post.URI,
			CanonicalURL:    canonicalURL,
			SourceTimestamp: timestamp,
			Payload: map[string]any{
				"uri":          post.URI,
				"cid":          post.CID,
				"author_did":   post.Author.DID,
				"reply_count":  post.ReplyCount,
				"repost_count": post.RepostCount,
				"like_count":   post.LikeCount,
				"quote_count":  post.QuoteCount,
			},
			Normalized: connectors.NormalizedRecord{
				Kind:            "signal",
				Body:            text,
				AuthorHandle:    post.Author.Handle,
				AuthorName:      post.Author.DisplayName,
				Topics:          topics,
				EngagementScore: engagement,
			},
		})
	}
	return result, nil
}

func topicsFromConfig(config map[string]any) []string {
	value, ok := config["topics"]
	if !ok {
		return []string{}
	}
	switch typed := value.(type) {
	case []string:
		return cleanTopics(typed)
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				items = append(items, text)
			}
		}
		return cleanTopics(items)
	case string:
		return cleanTopics(strings.Split(typed, ","))
	default:
		return []string{}
	}
}

func cleanTopics(items []string) []string {
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

func postURL(handle, uri string) string {
	parts := strings.Split(uri, "/")
	if handle == "" || len(parts) == 0 {
		return ""
	}
	rkey := parts[len(parts)-1]
	if rkey == "" {
		return ""
	}
	return "https://bsky.app/profile/" + url.PathEscape(handle) + "/post/" + url.PathEscape(rkey)
}

func parseTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil
	}
	return &parsed
}
