package events

import "time"

type Event struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	WebsiteURL  string    `json:"websiteUrl"`
	City        string    `json:"city"`
	Country     string    `json:"country"`
	Timezone    string    `json:"timezone"`
	StartsAt    time.Time `json:"startsAt"`
	EndsAt      time.Time `json:"endsAt"`
	EventType   string    `json:"eventType"`
	Topics      []string  `json:"topics"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type CFP struct {
	ID            string         `json:"id"`
	EventID       string         `json:"eventId"`
	EventName     string         `json:"eventName,omitempty"`
	Name          string         `json:"name"`
	SubmissionURL string         `json:"submissionUrl"`
	OpensAt       *time.Time     `json:"opensAt,omitempty"`
	ClosesAt      *time.Time     `json:"closesAt,omitempty"`
	Tracks        []string       `json:"tracks"`
	Requirements  string         `json:"requirements"`
	Status        string         `json:"status"`
	FitScore      *int           `json:"fitScore,omitempty"`
	ScoreReason   map[string]any `json:"scoreReason"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

type Talk struct {
	ID              string    `json:"id"`
	ProjectID       string    `json:"projectId"`
	Title           string    `json:"title"`
	Abstract        string    `json:"abstract"`
	Description     string    `json:"description"`
	Level           string    `json:"level"`
	DurationMinutes int       `json:"durationMinutes"`
	Topics          []string  `json:"topics"`
	DemoURL         string    `json:"demoUrl"`
	SlidesURL       string    `json:"slidesUrl"`
	RecordingURL    string    `json:"recordingUrl"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type Submission struct {
	ID               string     `json:"id"`
	CFPID            string     `json:"cfpId"`
	TalkID           string     `json:"talkId"`
	EventName        string     `json:"eventName,omitempty"`
	TalkTitle        string     `json:"talkTitle,omitempty"`
	TitleOverride    string     `json:"titleOverride"`
	AbstractOverride string     `json:"abstractOverride"`
	Status           string     `json:"status"`
	SubmittedAt      *time.Time `json:"submittedAt,omitempty"`
	DecisionAt       *time.Time `json:"decisionAt,omitempty"`
	Notes            string     `json:"notes"`
	FitScore         *int       `json:"fitScore,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type Community struct {
	ID               string     `json:"id"`
	ProjectID        string     `json:"projectId"`
	Name             string     `json:"name"`
	Platform         string     `json:"platform"`
	ExternalID       string     `json:"externalId"`
	WebsiteURL       string     `json:"websiteUrl"`
	City             string     `json:"city"`
	Country          string     `json:"country"`
	Timezone         string     `json:"timezone"`
	Topics           []string   `json:"topics"`
	MemberCount      *int       `json:"memberCount,omitempty"`
	ActivityScore    *int       `json:"activityScore,omitempty"`
	SpeakingFitScore *int       `json:"speakingFitScore,omitempty"`
	LastEventAt      *time.Time `json:"lastEventAt,omitempty"`
	NextEventAt      *time.Time `json:"nextEventAt,omitempty"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

// EventUpdate carries partial edits to an Event; nil fields are left unchanged.
type EventUpdate struct {
	Name        *string    `json:"name,omitempty"`
	Description *string    `json:"description,omitempty"`
	WebsiteURL  *string    `json:"websiteUrl,omitempty"`
	City        *string    `json:"city,omitempty"`
	Country     *string    `json:"country,omitempty"`
	Timezone    *string    `json:"timezone,omitempty"`
	StartsAt    *time.Time `json:"startsAt,omitempty"`
	EndsAt      *time.Time `json:"endsAt,omitempty"`
	EventType   *string    `json:"eventType,omitempty"`
	Topics      *[]string  `json:"topics,omitempty"`
	Status      *string    `json:"status,omitempty"`
}

// CFPUpdate carries partial edits to a CFP; nil fields are left unchanged.
type CFPUpdate struct {
	Name          *string    `json:"name,omitempty"`
	SubmissionURL *string    `json:"submissionUrl,omitempty"`
	OpensAt       *time.Time `json:"opensAt,omitempty"`
	ClosesAt      *time.Time `json:"closesAt,omitempty"`
	Tracks        *[]string  `json:"tracks,omitempty"`
	Requirements  *string    `json:"requirements,omitempty"`
	Status        *string    `json:"status,omitempty"`
	FitScore      *int       `json:"fitScore,omitempty"`
}

// TalkUpdate carries partial edits to a Talk; nil fields are left unchanged.
type TalkUpdate struct {
	Title           *string   `json:"title,omitempty"`
	Abstract        *string   `json:"abstract,omitempty"`
	Description     *string   `json:"description,omitempty"`
	Level           *string   `json:"level,omitempty"`
	DurationMinutes *int      `json:"durationMinutes,omitempty"`
	Topics          *[]string `json:"topics,omitempty"`
	DemoURL         *string   `json:"demoUrl,omitempty"`
	SlidesURL       *string   `json:"slidesUrl,omitempty"`
	RecordingURL    *string   `json:"recordingUrl,omitempty"`
	Status          *string   `json:"status,omitempty"`
}

// SubmissionUpdate carries partial edits to a Submission; nil fields are left
// unchanged. Status keeps flowing through the same field so the existing
// submitted_at/decision_at side effects still apply.
type SubmissionUpdate struct {
	TitleOverride    *string `json:"titleOverride,omitempty"`
	AbstractOverride *string `json:"abstractOverride,omitempty"`
	Status           *string `json:"status,omitempty"`
	Notes            *string `json:"notes,omitempty"`
	FitScore         *int    `json:"fitScore,omitempty"`
}

// CommunityUpdate carries partial edits to a Community; nil fields are left
// unchanged.
type CommunityUpdate struct {
	Name             *string    `json:"name,omitempty"`
	Platform         *string    `json:"platform,omitempty"`
	ExternalID       *string    `json:"externalId,omitempty"`
	WebsiteURL       *string    `json:"websiteUrl,omitempty"`
	City             *string    `json:"city,omitempty"`
	Country          *string    `json:"country,omitempty"`
	Timezone         *string    `json:"timezone,omitempty"`
	Topics           *[]string  `json:"topics,omitempty"`
	MemberCount      *int       `json:"memberCount,omitempty"`
	ActivityScore    *int       `json:"activityScore,omitempty"`
	SpeakingFitScore *int       `json:"speakingFitScore,omitempty"`
	LastEventAt      *time.Time `json:"lastEventAt,omitempty"`
	NextEventAt      *time.Time `json:"nextEventAt,omitempty"`
	Status           *string    `json:"status,omitempty"`
}

type Dashboard struct {
	OpenCFPs               int         `json:"openCfps"`
	ClosingSoon            int         `json:"closingSoon"`
	AcceptedTalks          int         `json:"acceptedTalks"`
	SubmissionsInFlight    int         `json:"submissionsInFlight"`
	HighFitCFPs            []CFP       `json:"highFitCfps"`
	UpcomingEvents         []Event     `json:"upcomingEvents"`
	CommunityOpportunities []Community `json:"communityOpportunities"`
}
