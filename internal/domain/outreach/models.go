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

// ContactUpdate carries partial edits to a Contact; nil fields are left
// unchanged.
type ContactUpdate struct {
	Name             *string `json:"name,omitempty"`
	Role             *string `json:"role,omitempty"`
	Email            *string `json:"email,omitempty"`
	PublicProfileURL *string `json:"publicProfileUrl,omitempty"`
	SourceURL        *string `json:"sourceUrl,omitempty"`
	DoNotContact     *bool   `json:"doNotContact,omitempty"`
}

// RelationshipUpdate carries partial edits to a Relationship; nil fields are
// left unchanged. CommunityID/ContactID are identity fields set at creation
// and are not editable afterwards.
type RelationshipUpdate struct {
	Stage          *string    `json:"stage,omitempty"`
	Strength       *int       `json:"strength,omitempty"`
	NextFollowUpAt *time.Time `json:"nextFollowUpAt,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
}

// TouchpointUpdate carries partial edits to a Touchpoint. Touchpoints are an
// append-only activity log, not a fully mutable record: only Summary and
// OccurredAt can be corrected after the fact (e.g. a typo or a wrong
// timestamp) — Channel, Direction and RelationshipID stay fixed once logged.
type TouchpointUpdate struct {
	Summary    *string    `json:"summary,omitempty"`
	OccurredAt *time.Time `json:"occurredAt,omitempty"`
}

// OutreachUpdate carries partial edits to an Outreach; nil fields are left
// unchanged. Status keeps flowing through the same validated transition path
// (validOutreachTransition plus the SMTP delivery gate) the old
// status-only endpoint used — see updateOutreach in services/api/outreach.go.
// Channel is not editable after creation.
type OutreachUpdate struct {
	Subject   *string `json:"subject,omitempty"`
	Body      *string `json:"body,omitempty"`
	Rationale *string `json:"rationale,omitempty"`
	Status    *string `json:"status,omitempty"`
}

type Delivery struct {
	ID             string     `json:"id"`
	OutreachID     string     `json:"outreachId"`
	ProjectID      string     `json:"projectId"`
	Transport      string     `json:"transport"`
	Status         string     `json:"status"`
	AttemptCount   int        `json:"attemptCount"`
	MessageID      string     `json:"messageId"`
	LastError      string     `json:"lastError"`
	RecipientName  string     `json:"recipientName"`
	RecipientEmail string     `json:"recipientEmail"`
	Subject        string     `json:"subject"`
	Body           string     `json:"body"`
	QueuedAt       time.Time  `json:"queuedAt"`
	NextAttemptAt  time.Time  `json:"nextAttemptAt"`
	SentAt         *time.Time `json:"sentAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}
