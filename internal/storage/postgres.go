package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/savisaluwadana/DevRelOS/internal/domain/events"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) DefaultProjectID(ctx context.Context) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		SELECT p.id::text
		FROM projects p
		JOIN workspaces w ON w.id = p.workspace_id
		WHERE w.slug = 'default' AND p.slug = 'default'
		LIMIT 1`).Scan(&id)
	return id, err
}

func (s *Store) ListEvents(ctx context.Context, projectID string) ([]events.Event, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, project_id::text, name, description, COALESCE(website_url,''),
		       COALESCE(city,''), COALESCE(country,''), COALESCE(timezone,''),
		       starts_at, ends_at, event_type, topics, status, created_at, updated_at
		FROM events
		WHERE project_id = $1
		ORDER BY starts_at NULLS LAST, created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]events.Event, 0)
	for rows.Next() {
		var e events.Event
		var startsAt, endsAt *time.Time
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.Name, &e.Description, &e.WebsiteURL,
			&e.City, &e.Country, &e.Timezone, &startsAt, &endsAt, &e.EventType, &e.Topics,
			&e.Status, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		if startsAt != nil {
			e.StartsAt = *startsAt
		}
		if endsAt != nil {
			e.EndsAt = *endsAt
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) CreateEvent(ctx context.Context, e events.Event) (events.Event, error) {
	if e.EventType == "" {
		e.EventType = "conference"
	}
	if e.Status == "" {
		e.Status = "discovered"
	}
	if e.Topics == nil {
		e.Topics = []string{}
	}

	var startsAt any
	if !e.StartsAt.IsZero() {
		startsAt = e.StartsAt
	}
	var endsAt any
	if !e.EndsAt.IsZero() {
		endsAt = e.EndsAt
	}

	err := s.pool.QueryRow(ctx, `
		INSERT INTO events (project_id, name, description, website_url, city, country, timezone, starts_at, ends_at, event_type, topics, status)
		VALUES ($1,$2,$3,NULLIF($4,''),NULLIF($5,''),NULLIF($6,''),NULLIF($7,''),$8,$9,$10,$11,$12)
		RETURNING id::text, created_at, updated_at`,
		e.ProjectID, e.Name, e.Description, e.WebsiteURL, e.City, e.Country, e.Timezone,
		startsAt, endsAt, e.EventType, e.Topics, e.Status).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	return e, err
}

func (s *Store) ListCFPs(ctx context.Context, projectID string) ([]events.CFP, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id::text, c.event_id::text, e.name, c.name, COALESCE(c.submission_url,''),
		       c.opens_at, c.closes_at, c.tracks, c.requirements, c.status, c.fit_score,
		       c.score_reason, c.created_at, c.updated_at
		FROM cfps c
		JOIN events e ON e.id = c.event_id
		WHERE e.project_id = $1
		ORDER BY c.closes_at NULLS LAST, c.created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]events.CFP, 0)
	for rows.Next() {
		var c events.CFP
		var reason []byte
		if err := rows.Scan(&c.ID, &c.EventID, &c.EventName, &c.Name, &c.SubmissionURL,
			&c.OpensAt, &c.ClosesAt, &c.Tracks, &c.Requirements, &c.Status, &c.FitScore,
			&reason, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.ScoreReason = map[string]any{}
		_ = json.Unmarshal(reason, &c.ScoreReason)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) CreateCFP(ctx context.Context, c events.CFP) (events.CFP, error) {
	if c.Name == "" {
		c.Name = "Main CFP"
	}
	if c.Status == "" {
		c.Status = "open"
	}
	if c.Tracks == nil {
		c.Tracks = []string{}
	}
	if c.ScoreReason == nil {
		c.ScoreReason = map[string]any{}
	}
	reason, _ := json.Marshal(c.ScoreReason)

	err := s.pool.QueryRow(ctx, `
		INSERT INTO cfps (event_id, name, submission_url, opens_at, closes_at, tracks, requirements, status, fit_score, score_reason)
		VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10::jsonb)
		RETURNING id::text, created_at, updated_at`,
		c.EventID, c.Name, c.SubmissionURL, c.OpensAt, c.ClosesAt, c.Tracks,
		c.Requirements, c.Status, c.FitScore, reason).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (s *Store) ListTalks(ctx context.Context, projectID string) ([]events.Talk, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, project_id::text, title, abstract, description, level, duration_minutes,
		       topics, COALESCE(demo_url,''), COALESCE(slides_url,''), COALESCE(recording_url,''),
		       status, created_at, updated_at
		FROM talks WHERE project_id = $1 ORDER BY updated_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]events.Talk, 0)
	for rows.Next() {
		var t events.Talk
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Abstract, &t.Description, &t.Level,
			&t.DurationMinutes, &t.Topics, &t.DemoURL, &t.SlidesURL, &t.RecordingURL,
			&t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) CreateTalk(ctx context.Context, t events.Talk) (events.Talk, error) {
	if t.Level == "" {
		t.Level = "intermediate"
	}
	if t.DurationMinutes == 0 {
		t.DurationMinutes = 30
	}
	if t.Status == "" {
		t.Status = "draft"
	}
	if t.Topics == nil {
		t.Topics = []string{}
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO talks (project_id, title, abstract, description, level, duration_minutes, topics, demo_url, slides_url, recording_url, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''),NULLIF($9,''),NULLIF($10,''),$11)
		RETURNING id::text, created_at, updated_at`,
		t.ProjectID, t.Title, t.Abstract, t.Description, t.Level, t.DurationMinutes, t.Topics,
		t.DemoURL, t.SlidesURL, t.RecordingURL, t.Status).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (s *Store) ListSubmissions(ctx context.Context, projectID string) ([]events.Submission, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT s.id::text, s.cfp_id::text, s.talk_id::text, e.name, t.title,
		       COALESCE(s.title_override,''), COALESCE(s.abstract_override,''), s.status,
		       s.submitted_at, s.decision_at, s.notes, s.fit_score, s.created_at, s.updated_at
		FROM submissions s
		JOIN cfps c ON c.id = s.cfp_id
		JOIN events e ON e.id = c.event_id
		JOIN talks t ON t.id = s.talk_id
		WHERE e.project_id = $1
		ORDER BY s.updated_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]events.Submission, 0)
	for rows.Next() {
		var srow events.Submission
		if err := rows.Scan(&srow.ID, &srow.CFPID, &srow.TalkID, &srow.EventName, &srow.TalkTitle,
			&srow.TitleOverride, &srow.AbstractOverride, &srow.Status, &srow.SubmittedAt, &srow.DecisionAt,
			&srow.Notes, &srow.FitScore, &srow.CreatedAt, &srow.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, srow)
	}
	return out, rows.Err()
}

