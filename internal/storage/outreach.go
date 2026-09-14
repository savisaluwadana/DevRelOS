package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	domain "github.com/savisaluwadana/DevRelOS/internal/domain/outreach"
)

func (s *Store) ListContacts(ctx context.Context, workspaceID string, page Page) ([]domain.Contact, error) {
	limit, limitArgs := page.clause(2)
	args := append([]any{workspaceID}, limitArgs...)
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, workspace_id::text, name, COALESCE(role,''), COALESCE(email,''),
		       COALESCE(public_profile_url,''), COALESCE(source_url,''), do_not_contact,
		       created_at, updated_at
		FROM contacts WHERE workspace_id=$1
		ORDER BY do_not_contact, name`+limit, args...)
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

// UpdateContact applies a partial edit to a Contact. Role/Email/PublicProfileURL/
// SourceURL are nullable columns, so an omitted (nil) field is left alone via
// COALESCE(NULLIF($n,''),col) the same way CreateContact treats them; Name and
// DoNotContact are plain COALESCE($n,col) since they are not nullable columns.
func (s *Store) UpdateContact(ctx context.Context, workspaceID, id string, update domain.ContactUpdate) (domain.Contact, error) {
	var item domain.Contact
	err := s.pool.QueryRow(ctx, `
		UPDATE contacts
		SET name=COALESCE($3,name),
		    role=COALESCE(NULLIF($4,''),role),
		    email=COALESCE(NULLIF($5,''),email),
		    public_profile_url=COALESCE(NULLIF($6,''),public_profile_url),
		    source_url=COALESCE(NULLIF($7,''),source_url),
		    do_not_contact=COALESCE($8,do_not_contact),
		    updated_at=now()
		WHERE id=$1 AND workspace_id=$2
		RETURNING id::text, workspace_id::text, name, COALESCE(role,''), COALESCE(email,''),
		          COALESCE(public_profile_url,''), COALESCE(source_url,''), do_not_contact,
		          created_at, updated_at`,
		id, workspaceID, update.Name, update.Role, update.Email, update.PublicProfileURL,
		update.SourceURL, update.DoNotContact,
	).Scan(&item.ID, &item.WorkspaceID, &item.Name, &item.Role, &item.Email,
		&item.PublicProfileURL, &item.SourceURL, &item.DoNotContact, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

// DeleteContact removes a Contact, blocking if any Relationship still points
// at it. community_contacts is an owned join row and cascades automatically
// (ON DELETE CASCADE), so it is not checked here.
func (s *Store) DeleteContact(ctx context.Context, workspaceID, id string) error {
	var relCount int
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM relationships r JOIN contacts c ON c.id=r.contact_id
		WHERE r.contact_id=$1 AND c.workspace_id=$2`, id, workspaceID).Scan(&relCount); err != nil {
		return err
	}
	if relCount > 0 {
		return dependentsErr("cannot delete: this contact still has relationships attached")
	}
	cmd, err := s.pool.Exec(ctx, `DELETE FROM contacts WHERE id=$1 AND workspace_id=$2`, id, workspaceID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) ListRelationships(ctx context.Context, projectID string, page Page) ([]domain.Relationship, error) {
	limit, limitArgs := page.clause(2)
	args := append([]any{projectID}, limitArgs...)
	rows, err := s.pool.Query(ctx, `
		SELECT r.id::text, r.project_id::text, COALESCE(r.community_id::text,''), COALESCE(c.name,''),
		       COALESCE(r.contact_id::text,''), COALESCE(ct.name,''), r.stage, r.strength,
		       r.last_touch_at, r.next_follow_up_at, r.notes, r.created_at, r.updated_at
		FROM relationships r
		LEFT JOIN communities c ON c.id=r.community_id
		LEFT JOIN contacts ct ON ct.id=r.contact_id
		WHERE r.project_id=$1
		ORDER BY r.next_follow_up_at NULLS LAST, r.strength DESC, r.updated_at DESC`+limit, args...)
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

// UpdateRelationship applies a partial edit to a Relationship. CommunityID/
// ContactID are identity fields fixed at creation and are not editable here.
// The joined community/contact names are optional (LEFT JOIN), so the update
// runs inside a CTE and the name lookup happens in the outer SELECT — an
// UPDATE...FROM cannot express an optional join for its RETURNING clause.
func (s *Store) UpdateRelationship(ctx context.Context, projectID, id string, update domain.RelationshipUpdate) (domain.Relationship, error) {
	var item domain.Relationship
	err := s.pool.QueryRow(ctx, `
		WITH updated AS (
			UPDATE relationships
			SET stage=COALESCE($3,stage),
			    strength=COALESCE($4,strength),
			    next_follow_up_at=COALESCE($5,next_follow_up_at),
			    notes=COALESCE($6,notes),
			    updated_at=now()
			WHERE id=$1 AND project_id=$2
			RETURNING *
		)
		SELECT u.id::text, u.project_id::text, COALESCE(u.community_id::text,''), COALESCE(c.name,''),
		       COALESCE(u.contact_id::text,''), COALESCE(ct.name,''), u.stage, u.strength,
		       u.last_touch_at, u.next_follow_up_at, u.notes, u.created_at, u.updated_at
		FROM updated u
		LEFT JOIN communities c ON c.id=u.community_id
		LEFT JOIN contacts ct ON ct.id=u.contact_id`,
		id, projectID, update.Stage, update.Strength, update.NextFollowUpAt, update.Notes,
	).Scan(&item.ID, &item.ProjectID, &item.CommunityID, &item.CommunityName,
		&item.ContactID, &item.ContactName, &item.Stage, &item.Strength, &item.LastTouchAt,
		&item.NextFollowUpAt, &item.Notes, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

// DeleteRelationship removes a Relationship, blocking if it still has
// Touchpoints logged against it.
func (s *Store) DeleteRelationship(ctx context.Context, projectID, id string) error {
	var tpCount int
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM touchpoints t JOIN relationships r ON r.id=t.relationship_id
		WHERE t.relationship_id=$1 AND r.project_id=$2`, id, projectID).Scan(&tpCount); err != nil {
		return err
	}
	if tpCount > 0 {
		return dependentsErr("cannot delete: this relationship still has touchpoints logged")
	}
	cmd, err := s.pool.Exec(ctx, `DELETE FROM relationships WHERE id=$1 AND project_id=$2`, id, projectID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
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

// UpdateTouchpoint applies a partial edit to a Touchpoint. Only Summary and
// OccurredAt are editable — Channel, Direction and RelationshipID are fixed
// once logged (see domain.TouchpointUpdate).
func (s *Store) UpdateTouchpoint(ctx context.Context, projectID, id string, update domain.TouchpointUpdate) (domain.Touchpoint, error) {
	var item domain.Touchpoint
	err := s.pool.QueryRow(ctx, `
		UPDATE touchpoints t
		SET summary=COALESCE($3,t.summary),
		    occurred_at=COALESCE($4,t.occurred_at)
		FROM relationships r
		WHERE t.id=$1 AND t.relationship_id=r.id AND r.project_id=$2
		RETURNING t.id::text, t.relationship_id::text, t.channel, t.direction, t.summary, t.occurred_at, t.created_at`,
		id, projectID, update.Summary, update.OccurredAt,
	).Scan(&item.ID, &item.RelationshipID, &item.Channel, &item.Direction, &item.Summary, &item.OccurredAt, &item.CreatedAt)
	return item, err
}

// DeleteTouchpoint removes a Touchpoint. Touchpoints have no dependents of
// their own, so this deletes freely once scoped to the caller's project.
func (s *Store) DeleteTouchpoint(ctx context.Context, projectID, id string) error {
	cmd, err := s.pool.Exec(ctx, `
		DELETE FROM touchpoints t USING relationships r
		WHERE t.id=$1 AND t.relationship_id=r.id AND r.project_id=$2`, id, projectID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) ListOutreach(ctx context.Context, projectID string, page Page) ([]domain.Outreach, error) {
	limit, limitArgs := page.clause(2)
	args := append([]any{projectID}, limitArgs...)
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
		         o.updated_at DESC`+limit, args...)
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

// UpdateOutreach applies a partial edit to an Outreach, including the status
// transitions the old status-only endpoint used to handle exclusively (the
// approved_at/sent_at side-effect timestamps still get set the same way).
// The caller (services/api/outreach.go's updateOutreach) is responsible for
// running validOutreachTransition and the email/SMTP delivery gate before
// calling this when Status is set — this method itself does not re-validate
// the transition, matching how UpdateEvent/UpdateScopedCFP leave status-enum
// validation to the API layer. Subject is a nullable column (COALESCE(NULLIF
// pattern); Body/Rationale/Status are not, so they use plain COALESCE.
func (s *Store) UpdateOutreach(ctx context.Context, projectID, id string, update domain.OutreachUpdate) (domain.Outreach, error) {
	var approvedAt, sentAt any
	if update.Status != nil {
		now := time.Now().UTC()
		if *update.Status == "approved" {
			approvedAt = now
		}
		if *update.Status == "sent" {
			sentAt = now
		}
	}
	var item domain.Outreach
	err := s.pool.QueryRow(ctx, `
		WITH updated AS (
			UPDATE outreach
			SET subject=COALESCE(NULLIF($3,''),subject),
			    body=COALESCE($4,body),
			    rationale=COALESCE($5,rationale),
			    status=COALESCE($6,status),
			    approved_at=COALESCE($7,approved_at),
			    sent_at=COALESCE($8,sent_at),
			    updated_at=now()
			WHERE id=$1 AND project_id=$2
			RETURNING *
		)
		SELECT u.id::text, u.project_id::text, COALESCE(u.community_id::text,''), COALESCE(c.name,''),
		       COALESCE(u.contact_id::text,''), COALESCE(ct.name,''), COALESCE(u.talk_id::text,''), COALESCE(t.title,''),
		       u.channel, COALESCE(u.subject,''), u.body, u.rationale, u.status, u.approved_at, u.sent_at,
		       u.created_at, u.updated_at
		FROM updated u
		LEFT JOIN communities c ON c.id=u.community_id
		LEFT JOIN contacts ct ON ct.id=u.contact_id
		LEFT JOIN talks t ON t.id=u.talk_id`,
		id, projectID, update.Subject, update.Body, update.Rationale, update.Status, approvedAt, sentAt,
	).Scan(&item.ID, &item.ProjectID, &item.CommunityID, &item.CommunityName,
		&item.ContactID, &item.ContactName, &item.TalkID, &item.TalkTitle, &item.Channel,
		&item.Subject, &item.Body, &item.Rationale, &item.Status, &item.ApprovedAt, &item.SentAt,
		&item.CreatedAt, &item.UpdatedAt)
	return item, err
}

// DeleteOutreach removes an Outreach. It blocks if the outreach is still
// linked to a campaign (campaign_items is a polymorphic reference with no DB
// FK) or is still cited as the source of a work item (work_items.source_type
// includes 'outreach' — see migrations/000003_work_items.up.sql — so a
// dangling source_id would otherwise be left behind). outreach_deliveries
// cascades automatically (ON DELETE CASCADE) and needs no check.
func (s *Store) DeleteOutreach(ctx context.Context, projectID, id string) error {
	if referenced, err := s.campaignItemReferences(ctx, "outreach", id); err != nil {
		return err
	} else if referenced {
		return dependentsErr("cannot delete: this outreach is linked to a campaign")
	}
	if referenced, err := s.workItemSourceReferences(ctx, "outreach", id); err != nil {
		return err
	} else if referenced {
		return dependentsErr("cannot delete: a work item was generated from this outreach")
	}
	cmd, err := s.pool.Exec(ctx, `DELETE FROM outreach WHERE id=$1 AND project_id=$2`, id, projectID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
