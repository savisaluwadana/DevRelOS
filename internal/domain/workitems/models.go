package workitems

import "time"

type WorkItem struct {
	ID          string         `json:"id"`
	ProjectID   string         `json:"projectId"`
	SourceType  string         `json:"sourceType"`
	SourceID    string         `json:"sourceId"`
	Kind        string         `json:"kind"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Priority    int            `json:"priority"`
	Status      string         `json:"status"`
	Owner       string         `json:"owner"`
	DueAt       *time.Time     `json:"dueAt,omitempty"`
	Metadata    map[string]any `json:"metadata"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

type PainPointConversion struct {
	Kind  string     `json:"kind"`
	Owner string     `json:"owner"`
	DueAt *time.Time `json:"dueAt,omitempty"`
}