func (s *Store) CreateSubmission(ctx context.Context, sub events.Submission) (events.Submission, error) {
	if sub.Status == "" {
		sub.Status = "draft"
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO submissions (cfp_id, talk_id, title_override, abstract_override, status, notes, fit_score)
		VALUES ($1,$2,NULLIF($3,''),NULLIF($4,''),$5,$6,$7)
		RETURNING id::text, created_at, updated_at`,
		sub.CFPID, sub.TalkID, sub.TitleOverride, sub.AbstractOverride, sub.Status,
		sub.Notes, sub.FitScore).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)
	return sub, err
}

func (s *Store) UpdateSubmissionStatus(ctx context.Context, id, status string) error {
	var submittedAt, decisionAt any
	now := time.Now().UTC()
	if status == "submitted" {
		submittedAt = now
	}
	if status == "accepted" || status == "rejected" {
		decisionAt = now
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE submissions
		SET status=$2,
		    submitted_at=COALESCE($3, submitted_at),
		    decision_at=COALESCE($4, decision_at),
		    updated_at=now()
		WHERE id=$1`, id, status, submittedAt, decisionAt)
	return err
}

func (s *Store) ListCommunities(ctx context.Context, projectID string) ([]events.Community, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, project_id::text, name, platform, COALESCE(external_id,''), COALESCE(website_url,''),
		       COALESCE(city,''), COALESCE(country,''), COALESCE(timezone,''), topics, member_count,
		       activity_score, speaking_fit_score, last_event_at, next_event_at, status, created_at, updated_at
		FROM communities WHERE project_id=$1
		ORDER BY speaking_fit_score DESC NULLS LAST, activity_score DESC NULLS LAST, name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]events.Community, 0)
	for rows.Next() {
		var c events.Community
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Name, &c.Platform, &c.ExternalID, &c.WebsiteURL,
			&c.City, &c.Country, &c.Timezone, &c.Topics, &c.MemberCount, &c.ActivityScore,
			&c.SpeakingFitScore, &c.LastEventAt, &c.NextEventAt, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) CreateCommunity(ctx context.Context, c events.Community) (events.Community, error) {
	if c.Platform == "" {
		c.Platform = "manual"
	}
	if c.Status == "" {
		c.Status = "discovered"
	}
	if c.Topics == nil {
		c.Topics = []string{}
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO communities (project_id, name, platform, external_id, website_url, city, country, timezone, topics, member_count, activity_score, speaking_fit_score, last_event_at, next_event_at, status)
		VALUES ($1,$2,$3,NULLIF($4,''),NULLIF($5,''),NULLIF($6,''),NULLIF($7,''),NULLIF($8,''),$9,$10,$11,$12,$13,$14,$15)
		RETURNING id::text, created_at, updated_at`,
		c.ProjectID, c.Name, c.Platform, c.ExternalID, c.WebsiteURL, c.City, c.Country, c.Timezone,
		c.Topics, c.MemberCount, c.ActivityScore, c.SpeakingFitScore, c.LastEventAt, c.NextEventAt,
		c.Status).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (s *Store) Dashboard(ctx context.Context, projectID string) (events.Dashboard, error) {
	var d events.Dashboard
	err := s.pool.QueryRow(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE c.status='open'),
		  COUNT(*) FILTER (WHERE c.status='open' AND c.closes_at BETWEEN now() AND now() + interval '7 days'),
		  (SELECT COUNT(*) FROM submissions s JOIN cfps c2 ON c2.id=s.cfp_id JOIN events e2 ON e2.id=c2.event_id WHERE e2.project_id=$1 AND s.status='accepted'),
		  (SELECT COUNT(*) FROM submissions s JOIN cfps c3 ON c3.id=s.cfp_id JOIN events e3 ON e3.id=c3.event_id WHERE e3.project_id=$1 AND s.status IN ('draft','needs_work','ready','submitted'))
		FROM cfps c JOIN events e ON e.id=c.event_id WHERE e.project_id=$1`, projectID).
		Scan(&d.OpenCFPs, &d.ClosingSoon, &d.AcceptedTalks, &d.SubmissionsInFlight)
	if err != nil {
		return d, err
	}

	cfps, err := s.ListCFPs(ctx, projectID)
	if err != nil {
		return d, err
	}
	for _, c := range cfps {
		if c.Status == "open" && c.FitScore != nil && *c.FitScore >= 70 {
			d.HighFitCFPs = append(d.HighFitCFPs, c)
			if len(d.HighFitCFPs) == 5 {
				break
			}
		}
	}

	eventsList, err := s.ListEvents(ctx, projectID)
	if err != nil {
		return d, err
	}
	now := time.Now()
	for _, e := range eventsList {
		if !e.StartsAt.IsZero() && e.StartsAt.After(now) {
			d.UpcomingEvents = append(d.UpcomingEvents, e)
			if len(d.UpcomingEvents) == 5 {
				break
			}
		}
	}

	communities, err := s.ListCommunities(ctx, projectID)
	if err != nil {
		return d, err
	}
	for _, c := range communities {
		if c.SpeakingFitScore != nil && *c.SpeakingFitScore >= 70 && c.Status != "do_not_contact" {
			d.CommunityOpportunities = append(d.CommunityOpportunities, c)
			if len(d.CommunityOpportunities) == 5 {
				break
			}
		}
	}
	return d, nil
}
