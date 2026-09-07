package developersevents

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/savisaluwadana/DevRelOS/internal/connectors"
)

const defaultFeedURL = "https://developers.events/all-events.json"

type Provider struct {
	client *http.Client
}

func New() *Provider {
	return &Provider{client: &http.Client{Timeout: 20 * time.Second}}
}

func (p *Provider) ID() string { return "developers.events" }

func (p *Provider) Capabilities() []connectors.Capability {
	return []connectors.Capability{connectors.CapabilityEventFeed, connectors.CapabilityCFPFeed}
}

func (p *Provider) ValidateConfig(config map[string]any) error {
	if value, ok := config["feed_url"]; ok {
		if _, ok := value.(string); !ok {
			return errors.New("feed_url must be a string")
		}
	}
	return nil
}

func (p *Provider) Policy(config map[string]any) connectors.Policy {
	return connectors.Policy{
		RequestsPerMinute: 2,
		DailyRequestLimit: 24,
		MonthlyBudgetUSD:  0,
		StoreRawPayload:   false,
		CommercialUseOK:   false,
		Notes:             "developers.events content/data is CC BY-NC 4.0; keep this provider disabled for commercial redistribution unless usage is reviewed and approved.",
	}
}

func (p *Provider) Fetch(ctx context.Context, config map[string]any, request connectors.FetchRequest) (connectors.FetchResult, error) {
	if err := p.ValidateConfig(config); err != nil {
		return connectors.FetchResult{}, err
	}
	feedURL := defaultFeedURL
	if value, ok := config["feed_url"].(string); ok && value != "" {
		feedURL = value
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return connectors.FetchResult{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "DevRelOS/0.1 (+https://github.com/savisaluwadana/DevRelOS)")

	resp, err := p.client.Do(req)
	if err != nil {
		return connectors.FetchResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return connectors.FetchResult{}, fmt.Errorf("developers.events returned %s", resp.Status)
	}

	var payload []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return connectors.FetchResult{}, err
	}

	limit := len(payload)
	if request.PageLimit > 0 && request.PageLimit < limit {
		limit = request.PageLimit
	}

	out := connectors.FetchResult{Records: make([]connectors.RawRecord, 0, limit), RequestsMade: 1}
	for _, item := range payload[:limit] {
		name, _ := item["name"].(string)
		date, _ := item["date"].(string)
		url, _ := item["hyperlink"].(string)
		hash := sha256.Sum256([]byte(name + "|" + date + "|" + url))
		out.Records = append(out.Records, connectors.RawRecord{
			ExternalID:   hex.EncodeToString(hash[:16]),
			CanonicalURL: url,
			Payload:      item,
		})
	}
	return out, nil
}
