package signals

import "time"

type Signal struct {
	ID              string     `json:"id"`
	ProjectID       string     `json:"projectId"`
	SourceRecordID  string     `json:"sourceRecordId"`
	Provider        string     `json:"provider"`
	ExternalID      string     `json:"externalId"`
	CanonicalURL    string     `json:"canonicalUrl"`
	AuthorHandle    string     `json:"authorHandle"`
	AuthorName      string     `json:"authorName"`
	Title           string     `json:"title"`
	Body            string     `json:"body"`
	OccurredAt      *time.Time `json:"occurredAt,omitempty"`
	Topics          []string   `json:"topics"`
	EngagementScore int        `json:"engagementScore"`
	RelevanceScore  *int       `json:"relevanceScore,omitempty"`
	Status          string     `json:"status"`
	// SourceShape is whether the author was reporting a problem or announcing
	// something. Only reports become pain-point evidence.
	SourceShape string    `json:"sourceShape"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type PainPoint struct {
	ID            string     `json:"id"`
	ProjectID     string     `json:"projectId"`
	Key           string     `json:"key"`
	Title         string     `json:"title"`
	Summary       string     `json:"summary"`
	Persona       string     `json:"persona"`
	Severity      int        `json:"severity"`
	TrendScore    int        `json:"trendScore"`
	EvidenceCount int        `json:"evidenceCount"`
	Topics        []string   `json:"topics"`
	Status        string     `json:"status"`
	FirstSeenAt   *time.Time `json:"firstSeenAt,omitempty"`
	LastSeenAt    *time.Time `json:"lastSeenAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}
