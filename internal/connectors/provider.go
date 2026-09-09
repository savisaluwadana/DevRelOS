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

// SourceShape describes whether a provider's records carry a developer
// reporting a problem, or a publisher announcing something.
//
// Pain-point clustering only accepts reports as evidence. Announcement sources
// stay fully useful for Signal Radar and content research; they just must not
// manufacture developer pain, because friction vocabulary ("setup",
// "configure", "complex", "manual") appears throughout ordinary technical
// prose that is describing rather than complaining.
type SourceShape string

const (
	// SourceShapeReport is content where the author has a problem: issues,
	// discussions, forum questions, support tickets.
	SourceShapeReport SourceShape = "report"
	// SourceShapeAnnouncement is content that publishes news: blog feeds,
	// release notes, changelogs, press releases.
	SourceShapeAnnouncement SourceShape = "announcement"
	// SourceShapeUnknown leaves the record eligible for clustering. It is the
	// default so that manually entered signals and older rows behave as before.
	SourceShapeUnknown SourceShape = "unknown"
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

type NormalizedRecord struct {
	Kind             string   `json:"kind"`
	Title            string   `json:"title"`
	Body             string   `json:"body"`
	AuthorHandle     string   `json:"authorHandle"`
	AuthorName       string   `json:"authorName"`
	Name             string   `json:"name"`
	Platform         string   `json:"platform"`
	City             string   `json:"city"`
	Country          string   `json:"country"`
	Topics           []string `json:"topics"`
	EngagementScore  int      `json:"engagementScore"`
	ActivityScore    *int     `json:"activityScore,omitempty"`
	SpeakingFitScore *int     `json:"speakingFitScore,omitempty"`
	// Shape is how pain-point clustering should treat this record.
	Shape SourceShape
}

type RawRecord struct {
	ExternalID      string           `json:"externalId"`
	CanonicalURL    string           `json:"canonicalUrl"`
	SourceTimestamp *time.Time       `json:"sourceTimestamp,omitempty"`
	Payload         map[string]any   `json:"payload"`
	Normalized      NormalizedRecord `json:"normalized"`
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
