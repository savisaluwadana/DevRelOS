package githubdiscussions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/savisaluwadana/DevRelOS/internal/connectors"
	"github.com/savisaluwadana/DevRelOS/internal/providers/githubauth"
	"github.com/savisaluwadana/DevRelOS/internal/providers/safehttp"
)

const defaultGraphQLURL = "https://api.github.com/graphql"

type Provider struct {
	client  *http.Client
	baseURL string
}

func New() *Provider {
	return &Provider{client: &http.Client{Timeout: 20 * time.Second}, baseURL: defaultGraphQLURL}
}

func (p *Provider) ID() string { return "github.discussions" }

func (p *Provider) Capabilities() []connectors.Capability {
	return []connectors.Capability{connectors.CapabilityTimeline}
}

func (p *Provider) ValidateConfig(config map[string]any) error {
	repository, _ := config["repository"].(string)
	if _, err := githubauth.ValidateRepository(repository); err != nil {
		return err
	}
	if !githubauth.HasToken(config) {
		return errors.New("GitHub Discussions requires an encrypted connector secret or token_env credential")
	}
	return nil
}

func (p *Provider) Policy(config map[string]any) connectors.Policy {
	return connectors.Policy{
		RequestsPerMinute: 20,
		DailyRequestLimit: 5000,
		MonthlyBudgetUSD:  0,
		StoreRawPayload:   false,
		CommercialUseOK:   true,
		Notes:             "Uses GitHub's authenticated GraphQL API for repository Discussions. Credentials may be hydrated from encrypted workspace secrets or an environment-variable reference. The connector stores normalized discussion evidence and canonical links.",
	}
}

const discussionsQuery = `query DevRelOSDiscussions($owner: String!, $name: String!, $first: Int!, $after: String) {
  repository(owner: $owner, name: $name) {
    discussions(first: $first, after: $after, orderBy: {field: UPDATED_AT, direction: DESC}) {
      pageInfo { hasNextPage endCursor }
      nodes {
        number
        title
        bodyText
        url
        createdAt
        updatedAt
        closed
        isAnswered
        upvoteCount
        comments { totalCount }
        category { name }
        author { login }
      }
    }
  }
}`

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type graphQLResponse struct {
	Data struct {
		Repository *struct {
			Discussions struct {
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Nodes []discussion `json:"nodes"`
			} `json:"discussions"`
		} `json:"repository"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type discussion struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	BodyText    string    `json:"bodyText"`
	URL         string    `json:"url"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Closed      bool      `json:"closed"`
	IsAnswered  *bool     `json:"isAnswered"`
	UpvoteCount int       `json:"upvoteCount"`
	Comments    struct {
		TotalCount int `json:"totalCount"`
	} `json:"comments"`
	Category struct {
		Name string `json:"name"`
	} `json:"category"`
	Author *struct {
		Login string `json:"login"`
	} `json:"author"`
}

func (p *Provider) Fetch(ctx context.Context, config map[string]any, request connectors.FetchRequest) (connectors.FetchResult, error) {
	if err := p.ValidateConfig(config); err != nil {
		return connectors.FetchResult{}, err
	}
	repository := strings.Trim(strings.TrimSpace(config["repository"].(string)), "/")
	parts := strings.Split(repository, "/")
	limit := request.PageLimit
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	payload := graphQLRequest{
		Query: discussionsQuery,
		Variables: map[string]any{
			"owner": parts[0],
			"name":  parts[1],
			"first": limit,
			"after": nil,
		},
	}
	if request.Cursor != "" {
		payload.Variables["after"] = request.Cursor
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return connectors.FetchResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(encoded))
	if err != nil {
		return connectors.FetchResult{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "DevRelOS/1.0 (+https://github.com/savisaluwadana/DevRelOS)")
	req.Header.Set("Authorization", "Bearer "+githubauth.Token(config))

	resp, err := p.client.Do(req)
	if err != nil {
		return connectors.FetchResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return connectors.FetchResult{}, fmt.Errorf("github discussions returned %s", resp.Status)
	}
	var response graphQLResponse
	if err := safehttp.DecodeJSON(resp.Body, &response); err != nil {
		return connectors.FetchResult{}, err
	}
	if len(response.Errors) > 0 {
		messages := make([]string, 0, len(response.Errors))
		for _, graphErr := range response.Errors {
			messages = append(messages, graphErr.Message)
		}
		return connectors.FetchResult{}, fmt.Errorf("github discussions graphql error: %s", strings.Join(messages, "; "))
	}
	if response.Data.Repository == nil {
		return connectors.FetchResult{}, errors.New("github repository was not found or discussions are unavailable")
	}

	query := strings.ToLower(strings.TrimSpace(stringConfig(config, "query", "")))
	baseTopics := topicsFromConfig(config)
	result := connectors.FetchResult{Records: make([]connectors.RawRecord, 0, len(response.Data.Repository.Discussions.Nodes)), RequestsMade: 1}
	if response.Data.Repository.Discussions.PageInfo.HasNextPage {
		result.NextCursor = response.Data.Repository.Discussions.PageInfo.EndCursor
	}
	for _, item := range response.Data.Repository.Discussions.Nodes {
		if query != "" && !strings.Contains(strings.ToLower(item.Title+"\n"+item.BodyText), query) {
			continue
		}
		topics := append([]string{}, baseTopics...)
		if category := normalizeTopic(item.Category.Name); category != "" && !contains(topics, category) {
			topics = append(topics, category)
		}
		engagement := item.UpvoteCount + 2*item.Comments.TotalCount
		if engagement > 1000 {
			engagement = 1000
		}
		author := ""
		if item.Author != nil {
			author = item.Author.Login
		}
		timestamp := item.UpdatedAt
		result.Records = append(result.Records, connectors.RawRecord{
			ExternalID:      fmt.Sprintf("%s:discussion:%d", repository, item.Number),
			CanonicalURL:    item.URL,
			SourceTimestamp: &timestamp,
			Payload: map[string]any{
				"repository":  repository,
				"number":      item.Number,
				"closed":      item.Closed,
				"is_answered": item.IsAnswered,
				"upvotes":     item.UpvoteCount,
				"comments":    item.Comments.TotalCount,
				"category":    item.Category.Name,
				"created_at":  item.CreatedAt,
			},
			Normalized: connectors.NormalizedRecord{
				Kind:            "signal",
				Title:           item.Title,
				Body:            item.BodyText,
				AuthorHandle:    author,
				AuthorName:      author,
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

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
