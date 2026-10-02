package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/savisaluwadana/DevRelOS/internal/domain/events"
)

func (s *Store) UpdateEvent(ctx context.Context, projectID, id string, update events.EventUpdate) (events.Event, error) {
	var e events.Event
	var startsAt, endsAt *time.Time
	err := s.pool.QueryRow(ctx, `
		UPDATE events
		SET name=COALESCE($3,name),
		    description=COALESCE($4,description),
		    website_url=COALESCE(NULLIF($5,''),website_url),
		    city=COALESCE(NULLIF($6,''),city),
		    country=COALESCE(NULLIF($7,''),country),
		    timezone=COALESCE(NULLIF($8,''),timezone),
		    starts_at=COALESCE($9,starts_at),
		    ends_at=COALESCE($10,ends_at),
		    event_type=COALESCE($11,event_type),
		    topics=COALESCE($12::text[],topics),
		    status=COALESCE($13,status),
		    updated_at=now()
		WHERE id=$1 AND project_id=$2
		RETURNING id::text, project_id::text, name, description, COALESCE(website_url,''),
		          COALESCE(city,''), COALESCE(country,''), COALESCE(timezone,''),
		          starts_at, ends_at, event_type, topics, status, created_at, updated_at`,
		id, projectID, update.Name, update.Description, update.WebsiteURL, update.City, update.Country,
		update.Timezone, update.StartsAt, update.EndsAt, update.EventType, update.Topics, update.Status,
	).Scan(&e.ID, &e.ProjectID, &e.Name, &e.Description, &e.WebsiteURL, &e.City, &e.Country, &e.Timezone,
		&startsAt, &endsAt, &e.EventType, &e.Topics, &e.Status, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return e, err
	}
	if startsAt != nil {
		e.StartsAt = *startsAt
	}
	if endsAt != nil {
		e.EndsAt = *endsAt
	}
	return e, nil
}

