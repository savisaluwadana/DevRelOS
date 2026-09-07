package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	domain "github.com/savisaluwadana/DevRelOS/internal/domain/connectors"
	"github.com/jackc/pgx/v5"
)

type ClaimedConnectorRun struct {
	Run       domain.Run
	Connector domain.Connector
}

func (s *Store) ClaimNextConnectorRun(ctx context.Context) (*ClaimedConnectorRun, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var claimed ClaimedConnectorRun
	var configJSON, policyJSON []byte
	err = tx.QueryRow(ctx, `
		SELECT r.id::text, r.connector_id::text, r.created_at,
		       c.id::text, c.workspace_id::text, c.provider, c.name, c.enabled, c.config, c.policy,
		       COALESCE(c.secret_id::text,''), c.created_at, c.updated_at
		FROM connector_runs r
		JOIN connectors c ON c.id=r.connector_id
		WHERE r.status='queued'
		ORDER BY r.created_at
		FOR UPDATE OF r SKIP LOCKED
		LIMIT 1`).Scan(
		&claimed.Run.ID, &claimed.Run.ConnectorID, &claimed.Run.CreatedAt,
		&claimed.Connector.ID, &claimed.Connector.WorkspaceID, &claimed.Connector.Provider,
		&claimed.Connector.Name, &claimed.Connector.Enabled, &configJSON, &policyJSON,
		&claimed.Connector.SecretID, &claimed.Connector.CreatedAt, &claimed.Connector.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `UPDATE connector_runs SET status='running', started_at=$2 WHERE id=$1`, claimed.Run.ID, now); err != nil {
		return nil, err
	}
	claimed.Run.Status = "running"
	claimed.Run.StartedAt = &now
	claimed.Connector.Config = map[string]any{}
	claimed.Connector.Policy = map[string]any{}
	_ = json.Unmarshal(configJSON, &claimed.Connector.Config)
	_ = json.Unmarshal(policyJSON, &claimed.Connector.Policy)

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &claimed, nil
}

func (s *Store) FinishConnectorRun(ctx context.Context, run domain.Run) error {
	if run.Status != "succeeded" && run.Status != "failed" && run.Status != "cancelled" {
		return errors.New("connector run must finish with succeeded, failed, or cancelled status")
	}
	warnings, err := json.Marshal(run.Warnings)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	_, err = s.pool.Exec(ctx, `
		UPDATE connector_runs
		SET status=$2,
		    cursor=NULLIF($3,''),
		    requests_made=$4,
		    items_fetched=$5,
		    items_created=$6,
		    items_updated=$7,
		    items_skipped=$8,
		    provider_cost_usd=$9,
		    warnings=$10::jsonb,
		    error=NULLIF($11,''),
		    finished_at=$12
		WHERE id=$1`,
		run.ID, run.Status, run.Cursor, run.RequestsMade, run.ItemsFetched, run.ItemsCreated,
		run.ItemsUpdated, run.ItemsSkipped, run.ProviderCostUSD, warnings, run.Error, now)
	return err
}
