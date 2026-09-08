package storage

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	campaigndomain "github.com/savisaluwadana/DevRelOS/internal/domain/campaigns"
)

func (s *Store) ListCampaignItems(ctx context.Context, projectID, campaignID string, page Page) ([]campaigndomain.Item, error) {
	limit, limitArgs := page.clause(3)
	args := append([]any{campaignID, projectID}, limitArgs...)
	rows, err := s.pool.Query(ctx, `
		SELECT ci.id::text, ci.campaign_id::text, ci.entity_type, ci.entity_id::text,
		       ci.channel, ci.cost_usd::float8, ci.metadata, ci.created_at
		FROM campaign_items ci
		JOIN campaigns c ON c.id=ci.campaign_id
		WHERE ci.campaign_id=$1 AND c.project_id=$2
		ORDER BY ci.created_at DESC`+limit, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]campaigndomain.Item, 0)
	for rows.Next() {
		var item campaigndomain.Item
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.CampaignID, &item.EntityType, &item.EntityID, &item.Channel, &item.CostUSD, &metadata, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Metadata = map[string]any{}
		_ = json.Unmarshal(metadata, &item.Metadata)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) DeleteCampaignItem(ctx context.Context, projectID, campaignID, itemID string) error {
	command, err := s.pool.Exec(ctx, `
		DELETE FROM campaign_items ci
		USING campaigns c
		WHERE ci.id=$1 AND ci.campaign_id=$2 AND c.id=ci.campaign_id AND c.project_id=$3`, itemID, campaignID, projectID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
