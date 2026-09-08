package ocg

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
)

const defaultBaseURL = "https://ocgroups.dev"

type Provider struct{ client *http.Client }

func New() *Provider           { return &Provider{client: &http.Client{Timeout: 20 * time.Second}} }
func (p *Provider) ID() string { return "ocg" }
func (p *Provider) Capabilities() []connectors.Capability {
	return []connectors.Capability{connectors.CapabilitySearch, connectors.CapabilityCommunityDirectory}
}

func (p *Provider) ValidateConfig(config map[string]any) error {
	if base, ok := config["base_url"].(string); ok && base != "" {
		parsed, err := url.Parse(base)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return errors.New("base_url must be an https URL")
		}
	}
	return nil
}

func (p *Provider) Policy(map[string]any) connectors.Policy {
	return connectors.Policy{
		RequestsPerMinute: 10,
		DailyRequestLimit: 500,
		MonthlyBudgetUSD:  0,
		StoreRawPayload:   false,
		CommercialUseOK:   true,
		Notes:             "Uses OCG's public JSON group-search endpoint. Store normalized group metadata and provenance; keep the provider replaceable if the public contract changes.",
	}
}

type searchResponse struct {
	Groups []groupSummary `json:"groups"`
	Total  int            `json:"total"`
}

type groupSummary struct {
	Active               bool            `json:"active"`
	Category             json.RawMessage `json:"category"`
	CommunityDisplayName string          `json:"community_display_name"`
	CommunityName        string          `json:"community_name"`
	CreatedAt            int64           `json:"created_at"`
	GroupID              string          `json:"group_id"`
	Name                 string          `json:"name"`
	Slug                 string          `json:"slug"`
	SlugPretty           *string         `json:"slug_pretty"`
	City                 *string         `json:"city"`
	CountryName          *string         `json:"country_name"`
	DescriptionShort     *string         `json:"description_short"`
	Region               json.RawMessage `json:"region"`
}

func (p *Provider) Fetch(ctx context.Context, config map[string]any, request connectors.FetchRequest) (connectors.FetchResult, error) {
	if err := p.ValidateConfig(config); err != nil {
		return connectors.FetchResult{}, err
	}
	base := defaultBaseURL
	if value, ok := config["base_url"].(string); ok && value != "" {
		base = strings.TrimRight(value, "/")
	}
	endpoint, err := url.Parse(base + "/explore/groups/search")
	if err != nil {
		return connectors.FetchResult{}, err
	}

	limit := request.PageLimit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	offset := 0
	if request.Cursor != "" {
		if parsed, parseErr := strconv.Atoi(request.Cursor); parseErr == nil && parsed >= 0 {
			offset = parsed
		}
	}

	q := endpoint.Query()
	community := stringValue(config, "community", "cncf")
	if community != "" {
		q.Set("community[0]", community)
	}
	if query := strings.TrimSpace(stringValue(config, "query", "")); query != "" {
		q.Set("ts_query", query)
	}
	if region := strings.TrimSpace(stringValue(config, "region", "")); region != "" {
		q.Set("region[0]", region)
	}
	if category := strings.TrimSpace(stringValue(config, "group_category", "")); category != "" {
		q.Set("group_category[0]", category)
	}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	endpoint.RawQuery = q.Encode()

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
		return connectors.FetchResult{}, fmt.Errorf("ocg returned %s", resp.Status)
	}

	var payload searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return connectors.FetchResult{}, err
	}
	result := connectors.FetchResult{Records: make([]connectors.RawRecord, 0, len(payload.Groups)), RequestsMade: 1}
	if offset+len(payload.Groups) < payload.Total {
		result.NextCursor = strconv.Itoa(offset + len(payload.Groups))
	}

	for _, group := range payload.Groups {
		if !group.Active || group.GroupID == "" || group.Name == "" {
			continue
		}
		slug := group.Slug
		if group.SlugPretty != nil && *group.SlugPretty != "" {
			slug = *group.SlugPretty
		}
		canonical := ""
		if group.CommunityName != "" && slug != "" {
			canonical = base + "/" + url.PathEscape(group.CommunityName) + "/group/" + url.PathEscape(slug)
		}
		var timestamp *time.Time
		if group.CreatedAt > 0 {
			value := time.Unix(group.CreatedAt, 0).UTC()
			timestamp = &value
		}
		topics := append(topicsFromConfig(config), categoryLabels(group.Category)...)
		result.Records = append(result.Records, connectors.RawRecord{
			ExternalID: group.GroupID, CanonicalURL: canonical, SourceTimestamp: timestamp,
			Payload: map[string]any{"community_name": group.CommunityName, "community_display_name": group.CommunityDisplayName},
			Normalized: connectors.NormalizedRecord{
				Kind: "community", Name: group.Name, Platform: "ocg", City: ptrValue(group.City), Country: ptrValue(group.CountryName),
				Body: ptrValue(group.DescriptionShort), Topics: unique(topics),
			},
		})
	}
	return result, nil
}

func stringValue(config map[string]any, key, fallback string) string {
	if value, ok := config[key].(string); ok {
		return value
	}
	return fallback
}
func ptrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func topicsFromConfig(config map[string]any) []string {
	value, ok := config["topics"]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case string:
		return strings.Split(typed, ",")
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}
func categoryLabels(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var text string
	if json.Unmarshal(raw, &text) == nil && text != "" {
		return []string{text}
	}
	var object map[string]any
	if json.Unmarshal(raw, &object) == nil {
		for _, key := range []string{"name", "slug", "normalized_name"} {
			if value, ok := object[key].(string); ok && value != "" {
				return []string{value}
			}
		}
	}
	return nil
}
func unique(items []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(items))
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
