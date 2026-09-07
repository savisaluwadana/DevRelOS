package storage

import (
	"context"
	"encoding/json"

	domain "github.com/savisaluwadana/DevRelOS/internal/domain/connectors"
)

func (s *Store) ListConnectorRunsForWorkspace(ctx context.Context, workspaceID, connectorID string) ([]domain.Run, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id::text, r.connector_id::text, r.status, COALESCE(r.cursor,''), r.requests_made,
		       r.items_fetched, r.items_created, r.items_updated, r.items_skipped, r.provider_cost_usd,
		       r.warnings, COALESCE(r.error,''), r.started_at, r.finished_at, r.created_at
		FROM connector_runs r
		JOIN connectors c ON c.id=r.connector_id
		WHERE r.connector_id=$1 AND c.workspace_id=$2
		ORDER BY r.created_at DESC
		LIMIT 100`, connectorID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.Run, 0)
	for rows.Next() {
		var item domain.Run
		var warningsJSON []byte
		if err := rows.Scan(&item.ID, &item.ConnectorID, &item.Status, &item.Cursor, &item.RequestsMade,
			&item.ItemsFetched, &item.ItemsCreated, &item.ItemsUpdated, &item.ItemsSkipped,
			&item.ProviderCostUSD, &warningsJSON, &item.Error, &item.StartedAt, &item.FinishedAt,
			&item.CreatedAt); err != nil {
			return nil, err
		}
		item.Warnings = []string{}
		_ = json.Unmarshal(warningsJSON, &item.Warnings)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) QueueConnectorRunForWorkspace(ctx context.Context, workspaceID, connectorID string) (domain.Run, error) {
	var run domain.Run
	run.ConnectorID = connectorID
	run.Status = "queued"
	err := s.pool.QueryRow(ctx, `
		INSERT INTO connector_runs (connector_id, status)
		SELECT c.id,'queued'
		FROM connectors c
		WHERE c.id=$1 AND c.workspace_id=$2 AND c.enabled=true
		RETURNING id::text, created_at`, connectorID, workspaceID).Scan(&run.ID, &run.CreatedAt)
	return run, err
}
