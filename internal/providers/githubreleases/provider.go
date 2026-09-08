package githubreleases

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/savisaluwadana/DevRelOS/internal/connectors"
	"github.com/savisaluwadana/DevRelOS/internal/providers/githubauth"
	"github.com/savisaluwadana/DevRelOS/internal/providers/safehttp"
)

const defaultBaseURL = "https://api.github.com"
const apiVersion = "2026-03-10"

type Provider struct {
	client  *http.Client
	baseURL string
}

func New() *Provider {
	return &Provider{client: &http.Client{Timeout: 20 * time.Second}, baseURL: defaultBaseURL}
}

func (p *Provider) ID() string { return "github.releases" }

func (p *Provider) Capabilities() []connectors.Capability {
	return []connectors.Capability{connectors.CapabilityTimeline}
}

func (p *Provider) ValidateConfig(config map[string]any) error {
	repository, _ := config["repository"].(string)
	if _, err := githubauth.ValidateRepository(repository); err != nil {
		return err
	}
	return nil
}

func (p *Provider) Policy(config map[string]any) connectors.Policy {
	authenticated := githubauth.HasToken(config)
	rpm := 1
	daily := 1200
	if authenticated {
		rpm = 30
		daily = 5000
	}
	return connectors.Policy{
		RequestsPerMinute: rpm,
		DailyRequestLimit: daily,
		MonthlyBudgetUSD:  0,
		StoreRawPayload:   false,
		CommercialUseOK:   true,
		Notes:             "Uses GitHub's repository Releases REST endpoint. Credentials may be hydrated from encrypted workspace secrets or, for backwards compatibility, an environment-variable reference. Release notes remain source-owned and are retained as normalized evidence with canonical links.",
	}
}

type release struct {
	ID          int64      `json:"id"`
	TagName     string     `json:"tag_name"`
	Name        string     `json:"name"`
	Body        string     `json:"body"`
	HTMLURL     string     `json:"html_url"`
	Draft       bool       `json:"draft"`
	Prerelease  bool       `json:"prerelease"`
	CreatedAt   time.Time  `json:"created_at"`
	PublishedAt *time.Time `json:"published_at"`
	Author      user       `json:"author"`
	Assets      []asset    `json:"assets"`
}

type user struct {
	Login string `json:"login"`
}

type asset struct {
	DownloadCount int `json:"download_count"`
}

func (p *Provider) Fetch(ctx context.Context, config map[string]any, request connectors.FetchRequest) (connectors.FetchResult, error) {
	if err := p.ValidateConfig(config); err != nil {
		return connectors.FetchResult{}, err
	}
	repository := strings.Trim(strings.TrimSpace(config["repository"].(string)), "/")
	limit := request.PageLimit
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	page := 1
	if request.Cursor != "" {
		if parsed, err := strconv.Atoi(request.Cursor); err == nil && parsed > 0 {
			page = parsed
		}
	}

	endpoint, err := url.Parse(strings.TrimRight(p.baseURL, "/") + "/repos/" + repository + "/releases")
	if err != nil {
		return connectors.FetchResult{}, err
	}
	params := endpoint.Query()
	params.Set("per_page", strconv.Itoa(limit))
	params.Set("page", strconv.Itoa(page))
	endpoint.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return connectors.FetchResult{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	req.Header.Set("User-Agent", "DevRelOS/1.0 (+https://github.com/savisaluwadana/DevRelOS)")
	if token := githubauth.Token(config); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return connectors.FetchResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return connectors.FetchResult{}, fmt.Errorf("github releases returned %s", resp.Status)
	}

	var payload []release
	if err := safehttp.DecodeJSON(resp.Body, &payload); err != nil {
		return connectors.FetchResult{}, err
	}

	query := strings.ToLower(strings.TrimSpace(stringConfig(config, "query", "")))
	includePrereleases := boolConfig(config, "include_prereleases")
	includeDrafts := boolConfig(config, "include_drafts")
	topics := topicsFromConfig(config)
	result := connectors.FetchResult{Records: make([]connectors.RawRecord, 0, len(payload)), RequestsMade: 1}
	if len(payload) == limit {
		result.NextCursor = strconv.Itoa(page + 1)
	}
	for _, item := range payload {
		if item.Draft && !includeDrafts {
			continue
		}
		if item.Prerelease && !includePrereleases {
			continue
		}
		title := strings.TrimSpace(item.Name)
		if title == "" {
			title = strings.TrimSpace(item.TagName)
		}
		if query != "" && !strings.Contains(strings.ToLower(title+"\n"+item.TagName+"\n"+item.Body), query) {
			continue
		}
		engagement := 0
		for _, releaseAsset := range item.Assets {
			engagement += releaseAsset.DownloadCount
		}
		if engagement > 1000 {
			engagement = 1000
		}
		timestamp := item.CreatedAt
		if item.PublishedAt != nil {
			timestamp = item.PublishedAt.UTC()
		}
		result.Records = append(result.Records, connectors.RawRecord{
			ExternalID:      fmt.Sprintf("%s:release:%d", repository, item.ID),
			CanonicalURL:    item.HTMLURL,
			SourceTimestamp: &timestamp,
			Payload: map[string]any{
				"repository":      repository,
				"release_id":      item.ID,
				"tag_name":        item.TagName,
				"draft":           item.Draft,
				"prerelease":      item.Prerelease,
				"asset_downloads": engagement,
			},
			Normalized: connectors.NormalizedRecord{
				Kind:            "signal",
				Shape:           connectors.SourceShapeAnnouncement,
				Title:           title,
				Body:            item.Body,
				AuthorHandle:    item.Author.Login,
				AuthorName:      item.Author.Login,
				Topics:          topics,
				EngagementScore: engagement,
			},
		})
	}
	return result, nil
}

func stringConfig(config map[string]any, key, fallback string) string {
	if value, ok := config[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func boolConfig(config map[string]any, key string) bool {
	value, _ := config[key].(bool)
	return value
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
		item = normalizeTopic(item)
		if item != "" && !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	return out
}

func normalizeTopic(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}
