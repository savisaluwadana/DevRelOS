package campaigns

import "time"

type Campaign struct {
	ID        string         `json:"id"`
	ProjectID string         `json:"projectId"`
	Name      string         `json:"name"`
	Objective string         `json:"objective"`
	Status    string         `json:"status"`
	StartsAt  *time.Time     `json:"startsAt,omitempty"`
	EndsAt    *time.Time     `json:"endsAt,omitempty"`
	BudgetUSD float64        `json:"budgetUsd"`
	Target    map[string]any `json:"target"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

type Item struct {
	ID         string         `json:"id"`
	CampaignID string         `json:"campaignId"`
	EntityType string         `json:"entityType"`
	EntityID   string         `json:"entityId"`
	Channel    string         `json:"channel"`
	CostUSD    float64        `json:"costUsd"`
	Metadata   map[string]any `json:"metadata"`
	CreatedAt  time.Time      `json:"createdAt"`
}

type Metric struct {
	ID          string         `json:"id"`
	CampaignID  string         `json:"campaignId"`
	MetricKey   string         `json:"metricKey"`
	MetricValue float64        `json:"metricValue"`
	Source      string         `json:"source"`
	ObservedAt  time.Time      `json:"observedAt"`
	Metadata    map[string]any `json:"metadata"`
	CreatedAt   time.Time      `json:"createdAt"`
}

type Report struct {
	Campaign           Campaign           `json:"campaign"`
	Items              []Item             `json:"items"`
	LinkedByType       map[string]int     `json:"linkedByType"`
	SpendUSD           float64            `json:"spendUsd"`
	BudgetRemainingUSD float64            `json:"budgetRemainingUsd"`
	PublishedContent   int                `json:"publishedContent"`
	CompletedWork      int                `json:"completedWork"`
	OutreachSent       int                `json:"outreachSent"`
	OutreachReplies    int                `json:"outreachReplies"`
	ReplyRate          float64            `json:"replyRate"`
	Submissions        int                `json:"submissions"`
	AcceptedTalks      int                `json:"acceptedTalks"`
	AcceptanceRate     float64            `json:"acceptanceRate"`
	FeedbackShipped    int                `json:"feedbackShipped"`
	OutcomeScore       int                `json:"outcomeScore"`
	Metrics            map[string]float64 `json:"metrics"`
	RecentMetrics      []Metric           `json:"recentMetrics"`
}

type RelationshipRadarItem struct {
	RelationshipID string     `json:"relationshipId"`
	ProjectID      string     `json:"projectId"`
	CommunityID    string     `json:"communityId,omitempty"`
	ContactID      string     `json:"contactId,omitempty"`
	Name           string     `json:"name"`
	Kind           string     `json:"kind"`
	Stage          string     `json:"stage"`
	Strength       int        `json:"strength"`
	LastTouchAt    *time.Time `json:"lastTouchAt,omitempty"`
	NextFollowUpAt *time.Time `json:"nextFollowUpAt,omitempty"`
	DaysSinceTouch *int       `json:"daysSinceTouch,omitempty"`
	Health         string     `json:"health"`
	RiskScore      int        `json:"riskScore"`
	Recommended    string     `json:"recommendedAction"`
}
