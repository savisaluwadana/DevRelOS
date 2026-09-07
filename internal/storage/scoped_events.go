package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/savisaluwadana/DevRelOS/internal/domain/events"
)

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

func (s *Store) UpdateScopedSubmissionStatus(ctx context.Context, projectID, id, status string) error {
	var submittedAt, decisionAt any
	now := time.Now().UTC()
	if status == "submitted" {
		submittedAt = now
	}
	if status == "accepted" || status == "rejected" {
		decisionAt = now
	}
	cmd, err := s.pool.Exec(ctx, `
		UPDATE submissions s
		SET status=$3,
		    submitted_at=COALESCE($4, s.submitted_at),
		    decision_at=COALESCE($5, s.decision_at),
		    updated_at=now()
		FROM cfps c
		JOIN events e ON e.id=c.event_id
		WHERE s.id=$1 AND s.cfp_id=c.id AND e.project_id=$2`, id, projectID, status, submittedAt, decisionAt)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
