package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	contentdomain "github.com/savisaluwadana/DevRelOS/internal/domain/content"
)

type ContentAssetFilter struct {
	Status  string
	Channel string
	Limit   int
}

func (s *Store) ListContentAssets(ctx context.Context, projectID string, filter ContentAssetFilter) ([]contentdomain.Asset, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, project_id::text, COALESCE(work_item_id::text,''), channel, format,
		       title, audience, objective, brief, draft, status, topics,
		       COALESCE(source_url,''), COALESCE(published_url,''), scheduled_at, published_at,
		       metadata, created_at, updated_at
		FROM content_assets
		WHERE project_id=$1
		  AND ($2='' OR status=$2)
		  AND ($3='' OR channel=$3)
		ORDER BY CASE status
		           WHEN 'review' THEN 0
		           WHEN 'drafting' THEN 1
		           WHEN 'brief' THEN 2
		           WHEN 'approved' THEN 3
		           WHEN 'published' THEN 4
		           ELSE 5
		         END,
		         scheduled_at NULLS LAST,
		         updated_at DESC
		LIMIT $4`, projectID, filter.Status, filter.Channel, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]contentdomain.Asset, 0)
	for rows.Next() {
		var item contentdomain.Asset
		var metadata []byte
		if err := rows.Scan(
			&item.ID, &item.ProjectID, &item.WorkItemID, &item.Channel, &item.Format,
			&item.Title, &item.Audience, &item.Objective, &item.Brief, &item.Draft,
			&item.Status, &item.Topics, &item.SourceURL, &item.PublishedURL,
			&item.ScheduledAt, &item.PublishedAt, &metadata, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Metadata = map[string]any{}
		_ = json.Unmarshal(metadata, &item.Metadata)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateContentAsset(ctx context.Context, item contentdomain.Asset) (contentdomain.Asset, error) {
	if item.Status == "" {
		item.Status = "brief"
	}
	if item.Audience == "" {
		item.Audience = "developers"
	}
	if item.Topics == nil {
		item.Topics = []string{}
	}
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	metadata, err := json.Marshal(item.Metadata)
	if err != nil {
		return item, err
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO content_assets (
			project_id, work_item_id, channel, format, title, audience, objective,
			brief, draft, status, topics, source_url, published_url, scheduled_at,
			published_at, metadata
		)
		VALUES (
			$1,NULLIF($2,'')::uuid,$3,$4,$5,$6,$7,$8,$9,$10,$11,
			NULLIF($12,''),NULLIF($13,''),$14,$15,$16::jsonb
		)
		RETURNING id::text, created_at, updated_at`,
		item.ProjectID, item.WorkItemID, item.Channel, item.Format, item.Title,
		item.Audience, item.Objective, item.Brief, item.Draft, item.Status, item.Topics,
		item.SourceURL, item.PublishedURL, item.ScheduledAt, item.PublishedAt, metadata,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) UpdateContentAsset(ctx context.Context, projectID, id string, update contentdomain.AssetUpdate) (contentdomain.Asset, error) {
	var publishedAt any
	if update.Status != nil && *update.Status == "published" {
		publishedAt = time.Now().UTC()
	}
	var item contentdomain.Asset
	var metadata []byte
	err := s.pool.QueryRow(ctx, `
		UPDATE content_assets
		SET title=COALESCE($3,title),
		    audience=COALESCE($4,audience),
		    objective=COALESCE($5,objective),
		    brief=COALESCE($6,brief),
		    draft=COALESCE($7,draft),
		    status=COALESCE($8,status),
		    source_url=COALESCE(NULLIF($9,''),source_url),
		    published_url=COALESCE(NULLIF($10,''),published_url),
		    scheduled_at=COALESCE($11,scheduled_at),
		    published_at=COALESCE($12,published_at),
		    updated_at=now()
		WHERE id=$1 AND project_id=$2
		RETURNING id::text, project_id::text, COALESCE(work_item_id::text,''), channel, format,
		          title, audience, objective, brief, draft, status, topics,
		          COALESCE(source_url,''), COALESCE(published_url,''), scheduled_at, published_at,
		          metadata, created_at, updated_at`,
		id, projectID, update.Title, update.Audience, update.Objective, update.Brief, update.Draft,
		update.Status, update.SourceURL, update.PublishedURL, update.ScheduledAt, publishedAt,
	).Scan(
		&item.ID, &item.ProjectID, &item.WorkItemID, &item.Channel, &item.Format,
		&item.Title, &item.Audience, &item.Objective, &item.Brief, &item.Draft,
		&item.Status, &item.Topics, &item.SourceURL, &item.PublishedURL,
		&item.ScheduledAt, &item.PublishedAt, &metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, err
	}
	item.Metadata = map[string]any{}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func (s *Store) ContentAssetExistsForWork(ctx context.Context, projectID, workItemID, channel, format string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM content_assets
			WHERE project_id=$1 AND work_item_id=$2::uuid AND channel=$3 AND format=$4
		)`, projectID, workItemID, channel, format).Scan(&exists)
	return exists, err
}

func (s *Store) GetContentAsset(ctx context.Context, projectID, id string) (contentdomain.Asset, error) {
	var item contentdomain.Asset
	var metadata []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, project_id::text, COALESCE(work_item_id::text,''), channel, format,
		       title, audience, objective, brief, draft, status, topics,
		       COALESCE(source_url,''), COALESCE(published_url,''), scheduled_at, published_at,
		       metadata, created_at, updated_at
		FROM content_assets WHERE id=$1 AND project_id=$2`, id, projectID).Scan(
		&item.ID, &item.ProjectID, &item.WorkItemID, &item.Channel, &item.Format,
		&item.Title, &item.Audience, &item.Objective, &item.Brief, &item.Draft,
		&item.Status, &item.Topics, &item.SourceURL, &item.PublishedURL,
		&item.ScheduledAt, &item.PublishedAt, &metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, err
	}
	item.Metadata = map[string]any{}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

var _ = pgx.ErrNoRows
