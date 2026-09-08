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

func (s *Store) ListEvents(ctx context.Context, projectID string, page Page) ([]events.Event, error) {
	limit, limitArgs := page.clause(2)
	args := append([]any{projectID}, limitArgs...)
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, project_id::text, name, description, COALESCE(website_url,''),
		       COALESCE(city,''), COALESCE(country,''), COALESCE(timezone,''),
		       starts_at, ends_at, event_type, topics, status, created_at, updated_at
		FROM events
		WHERE project_id = $1
		ORDER BY starts_at NULLS LAST, created_at DESC`+limit, args...)
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

func (s *Store) ListCFPs(ctx context.Context, projectID string, page Page) ([]events.CFP, error) {
	limit, limitArgs := page.clause(2)
	args := append([]any{projectID}, limitArgs...)
	rows, err := s.pool.Query(ctx, `
		SELECT c.id::text, c.event_id::text, e.name, c.name, COALESCE(c.submission_url,''),
		       c.opens_at, c.closes_at, c.tracks, c.requirements, c.status, c.fit_score,
		       c.score_reason, c.created_at, c.updated_at
		FROM cfps c
		JOIN events e ON e.id = c.event_id
		WHERE e.project_id = $1
		ORDER BY c.closes_at NULLS LAST, c.created_at DESC`+limit, args...)
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

func (s *Store) ListTalks(ctx context.Context, projectID string, page Page) ([]events.Talk, error) {
	limit, limitArgs := page.clause(2)
	args := append([]any{projectID}, limitArgs...)
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, project_id::text, title, abstract, description, level, duration_minutes,
		       topics, COALESCE(demo_url,''), COALESCE(slides_url,''), COALESCE(recording_url,''),
		       status, created_at, updated_at
		FROM talks WHERE project_id = $1 ORDER BY updated_at DESC`+limit, args...)
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

func (s *Store) ListSubmissions(ctx context.Context, projectID string, page Page) ([]events.Submission, error) {
	limit, limitArgs := page.clause(2)
	args := append([]any{projectID}, limitArgs...)
	rows, err := s.pool.Query(ctx, `
		SELECT s.id::text, s.cfp_id::text, s.talk_id::text, e.name, t.title,
		       COALESCE(s.title_override,''), COALESCE(s.abstract_override,''), s.status,
		       s.submitted_at, s.decision_at, s.notes, s.fit_score, s.created_at, s.updated_at
		FROM submissions s
		JOIN cfps c ON c.id = s.cfp_id
		JOIN events e ON e.id = c.event_id
		JOIN talks t ON t.id = s.talk_id
		WHERE e.project_id = $1
		ORDER BY s.updated_at DESC`+limit, args...)
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

func (s *Store) ListCommunities(ctx context.Context, projectID string, page Page) ([]events.Community, error) {
	limit, limitArgs := page.clause(2)
	args := append([]any{projectID}, limitArgs...)
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, project_id::text, name, platform, COALESCE(external_id,''), COALESCE(website_url,''),
		       COALESCE(city,''), COALESCE(country,''), COALESCE(timezone,''), topics, member_count,
		       activity_score, speaking_fit_score, last_event_at, next_event_at, status, created_at, updated_at
		FROM communities WHERE project_id=$1
		ORDER BY speaking_fit_score DESC NULLS LAST, activity_score DESC NULLS LAST, name`+limit, args...)
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

// Dashboard filters in Go and keeps the first five matches per section, so each
// read below uses AllRows: a page limit here would hide qualifying rows rather
// than paginate them.
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

	// HighFitCFPs is filled by the API layer from the live CFP/talk scorer.
	//
	// This used to load every CFP and filter on the stored cfps.fit_score
	// column, which nothing ever computes - it is only persisted if a client
	// happens to supply a value on create. The Command Center's headline
	// "High-fit opportunities" panel was therefore permanently empty even when
	// the scorer rated a CFP highly. The scorer lives in internal/intelligence
	// and must not be imported here, so the handler enriches the dashboard and
	// this query is gone.

	eventsList, err := s.ListEvents(ctx, projectID, AllRows())
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

	communities, err := s.ListCommunities(ctx, projectID, AllRows())
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