func (s *Store) DeleteEvent(ctx context.Context, projectID, id string) error {
	var cfpCount int
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM cfps c JOIN events e ON e.id=c.event_id WHERE c.event_id=$1 AND e.project_id=$2`,
		id, projectID).Scan(&cfpCount); err != nil {
		return err
	}
	if cfpCount > 0 {
		return dependentsErr("cannot delete: this event still has CFPs attached")
	}
	if referenced, err := s.campaignItemReferences(ctx, "event", id); err != nil {
		return err
	} else if referenced {
		return dependentsErr("cannot delete: this event is linked to a campaign")
	}
	cmd, err := s.pool.Exec(ctx, `DELETE FROM events WHERE id=$1 AND project_id=$2`, id, projectID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) CreateScopedCFP(ctx context.Context, projectID string, c events.CFP) (events.CFP, error) {
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
	reason, err := json.Marshal(c.ScoreReason)
	if err != nil {
		return c, err
	}

	err = s.pool.QueryRow(ctx, `
		INSERT INTO cfps (event_id, name, submission_url, opens_at, closes_at, tracks, requirements, status, fit_score, score_reason)
		SELECT e.id,$3,NULLIF($4,''),$5,$6,$7,$8,$9,$10,$11::jsonb
		FROM events e
		WHERE e.id=$1 AND e.project_id=$2
		RETURNING id::text, created_at, updated_at`,
		c.EventID, projectID, c.Name, c.SubmissionURL, c.OpensAt, c.ClosesAt, c.Tracks,
		c.Requirements, c.Status, c.FitScore, reason).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (s *Store) UpdateScopedCFP(ctx context.Context, projectID, id string, update events.CFPUpdate) (events.CFP, error) {
	var c events.CFP
	var reason []byte
	err := s.pool.QueryRow(ctx, `
		UPDATE cfps cf
		SET name=COALESCE($3,cf.name),
		    submission_url=COALESCE(NULLIF($4,''),cf.submission_url),
		    opens_at=COALESCE($5,cf.opens_at),
		    closes_at=COALESCE($6,cf.closes_at),
		    tracks=COALESCE($7::text[],cf.tracks),
		    requirements=COALESCE($8,cf.requirements),
		    status=COALESCE($9,cf.status),
		    fit_score=COALESCE($10,cf.fit_score),
		    updated_at=now()
		FROM events e
		WHERE cf.id=$1 AND cf.event_id=e.id AND e.project_id=$2
		RETURNING cf.id::text, cf.event_id::text, e.name, cf.name, COALESCE(cf.submission_url,''),
		          cf.opens_at, cf.closes_at, cf.tracks, cf.requirements, cf.status, cf.fit_score,
		          cf.score_reason, cf.created_at, cf.updated_at`,
		id, projectID, update.Name, update.SubmissionURL, update.OpensAt, update.ClosesAt,
		update.Tracks, update.Requirements, update.Status, update.FitScore,
	).Scan(&c.ID, &c.EventID, &c.EventName, &c.Name, &c.SubmissionURL, &c.OpensAt, &c.ClosesAt,
		&c.Tracks, &c.Requirements, &c.Status, &c.FitScore, &reason, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return c, err
	}
	c.ScoreReason = map[string]any{}
	_ = json.Unmarshal(reason, &c.ScoreReason)
	return c, nil
}

func (s *Store) DeleteScopedCFP(ctx context.Context, projectID, id string) error {
	var subCount int
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM submissions s JOIN cfps c ON c.id=s.cfp_id JOIN events e ON e.id=c.event_id
		WHERE s.cfp_id=$1 AND e.project_id=$2`, id, projectID).Scan(&subCount); err != nil {
		return err
	}
	if subCount > 0 {
		return dependentsErr("cannot delete: this CFP still has submissions attached")
	}
	if referenced, err := s.campaignItemReferences(ctx, "cfp", id); err != nil {
		return err
	} else if referenced {
		return dependentsErr("cannot delete: this CFP is linked to a campaign")
	}
	cmd, err := s.pool.Exec(ctx, `
		DELETE FROM cfps cf USING events e
		WHERE cf.id=$1 AND cf.event_id=e.id AND e.project_id=$2`, id, projectID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) CreateScopedSubmission(ctx context.Context, projectID string, sub events.Submission) (events.Submission, error) {
	if sub.Status == "" {
		sub.Status = "draft"
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO submissions (cfp_id, talk_id, title_override, abstract_override, status, notes, fit_score)
		SELECT c.id,t.id,NULLIF($4,''),NULLIF($5,''),$6,$7,$8
		FROM cfps c
		JOIN events e ON e.id=c.event_id
		JOIN talks t ON t.id=$2 AND t.project_id=$3
		WHERE c.id=$1 AND e.project_id=$3
		RETURNING id::text, created_at, updated_at`,
		sub.CFPID, sub.TalkID, projectID, sub.TitleOverride, sub.AbstractOverride, sub.Status,
		sub.Notes, sub.FitScore).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)
	return sub, err
}

// UpdateScopedSubmission applies a partial edit to a Submission. When Status
// is set, submitted_at/decision_at get the same automatic timestamps the old
// status-only endpoint used to set.
func (s *Store) UpdateScopedSubmission(ctx context.Context, projectID, id string, update events.SubmissionUpdate) (events.Submission, error) {
	var submittedAt, decisionAt any
	if update.Status != nil {
		now := time.Now().UTC()
		if *update.Status == "submitted" {
			submittedAt = now
		}
		if *update.Status == "accepted" || *update.Status == "rejected" {
			decisionAt = now
		}
	}
	var sub events.Submission
	err := s.pool.QueryRow(ctx, `
		UPDATE submissions s
		SET title_override=COALESCE($3,s.title_override),
		    abstract_override=COALESCE($4,s.abstract_override),
		    status=COALESCE($5,s.status),
		    notes=COALESCE($6,s.notes),
		    fit_score=COALESCE($7,s.fit_score),
		    submitted_at=COALESCE($8, s.submitted_at),
		    decision_at=COALESCE($9, s.decision_at),
		    updated_at=now()
		FROM cfps c
		JOIN events e ON e.id=c.event_id
		JOIN talks t ON t.id=s.talk_id
		WHERE s.id=$1 AND s.cfp_id=c.id AND e.project_id=$2
		RETURNING s.id::text, s.cfp_id::text, s.talk_id::text, e.name, t.title,
		          COALESCE(s.title_override,''), COALESCE(s.abstract_override,''), s.status,
		          s.submitted_at, s.decision_at, s.notes, s.fit_score, s.created_at, s.updated_at`,
		id, projectID, update.TitleOverride, update.AbstractOverride, update.Status, update.Notes,
		update.FitScore, submittedAt, decisionAt,
	).Scan(&sub.ID, &sub.CFPID, &sub.TalkID, &sub.EventName, &sub.TalkTitle, &sub.TitleOverride,
		&sub.AbstractOverride, &sub.Status, &sub.SubmittedAt, &sub.DecisionAt, &sub.Notes, &sub.FitScore,
		&sub.CreatedAt, &sub.UpdatedAt)
	return sub, err
}

func (s *Store) DeleteScopedSubmission(ctx context.Context, projectID, id string) error {
	if referenced, err := s.campaignItemReferences(ctx, "submission", id); err != nil {
		return err
	} else if referenced {
		return dependentsErr("cannot delete: this submission is linked to a campaign")
	}
	cmd, err := s.pool.Exec(ctx, `
		DELETE FROM submissions s USING cfps c, events e
		WHERE s.id=$1 AND s.cfp_id=c.id AND c.event_id=e.id AND e.project_id=$2`, id, projectID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
