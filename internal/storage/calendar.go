package storage

import (
	"context"
	"time"

	calendardomain "github.com/savisaluwadana/DevRelOS/internal/domain/calendar"
)

func (s *Store) ListCalendarItems(ctx context.Context, projectID string, from, to time.Time, page Page) ([]calendardomain.Item, error) {
	limit, limitArgs := page.clause(4)
	args := append([]any{projectID, from, to}, limitArgs...)
	rows, err := s.pool.Query(ctx, `
		SELECT id, kind, title, subtitle, starts_at, ends_at, status, href, priority, source_id
		FROM (
			SELECT 'event:' || e.id::text AS id, 'event' AS kind, e.name AS title,
			       concat_ws(' · ', NULLIF(e.city,''), NULLIF(e.country,''), NULLIF(e.event_type,'')) AS subtitle,
			       e.starts_at AS starts_at, e.ends_at AS ends_at, e.status,
			       '/manage#events' AS href, 60 AS priority, e.id::text AS source_id
			FROM events e
			WHERE e.project_id=$1 AND e.starts_at IS NOT NULL AND e.status <> 'archived'

			UNION ALL
			SELECT 'cfp:' || c.id::text, 'cfp', COALESCE(NULLIF(c.name,''),'CFP') || ' — ' || e.name,
			       CASE WHEN cardinality(c.tracks) > 0 THEN array_to_string(c.tracks, ' · ') ELSE 'Submission deadline' END,
			       c.closes_at, NULL::timestamptz, c.status,
			       '/manage#cfps', 95, c.id::text
			FROM cfps c JOIN events e ON e.id=c.event_id
			WHERE e.project_id=$1 AND c.closes_at IS NOT NULL AND c.status IN ('upcoming','open')

			UNION ALL
			SELECT 'work:' || w.id::text, 'work_item', w.title,
			       replace(w.kind, '_', ' ') || CASE WHEN w.owner <> '' THEN ' · ' || w.owner ELSE '' END,
			       w.due_at, NULL::timestamptz, w.status,
			       '/work', LEAST(100, GREATEST(1, w.priority)), w.id::text
			FROM work_items w
			WHERE w.project_id=$1 AND w.due_at IS NOT NULL AND w.status NOT IN ('done','cancelled')

			UNION ALL
			SELECT 'content:' || ca.id::text, 'content', ca.title,
			       replace(ca.channel, '_', ' ') || ' · ' || replace(ca.format, '_', ' '),
			       ca.scheduled_at, NULL::timestamptz, ca.status,
			       '/content', 55, ca.id::text
			FROM content_assets ca
			WHERE ca.project_id=$1 AND ca.scheduled_at IS NOT NULL AND ca.status NOT IN ('published','archived')

			UNION ALL
			SELECT 'campaign-start:' || c.id::text, 'campaign_start', c.name,
			       'Campaign starts', c.starts_at, NULL::timestamptz, c.status,
			       '/campaigns', 50, c.id::text
			FROM campaigns c
			WHERE c.project_id=$1 AND c.starts_at IS NOT NULL AND c.status <> 'archived'

			UNION ALL
			SELECT 'campaign-end:' || c.id::text, 'campaign_end', c.name,
			       'Campaign target end', c.ends_at, NULL::timestamptz, c.status,
			       '/campaigns', 70, c.id::text
			FROM campaigns c
			WHERE c.project_id=$1 AND c.ends_at IS NOT NULL AND c.status <> 'archived'

			UNION ALL
			SELECT 'relationship:' || r.id::text, 'relationship_follow_up',
			       COALESCE(NULLIF(ct.name,''), NULLIF(com.name,''), 'Relationship follow-up'),
			       'Relationship follow-up · ' || r.stage,
			       r.next_follow_up_at, NULL::timestamptz, r.stage,
			       '/relationships', 85, r.id::text
			FROM relationships r
			LEFT JOIN contacts ct ON ct.id=r.contact_id
			LEFT JOIN communities com ON com.id=r.community_id
			WHERE r.project_id=$1 AND r.next_follow_up_at IS NOT NULL AND r.stage <> 'dormant'
		) calendar
		WHERE starts_at >= $2 AND starts_at <= $3
		ORDER BY starts_at ASC, priority DESC, kind ASC`+limit, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]calendardomain.Item, 0)
	for rows.Next() {
		var item calendardomain.Item
		if err := rows.Scan(&item.ID, &item.Kind, &item.Title, &item.Subtitle, &item.StartsAt, &item.EndsAt,
			&item.Status, &item.Href, &item.Priority, &item.SourceID); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
