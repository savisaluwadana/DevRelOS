package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	domain "github.com/savisaluwadana/DevRelOS/internal/domain/outreach"
)

func (s *Store) ListContacts(ctx context.Context, workspaceID string) ([]domain.Contact, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, workspace_id::text, name, COALESCE(role,''), COALESCE(email,''),
		       COALESCE(public_profile_url,''), COALESCE(source_url,''), do_not_contact,
		       created_at, updated_at
		FROM contacts WHERE workspace_id=$1
		ORDER BY do_not_contact, name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.Contact, 0)
	for rows.Next() {
		var item domain.Contact
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.Name, &item.Role, &item.Email,
			&item.PublicProfileURL, &item.SourceURL, &item.DoNotContact, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateContact(ctx context.Context, item domain.Contact) (domain.Contact, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO contacts (workspace_id, name, role, email, public_profile_url, source_url, do_not_contact)
		VALUES ($1,$2,NULLIF($3,''),NULLIF($4,''),NULLIF($5,''),NULLIF($6,''),$7)
		RETURNING id::text, created_at, updated_at`,
		item.WorkspaceID, item.Name, item.Role, item.Email, item.PublicProfileURL, item.SourceURL, item.DoNotContact,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) ListRelationships(ctx context.Context, projectID string) ([]domain.Relationship, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id::text, r.project_id::text, COALESCE(r.community_id::text,''), COALESCE(c.name,''),
		       COALESCE(r.contact_id::text,''), COALESCE(ct.name,''), r.stage, r.strength,
		       r.last_touch_at, r.next_follow_up_at, r.notes, r.created_at, r.updated_at
		FROM relationships r
		LEFT JOIN communities c ON c.id=r.community_id
		LEFT JOIN contacts ct ON ct.id=r.contact_id
		WHERE r.project_id=$1
		ORDER BY r.next_follow_up_at NULLS LAST, r.strength DESC, r.updated_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.Relationship, 0)
	for rows.Next() {
		var item domain.Relationship
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.CommunityID, &item.CommunityName,
			&item.ContactID, &item.ContactName, &item.Stage, &item.Strength, &item.LastTouchAt,
			&item.NextFollowUpAt, &item.Notes, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateRelationship(ctx context.Context, item domain.Relationship) (domain.Relationship, error) {
	if item.CommunityID == "" && item.ContactID == "" {
		return item, errors.New("communityId or contactId is required")
	}
	if item.Stage == "" {
		item.Stage = "cold"
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO relationships (project_id, community_id, contact_id, stage, strength, next_follow_up_at, notes)
		VALUES ($1,NULLIF($2,'')::uuid,NULLIF($3,'')::uuid,$4,$5,$6,$7)
		RETURNING id::text, created_at, updated_at`,
		item.ProjectID, item.CommunityID, item.ContactID, item.Stage, item.Strength, item.NextFollowUpAt, item.Notes,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) AddTouchpoint(ctx context.Context, projectID string, item domain.Touchpoint, nextFollowUp *time.Time) (domain.Touchpoint, error) {
	if item.OccurredAt.IsZero() {
		item.OccurredAt = time.Now().UTC()
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return item, err
	}
	defer tx.Rollback(ctx)

	var relationshipExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM relationships WHERE id=$1 AND project_id=$2)`, item.RelationshipID, projectID).Scan(&relationshipExists); err != nil {
		return item, err
	}
	if !relationshipExists {
		return item, pgx.ErrNoRows
	}

	if err := tx.QueryRow(ctx, `
		INSERT INTO touchpoints (relationship_id, channel, direction, summary, occurred_at)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING id::text, created_at`, item.RelationshipID, item.Channel, item.Direction, item.Summary, item.OccurredAt,
	).Scan(&item.ID, &item.CreatedAt); err != nil {
		return item, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE relationships
		SET last_touch_at=$2,
		    next_follow_up_at=$3,
		    strength=LEAST(100, strength + CASE WHEN $4='inbound' THEN 10 ELSE 5 END),
		    stage=CASE WHEN stage='cold' THEN 'warm' ELSE stage END,
		    updated_at=now()
		WHERE id=$1`, item.RelationshipID, item.OccurredAt, nextFollowUp, item.Direction); err != nil {
		return item, err
	}
	return item, tx.Commit(ctx)
}

