package connectors

import "time"

type Connector struct {
	ID              string         `json:"id"`
	WorkspaceID     string         `json:"workspaceId"`
	Provider        string         `json:"provider"`
	Name            string         `json:"name"`
	Enabled         bool           `json:"enabled"`
	Config          map[string]any `json:"config"`
	Policy          map[string]any `json:"policy"`
	SecretID        string         `json:"secretId,omitempty"`
	ScheduleMinutes *int           `json:"scheduleMinutes,omitempty"`
	NextRunAt       *time.Time     `json:"nextRunAt,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

type Run struct {
	ID              string     `json:"id"`
	ConnectorID     string     `json:"connectorId"`
	Status          string     `json:"status"`
	Cursor          string     `json:"cursor"`
	RequestsMade    int        `json:"requestsMade"`
	ItemsFetched    int        `json:"itemsFetched"`
	ItemsCreated    int        `json:"itemsCreated"`
	ItemsUpdated    int        `json:"itemsUpdated"`
	ItemsSkipped    int        `json:"itemsSkipped"`
	ProviderCostUSD *float64   `json:"providerCostUsd,omitempty"`
	Warnings        []string   `json:"warnings"`
	Error           string     `json:"error"`
	StartedAt       *time.Time `json:"startedAt,omitempty"`
	FinishedAt      *time.Time `json:"finishedAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
}

type SourceRecord struct {
	ID              string         `json:"id"`
	WorkspaceID     string         `json:"workspaceId"`
	Provider        string         `json:"provider"`
	ExternalID      string         `json:"externalId"`
	CanonicalURL    string         `json:"canonicalUrl"`
	SourceTimestamp *time.Time     `json:"sourceTimestamp,omitempty"`
	FetchedAt       time.Time      `json:"fetchedAt"`
	ContentHash     string         `json:"contentHash"`
	RawPayload      map[string]any `json:"rawPayload,omitempty"`
	Provenance      map[string]any `json:"provenance"`
}
