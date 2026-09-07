package storage

import "context"

func (s *Store) OutreachStatus(ctx context.Context, projectID, id string) (string, error) {
	var status string
	err := s.pool.QueryRow(ctx, `SELECT status FROM outreach WHERE id=$1 AND project_id=$2`, id, projectID).Scan(&status)
	return status, err
}