func (s *Store) ListTouchpoints(ctx context.Context, projectID, relationshipID string, limit int) ([]domain.Touchpoint, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT t.id::text, t.relationship_id::text, t.channel, t.direction, t.summary, t.occurred_at, t.created_at
		FROM touchpoints t
		JOIN relationships r ON r.id=t.relationship_id
		WHERE r.project_id=$1 AND ($2='' OR r.id=$2::uuid)
		ORDER BY t.occurred_at DESC
		LIMIT $3`, projectID, relationshipID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Touchpoint, 0)
	for rows.Next() {
		var item domain.Touchpoint
		if err := rows.Scan(&item.ID, &item.RelationshipID, &item.Channel, &item.Direction, &item.Summary, &item.OccurredAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListOutreach(ctx context.Context, projectID string) ([]domain.Outreach, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT o.id::text, o.project_id::text, COALESCE(o.community_id::text,''), COALESCE(c.name,''),
		       COALESCE(o.contact_id::text,''), COALESCE(ct.name,''), COALESCE(o.talk_id::text,''), COALESCE(t.title,''),
		       o.channel, COALESCE(o.subject,''), o.body, o.rationale, o.status, o.approved_at, o.sent_at,
		       o.created_at, o.updated_at
		FROM outreach o
		LEFT JOIN communities c ON c.id=o.community_id
		LEFT JOIN contacts ct ON ct.id=o.contact_id
		LEFT JOIN talks t ON t.id=o.talk_id
		WHERE o.project_id=$1
		ORDER BY CASE o.status WHEN 'needs_approval' THEN 0 WHEN 'draft' THEN 1 WHEN 'approved' THEN 2 ELSE 3 END,
		         o.updated_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Outreach, 0)
	for rows.Next() {
		var item domain.Outreach
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.CommunityID, &item.CommunityName,
			&item.ContactID, &item.ContactName, &item.TalkID, &item.TalkTitle, &item.Channel,
			&item.Subject, &item.Body, &item.Rationale, &item.Status, &item.ApprovedAt, &item.SentAt,
			&item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateOutreach(ctx context.Context, item domain.Outreach) (domain.Outreach, error) {
	if item.Status == "" {
		item.Status = "draft"
	}
	if item.ContactID != "" {
		var blocked bool
		if err := s.pool.QueryRow(ctx, `SELECT do_not_contact FROM contacts WHERE id=$1`, item.ContactID).Scan(&blocked); err != nil {
			return item, err
		}
		if blocked {
			return item, errors.New("contact is marked do-not-contact")
		}
	}
	if item.CommunityID != "" {
		var status string
		if err := s.pool.QueryRow(ctx, `SELECT status FROM communities WHERE id=$1 AND project_id=$2`, item.CommunityID, item.ProjectID).Scan(&status); err != nil {
			return item, err
		}
		if status == "do_not_contact" {
			return item, errors.New("community is marked do-not-contact")
		}
	}

	err := s.pool.QueryRow(ctx, `
		INSERT INTO outreach (project_id, community_id, contact_id, talk_id, channel, subject, body, rationale, status)
		VALUES ($1,NULLIF($2,'')::uuid,NULLIF($3,'')::uuid,NULLIF($4,'')::uuid,$5,NULLIF($6,''),$7,$8,$9)
		RETURNING id::text, created_at, updated_at`,
		item.ProjectID, item.CommunityID, item.ContactID, item.TalkID, item.Channel,
		item.Subject, item.Body, item.Rationale, item.Status,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) UpdateOutreachStatus(ctx context.Context, projectID, id, status string) error {
	var approvedAt, sentAt any
	now := time.Now().UTC()
	if status == "approved" {
		approvedAt = now
	}
	if status == "sent" {
		sentAt = now
	}
	command, err := s.pool.Exec(ctx, `
		UPDATE outreach
		SET status=$3,
		    approved_at=COALESCE($4, approved_at),
		    sent_at=COALESCE($5, sent_at),
		    updated_at=now()
		WHERE id=$1 AND project_id=$2`, id, projectID, status, approvedAt, sentAt)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
