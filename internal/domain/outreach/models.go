package outreach

import "time"

type Contact struct {
	ID               string    `json:"id"`
	WorkspaceID      string    `json:"workspaceId"`
	Name             string    `json:"name"`
	Role             string    `json:"role"`
	Email            string    `json:"email"`
	PublicProfileURL string    `json:"publicProfileUrl"`
	SourceURL        string    `json:"sourceUrl"`
	DoNotContact     bool      `json:"doNotContact"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type Relationship struct {
	ID             string     `json:"id"`
	ProjectID      string     `json:"projectId"`
	CommunityID    string     `json:"communityId"`
	CommunityName  string     `json:"communityName"`
	ContactID      string     `json:"contactId"`
	ContactName    string     `json:"contactName"`
	Stage          string     `json:"stage"`
	Strength       int        `json:"strength"`
	LastTouchAt    *time.Time `json:"lastTouchAt,omitempty"`
	NextFollowUpAt *time.Time `json:"nextFollowUpAt,omitempty"`
	Notes          string     `json:"notes"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type Touchpoint struct {
	ID             string    `json:"id"`
	RelationshipID string    `json:"relationshipId"`
	Channel        string    `json:"channel"`
	Direction      string    `json:"direction"`
	Summary        string    `json:"summary"`
	OccurredAt     time.Time `json:"occurredAt"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Outreach struct {
	ID            string     `json:"id"`
	ProjectID     string     `json:"projectId"`
	CommunityID   string     `json:"communityId"`
	CommunityName string     `json:"communityName"`
	ContactID     string     `json:"contactId"`
	ContactName   string     `json:"contactName"`
	TalkID        string     `json:"talkId"`
	TalkTitle     string     `json:"talkTitle"`
	Channel       string     `json:"channel"`
	Subject       string     `json:"subject"`
	Body          string     `json:"body"`
	Rationale     string     `json:"rationale"`
	Status        string     `json:"status"`
	ApprovedAt    *time.Time `json:"approvedAt,omitempty"`
	SentAt        *time.Time `json:"sentAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type Delivery struct {
	ID            string     `json:"id"`
	OutreachID    string     `json:"outreachId"`
	ProjectID     string     `json:"projectId"`
	Transport     string     `json:"transport"`
	Status        string     `json:"status"`
	AttemptCount  int        `json:"attemptCount"`
	MessageID     string     `json:"messageId"`
	LastError     string     `json:"lastError"`
	RecipientName string     `json:"recipientName"`
	RecipientEmail string    `json:"recipientEmail"`
	Subject       string     `json:"subject"`
	Body          string     `json:"body"`
	QueuedAt      time.Time  `json:"queuedAt"`
	NextAttemptAt time.Time  `json:"nextAttemptAt"`
	SentAt        *time.Time `json:"sentAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}
