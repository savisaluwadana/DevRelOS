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

type Dashboard struct {
	OpenCFPs               int         `json:"openCfps"`
	ClosingSoon            int         `json:"closingSoon"`
	AcceptedTalks          int         `json:"acceptedTalks"`
	SubmissionsInFlight    int         `json:"submissionsInFlight"`
	HighFitCFPs            []CFP       `json:"highFitCfps"`
	UpcomingEvents         []Event     `json:"upcomingEvents"`
	CommunityOpportunities []Community `json:"communityOpportunities"`
}
