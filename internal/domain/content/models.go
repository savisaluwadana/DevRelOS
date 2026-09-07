package content

import "time"

type Asset struct {
	ID           string         `json:"id"`
	ProjectID    string         `json:"projectId"`
	WorkItemID   string         `json:"workItemId"`
	Channel      string         `json:"channel"`
	Format       string         `json:"format"`
	Title        string         `json:"title"`
	Audience     string         `json:"audience"`
	Objective    string         `json:"objective"`
	Brief        string         `json:"brief"`
	Draft        string         `json:"draft"`
	Status       string         `json:"status"`
	Topics       []string       `json:"topics"`
	SourceURL    string         `json:"sourceUrl"`
	PublishedURL string         `json:"publishedUrl"`
	ScheduledAt  *time.Time     `json:"scheduledAt,omitempty"`
	PublishedAt  *time.Time     `json:"publishedAt,omitempty"`
	Metadata     map[string]any `json:"metadata"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

type WorkItemConversion struct {
	Channel  string `json:"channel"`
	Format   string `json:"format"`
	Audience string `json:"audience"`
}

type AssetUpdate struct {
	Title        *string    `json:"title,omitempty"`
	Audience     *string    `json:"audience,omitempty"`
	Objective    *string    `json:"objective,omitempty"`
	Brief        *string    `json:"brief,omitempty"`
	Draft        *string    `json:"draft,omitempty"`
	Status       *string    `json:"status,omitempty"`
	SourceURL    *string    `json:"sourceUrl,omitempty"`
	PublishedURL *string    `json:"publishedUrl,omitempty"`
	ScheduledAt  *time.Time `json:"scheduledAt,omitempty"`
}
