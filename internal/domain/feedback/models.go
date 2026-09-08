package feedback

import "time"

type Item struct {
	ID                string         `json:"id"`
	ProjectID         string         `json:"projectId"`
	SourceType        string         `json:"sourceType"`
	SourceID          string         `json:"sourceId"`
	Title             string         `json:"title"`
	Summary           string         `json:"summary"`
	Persona           string         `json:"persona"`
	Component         string         `json:"component"`
	ImpactScore       int            `json:"impactScore"`
	FrequencyScore    int            `json:"frequencyScore"`
	Status            string         `json:"status"`
	Owner             string         `json:"owner"`
	GitHubRepository  string         `json:"githubRepository"`
	GitHubIssueNumber *int           `json:"githubIssueNumber,omitempty"`
	GitHubIssueURL    string         `json:"githubIssueUrl"`
	GitHubIssueTitle  string         `json:"githubIssueTitle"`
	GitHubIssueBody   string         `json:"githubIssueBody"`
	FollowUpNote      string         `json:"followUpNote"`
	ShippedAt         *time.Time     `json:"shippedAt,omitempty"`
	Metadata          map[string]any `json:"metadata"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
}

type PainPointConversion struct {
	Component        string `json:"component"`
	Owner            string `json:"owner"`
	GitHubRepository string `json:"githubRepository"`
}

type Update struct {
	Title             *string `json:"title,omitempty"`
	Summary           *string `json:"summary,omitempty"`
	Persona           *string `json:"persona,omitempty"`
	Component         *string `json:"component,omitempty"`
	ImpactScore       *int    `json:"impactScore,omitempty"`
	FrequencyScore    *int    `json:"frequencyScore,omitempty"`
	Status            *string `json:"status,omitempty"`
	Owner             *string `json:"owner,omitempty"`
	GitHubRepository  *string `json:"githubRepository,omitempty"`
	GitHubIssueNumber *int    `json:"githubIssueNumber,omitempty"`
	GitHubIssueURL    *string `json:"githubIssueUrl,omitempty"`
	GitHubIssueTitle  *string `json:"githubIssueTitle,omitempty"`
	GitHubIssueBody   *string `json:"githubIssueBody,omitempty"`
	FollowUpNote      *string `json:"followUpNote,omitempty"`
}
