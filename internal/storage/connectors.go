package storage

import (
	"context"
	"encoding/json"

	domain "github.com/savisaluwadana/DevRelOS/internal/domain/connectors"
	"github.com/jackc/pgx/v5"
)

func (s *Store) DefaultWorkspaceID(ctx context.Context) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug='default' LIMIT 1`).Scan(&id)
	return id, err
}

func (s *Store) ListConnectors(ctx context.Context, workspaceID string) ([]domain.Connector, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, workspace_id::text, provider, name, enabled, config, policy,
		       COALESCE(secret_id::text,''), schedule_minutes, next_run_at, created_at, updated_at
		FROM connectors
		WHERE workspace_id=$1
		ORDER BY provider, name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.Connector, 0)
	for rows.Next() {
		var item domain.Connector
		var configJSON, policyJSON []byte
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.Provider, &item.Name, &item.Enabled,
			&configJSON, &policyJSON, &item.SecretID, &item.ScheduleMinutes, &item.NextRunAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Config = map[string]any{}
		item.Policy = map[string]any{}
		_ = json.Unmarshal(configJSON, &item.Config)
		_ = json.Unmarshal(policyJSON, &item.Policy)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateConnector(ctx context.Context, item domain.Connector) (domain.Connector, error) {
	if err := validateScheduleMinutes(item.ScheduleMinutes); err != nil {
		return item, err
	}
	if item.Config == nil {
		item.Config = map[string]any{}
	}
	if item.Policy == nil {
		item.Policy = map[string]any{}
	}
	configJSON, err := json.Marshal(item.Config)
	if err != nil {
		return item, err
	}
	policyJSON, err := json.Marshal(item.Policy)
	if err != nil {
		return item, err
	}

	err = s.pool.QueryRow(ctx, `
		INSERT INTO connectors (workspace_id, provider, name, enabled, config, policy, secret_id, schedule_minutes, next_run_at)
		VALUES ($1,$2,$3,$4,$5::jsonb,$6::jsonb,NULLIF($7,'')::uuid,$8,
		        CASE WHEN $8::integer IS NULL THEN NULL ELSE now() END)
		RETURNING id::text, COALESCE(secret_id::text,''), schedule_minutes, next_run_at, created_at, updated_at`,
		item.WorkspaceID, item.Provider, item.Name, item.Enabled, configJSON, policyJSON, item.SecretID, item.ScheduleMinutes).
		Scan(&item.ID, &item.SecretID, &item.ScheduleMinutes, &item.NextRunAt, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) UpdateConnectorSecret(ctx context.Context, workspaceID, connectorID, secretID string) error {
	command, err := s.pool.Exec(ctx, `
		UPDATE connectors
		SET secret_id=NULLIF($3,'')::uuid, updated_at=now()
		WHERE id=$1 AND workspace_id=$2`, connectorID, workspaceID, secretID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) ListConnectorRuns(ctx context.Context, connectorID string) ([]domain.Run, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, connector_id::text, status, COALESCE(cursor,''), requests_made,
		       items_fetched, items_created, items_updated, items_skipped, provider_cost_usd,
		       warnings, COALESCE(error,''), started_at, finished_at, created_at
		FROM connector_runs
		WHERE connector_id=$1
		ORDER BY created_at DESC
		LIMIT 100`, connectorID)
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

func (s *Store) QueueConnectorRun(ctx context.Context, connectorID string) (domain.Run, error) {
	var run domain.Run
	run.ConnectorID = connectorID
	run.Status = "queued"
	err := s.pool.QueryRow(ctx, `
		INSERT INTO connector_runs (connector_id, status)
		VALUES ($1,'queued')
		RETURNING id::text, created_at`, connectorID).Scan(&run.ID, &run.CreatedAt)
	return run, err
}

func (s *Store) UpsertSourceRecord(ctx context.Context, item domain.SourceRecord) (domain.SourceRecord, bool, error) {
	if item.Provenance == nil {
		item.Provenance = map[string]any{}
	}
	provenanceJSON, err := json.Marshal(item.Provenance)
	if err != nil {
		return item, false, err
	}
	var payload any
	if item.RawPayload != nil {
		rawJSON, marshalErr := json.Marshal(item.RawPayload)
		if marshalErr != nil {
			return item, false, marshalErr
		}
		payload = rawJSON
	}

	var inserted bool
	err = s.pool.QueryRow(ctx, `
		INSERT INTO source_records (workspace_id, provider, external_id, canonical_url, source_timestamp, content_hash, raw_payload, provenance)
		VALUES ($1,$2,NULLIF($3,''),NULLIF($4,''),$5,NULLIF($6,''),$7,$8::jsonb)
		ON CONFLICT (workspace_id, provider, external_id)
		DO UPDATE SET canonical_url=EXCLUDED.canonical_url,
		              source_timestamp=EXCLUDED.source_timestamp,
		              fetched_at=now(),
		              content_hash=EXCLUDED.content_hash,
		              raw_payload=EXCLUDED.raw_payload,
		              provenance=EXCLUDED.provenance
		RETURNING id::text, fetched_at, (xmax = 0)`,
		item.WorkspaceID, item.Provider, item.ExternalID, item.CanonicalURL, item.SourceTimestamp,
		item.ContentHash, payload, provenanceJSON).Scan(&item.ID, &item.FetchedAt, &inserted)
	return item, inserted, err
}
