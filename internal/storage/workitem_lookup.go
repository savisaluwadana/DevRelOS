package storage

import (
	"context"
	"encoding/json"

	workdomain "github.com/savisaluwadana/DevRelOS/internal/domain/workitems"
)

func (s *Store) GetWorkItem(ctx context.Context, projectID, id string) (workdomain.WorkItem, error) {
	var item workdomain.WorkItem
	var metadata []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, project_id::text, source_type, COALESCE(source_id::text,''), kind,
		       title, description, priority, status, owner, due_at, metadata, created_at, updated_at
		FROM work_items
		WHERE id=$1 AND project_id=$2`, id, projectID).Scan(
		&item.ID, &item.ProjectID, &item.SourceType, &item.SourceID, &item.Kind,
		&item.Title, &item.Description, &item.Priority, &item.Status, &item.Owner, &item.DueAt,
		&metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, err
	}
	item.Metadata = map[string]any{}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}
