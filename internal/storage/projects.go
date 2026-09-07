package storage

import "context"

func (s *Store) DefaultProjectIDForWorkspace(ctx context.Context, workspaceID string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		SELECT id::text
		FROM projects
		WHERE workspace_id=$1
		ORDER BY CASE WHEN slug='default' THEN 0 ELSE 1 END, created_at
		LIMIT 1`, workspaceID).Scan(&id)
	return id, err
}
