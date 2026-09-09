package storage

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	domain "github.com/savisaluwadana/DevRelOS/internal/domain/signals"
)

type SignalFilter struct {
	Status   string
	Provider string
	Topic    string
	Query    string
	Limit    int
	Offset   int
}

func (s *Store) ListSignals(ctx context.Context, projectID string, filter SignalFilter) ([]domain.Signal, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id::text, project_id::text, COALESCE(source_record_id::text,''), provider,
		       COALESCE(external_id,''), COALESCE(canonical_url,''), COALESCE(author_handle,''),
		       COALESCE(author_name,''), title, body, occurred_at, topics, engagement_score,
		       relevance_score, status, source_shape, created_at, updated_at
		FROM signals
		WHERE project_id=$1
		  AND ($2='' OR status=$2)
		  AND ($3='' OR provider=$3)
		  AND ($4='' OR $4=ANY(topics))
		  AND ($5='' OR title ILIKE '%%' || $5 || '%%' OR body ILIKE '%%' || $5 || '%%')
		ORDER BY COALESCE(occurred_at, created_at) DESC, engagement_score DESC
		LIMIT $6 OFFSET $7`, projectID, filter.Status, filter.Provider, filter.Topic, filter.Query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.Signal, 0)
	for rows.Next() {
		var item domain.Signal
		if err := rows.Scan(
			&item.ID, &item.ProjectID, &item.SourceRecordID, &item.Provider, &item.ExternalID,
			&item.CanonicalURL, &item.AuthorHandle, &item.AuthorName, &item.Title, &item.Body,
			&item.OccurredAt, &item.Topics, &item.EngagementScore, &item.RelevanceScore,
			&item.Status, &item.SourceShape, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateSignal(ctx context.Context, item domain.Signal) (domain.Signal, error) {
	if item.Status == "" {
		item.Status = "new"
	}
	if item.Topics == nil {
		item.Topics = []string{}
	}

	err := s.pool.QueryRow(ctx, `
		INSERT INTO signals (
			project_id, source_record_id, provider, external_id, canonical_url, author_handle,
			author_name, title, body, occurred_at, topics, engagement_score, relevance_score, status,
			source_shape
		)
		VALUES (
			$1, NULLIF($2,'')::uuid, $3, NULLIF($4,''), NULLIF($5,''), NULLIF($6,''),
			NULLIF($7,''), $8, $9, $10, $11, $12, $13, $14, COALESCE(NULLIF($15,''),'unknown')
		)
		ON CONFLICT (project_id, provider, external_id)
		DO UPDATE SET canonical_url=EXCLUDED.canonical_url,
		              author_handle=EXCLUDED.author_handle,
		              author_name=EXCLUDED.author_name,
		              title=EXCLUDED.title,
		              body=EXCLUDED.body,
		              occurred_at=EXCLUDED.occurred_at,
		              topics=EXCLUDED.topics,
		              engagement_score=EXCLUDED.engagement_score,
		              relevance_score=EXCLUDED.relevance_score,
		              source_shape=EXCLUDED.source_shape,
		              updated_at=now()
		RETURNING id::text, created_at, updated_at`,
		item.ProjectID, item.SourceRecordID, item.Provider, item.ExternalID, item.CanonicalURL,
		item.AuthorHandle, item.AuthorName, item.Title, item.Body, item.OccurredAt, item.Topics,
		item.EngagementScore, item.RelevanceScore, item.Status, item.SourceShape,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) UpdateSignalStatus(ctx context.Context, id, projectID, status string) error {
	command, err := s.pool.Exec(ctx, `
		UPDATE signals SET status=$3, updated_at=now() WHERE id=$1 AND project_id=$2`, id, projectID, status)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ListPainPoints returns clusters for a project. An empty status means active
// only; pass "all" to include archived history, or a specific status to filter.
//
// The default used to be every status, which defeated the point of archiving:
// a rebuild retires clusters that no longer form by setting status='archived',
// but they kept appearing in the listing - and being sorted by severity, the
// retired ones sat at the top. On a real dataset that meant 13 obsolete pain
// points shown above the 5 live ones.
func (s *Store) ListPainPoints(ctx context.Context, projectID, status string, limit, offset int) ([]domain.PainPoint, error) {
	if strings.TrimSpace(status) == "" {
		status = "active"
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, project_id::text, key, title, summary, persona, severity, trend_score,
		       evidence_count, topics, status, first_seen_at, last_seen_at, created_at, updated_at
		FROM pain_points
		WHERE project_id=$1 AND ($2='all' OR status=$2)
		ORDER BY severity DESC, trend_score DESC, evidence_count DESC
		LIMIT $3 OFFSET $4`, projectID, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.PainPoint, 0)
	for rows.Next() {
		var item domain.PainPoint
		if err := rows.Scan(
			&item.ID, &item.ProjectID, &item.Key, &item.Title, &item.Summary, &item.Persona,
			&item.Severity, &item.TrendScore, &item.EvidenceCount, &item.Topics, &item.Status,
			&item.FirstSeenAt, &item.LastSeenAt, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListPainPointEvidence(ctx context.Context, projectID, painPointID string, limit int) ([]domain.Signal, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT s.id::text, s.project_id::text, COALESCE(s.source_record_id::text,''), s.provider,
		       COALESCE(s.external_id,''), COALESCE(s.canonical_url,''), COALESCE(s.author_handle,''),
		       COALESCE(s.author_name,''), s.title, s.body, s.occurred_at, s.topics,
		       s.engagement_score, s.relevance_score, s.status, s.created_at, s.updated_at
		FROM pain_point_signals pps
		JOIN pain_points pp ON pp.id=pps.pain_point_id
		JOIN signals s ON s.id=pps.signal_id
		WHERE pp.id=$1 AND pp.project_id=$2
		ORDER BY pps.weight DESC, COALESCE(s.occurred_at, s.created_at) DESC
		LIMIT $3`, painPointID, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.Signal, 0)
	for rows.Next() {
		var item domain.Signal
		if err := rows.Scan(
			&item.ID, &item.ProjectID, &item.SourceRecordID, &item.Provider, &item.ExternalID,
			&item.CanonicalURL, &item.AuthorHandle, &item.AuthorName, &item.Title, &item.Body,
			&item.OccurredAt, &item.Topics, &item.EngagementScore, &item.RelevanceScore,
			&item.Status, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

type PainPointCluster struct {
	Key        string
	Title      string
	Summary    string
	Persona    string
	Severity   int
	TrendScore int
	Topics     []string
	FirstSeen  *time.Time
	LastSeen   *time.Time
	SignalIDs  []string
}

// ReplacePainPointClusters synchronizes the current active cluster set while preserving
// stable pain-point IDs. Clusters that disappear are archived instead of deleted so any
// work items or audit records that reference them retain historical provenance.
func (s *Store) ReplacePainPointClusters(ctx context.Context, projectID string, clusters []PainPointCluster) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE pain_points
		SET status='archived', updated_at=now()
		WHERE project_id=$1 AND status='active'`, projectID); err != nil {
		return err
	}

	for _, cluster := range clusters {
		var painPointID string
		err := tx.QueryRow(ctx, `
			INSERT INTO pain_points (
				project_id, key, title, summary, persona, severity, trend_score, evidence_count,
				topics, status, first_seen_at, last_seen_at
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'active',$10,$11)
			ON CONFLICT (project_id, key)
			DO UPDATE SET title=EXCLUDED.title,
			              summary=EXCLUDED.summary,
			              persona=EXCLUDED.persona,
			              severity=EXCLUDED.severity,
			              trend_score=EXCLUDED.trend_score,
			              evidence_count=EXCLUDED.evidence_count,
			              topics=EXCLUDED.topics,
			              status='active',
			              first_seen_at=EXCLUDED.first_seen_at,
			              last_seen_at=EXCLUDED.last_seen_at,
			              updated_at=now()
			RETURNING id::text`,
			projectID, cluster.Key, cluster.Title, cluster.Summary, cluster.Persona,
			cluster.Severity, cluster.TrendScore, len(cluster.SignalIDs), cluster.Topics,
			cluster.FirstSeen, cluster.LastSeen,
		).Scan(&painPointID)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `DELETE FROM pain_point_signals WHERE pain_point_id=$1`, painPointID); err != nil {
			return err
		}
		for _, signalID := range cluster.SignalIDs {
			if _, err := tx.Exec(ctx, `
				INSERT INTO pain_point_signals (pain_point_id, signal_id, weight)
				SELECT $1, id, 1 FROM signals WHERE id=$2 AND project_id=$3`,
				painPointID, signalID, projectID); err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}
