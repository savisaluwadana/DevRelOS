package storage

import "context"

func (s *Store) ProjectWorkspaceID(ctx context.Context, projectID string) (string, error) {
	var workspaceID string
	err := s.pool.QueryRow(ctx, `SELECT workspace_id::text FROM projects WHERE id=$1`, projectID).Scan(&workspaceID)
	return workspaceID, err
}
