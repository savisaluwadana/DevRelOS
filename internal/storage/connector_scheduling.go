package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

func validateScheduleMinutes(minutes *int) error {
	if minutes == nil {
		return nil
	}
	if *minutes < 15 || *minutes > 10080 {
		return errors.New("scheduleMinutes must be between 15 and 10080 minutes")
	}
	return nil
}

func (s *Store) UpdateConnectorSchedule(ctx context.Context, workspaceID, connectorID string, minutes *int) (*time.Time, error) {
	if err := validateScheduleMinutes(minutes); err != nil {
		return nil, err
	}
	var nextRunAt *time.Time
	err := s.pool.QueryRow(ctx, `
		UPDATE connectors
		SET schedule_minutes=$3,
		    next_run_at=CASE WHEN $3::integer IS NULL THEN NULL ELSE now() END,
		    updated_at=now()
		WHERE id=$1 AND workspace_id=$2
		RETURNING next_run_at`, connectorID, workspaceID, minutes).Scan(&nextRunAt)
	if err != nil {
		return nil, err
	}
	return nextRunAt, nil
}

func (s *Store) QueueDueConnectorRuns(ctx context.Context, now time.Time) (int, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT c.id::text, c.schedule_minutes
		FROM connectors c
		WHERE c.enabled=true
		  AND c.schedule_minutes IS NOT NULL
		  AND c.next_run_at IS NOT NULL
		  AND c.next_run_at <= $1
		  AND NOT EXISTS (
		    SELECT 1 FROM connector_runs r
		    WHERE r.connector_id=c.id AND r.status IN ('queued','running')
		  )
		ORDER BY c.next_run_at
		FOR UPDATE OF c SKIP LOCKED`, now)
	if err != nil {
		return 0, err
	}

	type dueConnector struct {
		id      string
		minutes int
	}
	due := make([]dueConnector, 0)
	for rows.Next() {
		var item dueConnector
		if err := rows.Scan(&item.id, &item.minutes); err != nil {
			rows.Close()
			return 0, err
		}
		due = append(due, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	queued := 0
	for _, item := range due {
		command, err := tx.Exec(ctx, `
			INSERT INTO connector_runs (connector_id, status)
			VALUES ($1,'queued')`, item.id)
		if err != nil {
			return queued, err
		}
		if command.RowsAffected() == 0 {
			continue
		}
		next := now.Add(time.Duration(item.minutes) * time.Minute)
		if _, err := tx.Exec(ctx, `
			UPDATE connectors SET next_run_at=$2, updated_at=now() WHERE id=$1`, item.id, next); err != nil {
			return queued, err
		}
		queued++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return queued, nil
}
