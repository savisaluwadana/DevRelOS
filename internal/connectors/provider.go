package connectors

import (
	"context"
	"time"
)

type Capability string

const (
	CapabilitySearch             Capability = "search"
	CapabilityStream             Capability = "stream"
	CapabilityTimeline           Capability = "timeline"
	CapabilityEventFeed          Capability = "event_feed"
	CapabilityCFPFeed            Capability = "cfp_feed"
	CapabilityCommunityDirectory Capability = "community_directory"
	CapabilityPublishing         Capability = "publishing"
	CapabilityAnalytics          Capability = "analytics"
	CapabilityWebhooks           Capability = "webhooks"
)

type Policy struct {
	RequestsPerMinute int     `json:"requestsPerMinute"`
	DailyRequestLimit int     `json:"dailyRequestLimit"`
	MonthlyBudgetUSD  float64 `json:"monthlyBudgetUsd"`
	StoreRawPayload   bool    `json:"storeRawPayload"`
	CommercialUseOK   bool    `json:"commercialUseOk"`
	Notes             string  `json:"notes"`
}

type FetchRequest struct {
	Cursor    string            `json:"cursor"`
	Since     *time.Time        `json:"since,omitempty"`
	Until     *time.Time        `json:"until,omitempty"`
	Filters   map[string]string `json:"filters"`
	PageLimit int               `json:"pageLimit"`
}

type RawRecord struct {
	ExternalID      string         `json:"externalId"`
	CanonicalURL    string         `json:"canonicalUrl"`
	SourceTimestamp *time.Time     `json:"sourceTimestamp,omitempty"`
	Payload         map[string]any `json:"payload"`
}

type FetchResult struct {
	Records      []RawRecord `json:"records"`
	NextCursor   string      `json:"nextCursor"`
	RequestsMade int         `json:"requestsMade"`
	CostUSD      float64     `json:"costUsd"`
	Warnings     []string    `json:"warnings"`
}

type Provider interface {
	ID() string
	Capabilities() []Capability
	ValidateConfig(config map[string]any) error
	Fetch(ctx context.Context, config map[string]any, request FetchRequest) (FetchResult, error)
	Policy(config map[string]any) Policy
}

type Registry struct {
	providers map[string]Provider
}

func NewRegistry(providers ...Provider) *Registry {
	r := &Registry{providers: make(map[string]Provider, len(providers))}
	for _, provider := range providers {
		r.providers[provider.ID()] = provider
	}
	return r
}

func (r *Registry) Get(id string) (Provider, bool) {
	provider, ok := r.providers[id]
	return provider, ok
}

func (r *Registry) Providers() []Provider {
	out := make([]Provider, 0, len(r.providers))
	for _, provider := range r.providers {
		out = append(out, provider)
	}
	return out
}
