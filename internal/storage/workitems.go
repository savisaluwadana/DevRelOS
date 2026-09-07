package storage

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	signaldomain "github.com/savisaluwadana/DevRelOS/internal/domain/signals"
	workdomain "github.com/savisaluwadana/DevRelOS/internal/domain/workitems"
)

type WorkItemFilter struct {
	Status string
	Kind   string
	Limit  int
}

func (s *Store) ListWorkItems(ctx context.Context, projectID string, filter WorkItemFilter) ([]workdomain.WorkItem, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, project_id::text, source_type, COALESCE(source_id::text,''), kind,
		       title, description, priority, status, owner, due_at, metadata, created_at, updated_at
		FROM work_items
		WHERE project_id=$1
		  AND ($2='' OR status=$2)
		  AND ($3='' OR kind=$3)
		ORDER BY CASE status
		           WHEN 'in_progress' THEN 0
		           WHEN 'planned' THEN 1
		           WHEN 'backlog' THEN 2
		           WHEN 'blocked' THEN 3
		           WHEN 'done' THEN 4
		           ELSE 5
		         END,
		         priority DESC,
		         due_at NULLS LAST,
		         updated_at DESC
		LIMIT $4`, projectID, filter.Status, filter.Kind, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]workdomain.WorkItem, 0)
	for rows.Next() {
		var item workdomain.WorkItem
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.SourceType, &item.SourceID, &item.Kind,
			&item.Title, &item.Description, &item.Priority, &item.Status, &item.Owner, &item.DueAt,
			&metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Metadata = map[string]any{}
		_ = json.Unmarshal(metadata, &item.Metadata)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetWorkItem(ctx context.Context, projectID, id string) (workdomain.WorkItem, error) {
	var item workdomain.WorkItem
	var metadata []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, project_id::text, source_type, COALESCE(source_id::text,''), kind,
		       title, description, priority, status, owner, due_at, metadata, created_at, updated_at
		FROM work_items
		WHERE id=$1 AND project_id=$2`, id, projectID).Scan(
		&item.ID, &item.ProjectID, &item.SourceType, &item.SourceID, &item.Kind,
		&item.Title, &item.Description, &item.Priority, &item.Status, &item.Owner,
		&item.DueAt, &metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, err
	}
	item.Metadata = map[string]any{}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func (s *Store) CreateWorkItem(ctx context.Context, item workdomain.WorkItem) (workdomain.WorkItem, error) {
	if item.SourceType == "" {
		item.SourceType = "manual"
	}
	if item.Status == "" {
		item.Status = "backlog"
	}
	if item.Priority == 0 {
		item.Priority = 50
	}
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	metadata, err := json.Marshal(item.Metadata)
	if err != nil {
		return item, err
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO work_items (
			project_id, source_type, source_id, kind, title, description,
			priority, status, owner, due_at, metadata
		)
		VALUES ($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8,$9,$10,$11::jsonb)
		RETURNING id::text, created_at, updated_at`,
		item.ProjectID, item.SourceType, item.SourceID, item.Kind, item.Title, item.Description,
		item.Priority, item.Status, item.Owner, item.DueAt, metadata,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) UpdateWorkItemStatus(ctx context.Context, projectID, id, status string) error {
	command, err := s.pool.Exec(ctx, `
		UPDATE work_items SET status=$3, updated_at=now() WHERE id=$1 AND project_id=$2`, id, projectID, status)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) GetPainPoint(ctx context.Context, projectID, id string) (signaldomain.PainPoint, error) {
	var item signaldomain.PainPoint
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, project_id::text, key, title, summary, persona, severity, trend_score,
		       evidence_count, topics, status, first_seen_at, last_seen_at, created_at, updated_at
		FROM pain_points WHERE id=$1 AND project_id=$2`, id, projectID).Scan(
		&item.ID, &item.ProjectID, &item.Key, &item.Title, &item.Summary, &item.Persona,
		&item.Severity, &item.TrendScore, &item.EvidenceCount, &item.Topics, &item.Status,
		&item.FirstSeenAt, &item.LastSeenAt, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func (s *Store) WorkItemExistsForSource(ctx context.Context, projectID, sourceType, sourceID, kind string) (bool, error) {
	if sourceID == "" {
		return false, errors.New("source id is required")
	}
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM work_items
			WHERE project_id=$1 AND source_type=$2 AND source_id=$3::uuid AND kind=$4
		)`, projectID, sourceType, sourceID, kind).Scan(&exists)
	return exists, err
}
