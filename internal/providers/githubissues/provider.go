package githubissues

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/savisaluwadana/DevRelOS/internal/connectors"
	"github.com/savisaluwadana/DevRelOS/internal/providers/githubauth"
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

func (p *Provider) ID() string { return "github.issues" }

func (p *Provider) Capabilities() []connectors.Capability {
	return []connectors.Capability{connectors.CapabilityTimeline}
}

func (p *Provider) ValidateConfig(config map[string]any) error {
	repository, _ := config["repository"].(string)
	parts := strings.Split(strings.Trim(strings.TrimSpace(repository), "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return errors.New("repository must use owner/repo format")
	}
	if state, _ := config["state"].(string); state != "" && state != "open" && state != "closed" && state != "all" {
		return errors.New("state must be open, closed or all")
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
		Notes:             "Uses GitHub's repository Issues REST endpoint. Credentials may be hydrated from encrypted workspace secrets or an environment-variable reference. Issue content remains source-owned; DevRelOS stores normalized evidence and canonical links by default.",
	}
}

type issue struct {
	Number      int        `json:"number"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	HTMLURL     string     `json:"html_url"`
	State       string     `json:"state"`
	Comments    int        `json:"comments"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	User        user       `json:"user"`
	Labels      []label    `json:"labels"`
	Reactions   reactions  `json:"reactions"`
	PullRequest *struct{}  `json:"pull_request"`
}

type user struct {
	Login   string `json:"login"`
	HTMLURL string `json:"html_url"`
}

type label struct {
	Name string `json:"name"`
}

type reactions struct {
	TotalCount int `json:"total_count"`
}

func (p *Provider) Fetch(ctx context.Context, config map[string]any, request connectors.FetchRequest) (connectors.FetchResult, error) {
	if err := p.ValidateConfig(config); err != nil {
		return connectors.FetchResult{}, err
	}
	repository := strings.Trim(strings.TrimSpace(config["repository"].(string)), "/")
	limit := request.PageLimit
	if limit <= 0 {
		limit = 50
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

	endpoint, err := url.Parse(strings.TrimRight(p.baseURL, "/") + "/repos/" + repository + "/issues")
	if err != nil {
		return connectors.FetchResult{}, err
	}
	params := endpoint.Query()
	state, _ := config["state"].(string)
	if state == "" {
		state = "open"
	}
	params.Set("state", state)
	params.Set("per_page", strconv.Itoa(limit))
	params.Set("page", strconv.Itoa(page))
	params.Set("sort", "updated")
	params.Set("direction", "desc")
	if labels, _ := config["labels"].(string); strings.TrimSpace(labels) != "" {
		params.Set("labels", strings.TrimSpace(labels))
	}
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
		return connectors.FetchResult{}, fmt.Errorf("github issues returned %s", resp.Status)
	}

	var payload []issue
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return connectors.FetchResult{}, err
	}

	query, _ := config["query"].(string)
	query = strings.ToLower(strings.TrimSpace(query))
	includePRs, _ := config["include_pull_requests"].(bool)
	baseTopics := topicsFromConfig(config)
	result := connectors.FetchResult{Records: make([]connectors.RawRecord, 0, len(payload)), RequestsMade: 1}
	if len(payload) == limit {
		result.NextCursor = strconv.Itoa(page + 1)
	}
	for _, item := range payload {
		if item.PullRequest != nil && !includePRs {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(item.Title+"\n"+item.Body), query) {
			continue
		}
		topics := append([]string{}, baseTopics...)
		for _, issueLabel := range item.Labels {
			topic := normalizeTopic(issueLabel.Name)
			if topic != "" && !contains(topics, topic) {
				topics = append(topics, topic)
			}
		}
		engagement := item.Comments*3 + item.Reactions.TotalCount
		if engagement > 1000 {
			engagement = 1000
		}
		timestamp := item.UpdatedAt
		result.Records = append(result.Records, connectors.RawRecord{
			ExternalID:      fmt.Sprintf("%s#%d", repository, item.Number),
			CanonicalURL:    item.HTMLURL,
			SourceTimestamp: &timestamp,
			Payload: map[string]any{
				"repository": repository,
				"number":     item.Number,
				"state":      item.State,
				"comments":   item.Comments,
				"reactions":  item.Reactions.TotalCount,
				"created_at": item.CreatedAt,
			},
			Normalized: connectors.NormalizedRecord{
				Kind:            "signal",
				Title:           item.Title,
				Body:            item.Body,
				AuthorHandle:    item.User.Login,
				AuthorName:      item.User.Login,
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
	for _, item := range items {
		topic := normalizeTopic(item)
		if topic != "" && !contains(out, topic) {
			out = append(out, topic)
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

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
