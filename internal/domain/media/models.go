package media

import "time"

type Asset struct {
	ID                 string         `json:"id"`
	ProjectID          string         `json:"projectId"`
	ContentAssetID     string         `json:"contentAssetId"`
	Title              string         `json:"title"`
	SourcePath         string         `json:"sourcePath"`
	SourceURL          string         `json:"sourceUrl"`
	MediaType          string         `json:"mediaType"`
	DurationMS         int64          `json:"durationMs"`
	Status             string         `json:"status"`
	TranscriptText     string         `json:"transcriptText"`
	TranscriptLanguage string         `json:"transcriptLanguage"`
	TranscriptSegments []Segment      `json:"transcriptSegments"`
	Metadata           map[string]any `json:"metadata"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
}

type Segment struct {
	StartMS int64  `json:"startMs"`
	EndMS   int64  `json:"endMs"`
	Text    string `json:"text"`
}

type Clip struct {
	ID             string         `json:"id"`
	ProjectID      string         `json:"projectId"`
	MediaAssetID   string         `json:"mediaAssetId"`
	ContentAssetID string         `json:"contentAssetId"`
	Title          string         `json:"title"`
	StartMS        int64          `json:"startMs"`
	EndMS          int64          `json:"endMs"`
	AspectRatio    string         `json:"aspectRatio"`
	Score          int            `json:"score"`
	Rationale      string         `json:"rationale"`
	CaptionText    string         `json:"captionText"`
	Status         string         `json:"status"`
	OutputPath     string         `json:"outputPath"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

type Job struct {
	ID           string         `json:"id"`
	ProjectID    string         `json:"projectId"`
	MediaAssetID string         `json:"mediaAssetId"`
	ClipID       string         `json:"clipId"`
	JobType      string         `json:"jobType"`
	Status       string         `json:"status"`
	Attempts     int            `json:"attempts"`
	Error        string         `json:"error"`
	Payload      map[string]any `json:"payload"`
	StartedAt    *time.Time     `json:"startedAt,omitempty"`
	FinishedAt   *time.Time     `json:"finishedAt,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

type TranscriptUpdate struct {
	Language string    `json:"language"`
	Text     string    `json:"text"`
	Segments []Segment `json:"segments"`
}
