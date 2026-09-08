package calendar

import "time"

type Item struct {
	ID       string     `json:"id"`
	Kind     string     `json:"kind"`
	Title    string     `json:"title"`
	Subtitle string     `json:"subtitle"`
	StartsAt time.Time  `json:"startsAt"`
	EndsAt   *time.Time `json:"endsAt,omitempty"`
	Status   string     `json:"status"`
	Href     string     `json:"href"`
	Priority int        `json:"priority"`
	SourceID string     `json:"sourceId"`
}
