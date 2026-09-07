package storage

import "context"

type OutreachState struct {
	Status  string
	Channel string
}

func (s *Store) OutreachStatus(ctx context.Context, projectID, id string) (string, error) {
	state, err := s.OutreachState(ctx, projectID, id)
	return state.Status, err
}

func (s *Store) OutreachState(ctx context.Context, projectID, id string) (OutreachState, error) {
	var state OutreachState
	err := s.pool.QueryRow(ctx, `SELECT status, channel FROM outreach WHERE id=$1 AND project_id=$2`, id, projectID).
		Scan(&state.Status, &state.Channel)
	return state, err
}
