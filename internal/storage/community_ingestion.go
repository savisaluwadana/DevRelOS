package storage

import (
	"context"

	"github.com/savisaluwadana/DevRelOS/internal/domain/events"
)

func (s *Store) UpsertCommunity(ctx context.Context, item events.Community, sourceRecordID string) (events.Community, bool, error) {
	if item.Platform == "" {
		item.Platform = "manual"
	}
	if item.Status == "" {
		item.Status = "discovered"
	}
	if item.Topics == nil {
		item.Topics = []string{}
	}
	var inserted bool
	err := s.pool.QueryRow(ctx, `
		INSERT INTO communities (
			project_id, source_record_id, name, platform, external_id, website_url, city, country,
			timezone, topics, member_count, activity_score, speaking_fit_score, last_event_at,
			next_event_at, status
		)
		VALUES ($1,NULLIF($2,'')::uuid,$3,$4,NULLIF($5,''),NULLIF($6,''),NULLIF($7,''),NULLIF($8,''),
		        NULLIF($9,''),$10,$11,$12,$13,$14,$15,$16)
		ON CONFLICT (project_id, platform, external_id)
		DO UPDATE SET source_record_id=EXCLUDED.source_record_id,
		              name=EXCLUDED.name,
		              website_url=EXCLUDED.website_url,
		              city=EXCLUDED.city,
		              country=EXCLUDED.country,
		              timezone=EXCLUDED.timezone,
		              topics=EXCLUDED.topics,
		              member_count=COALESCE(EXCLUDED.member_count, communities.member_count),
		              activity_score=COALESCE(EXCLUDED.activity_score, communities.activity_score),
		              speaking_fit_score=COALESCE(EXCLUDED.speaking_fit_score, communities.speaking_fit_score),
		              last_event_at=COALESCE(EXCLUDED.last_event_at, communities.last_event_at),
		              next_event_at=COALESCE(EXCLUDED.next_event_at, communities.next_event_at),
		              updated_at=now()
		RETURNING id::text, created_at, updated_at, (xmax = 0)`,
		item.ProjectID, sourceRecordID, item.Name, item.Platform, item.ExternalID, item.WebsiteURL,
		item.City, item.Country, item.Timezone, item.Topics, item.MemberCount, item.ActivityScore,
		item.SpeakingFitScore, item.LastEventAt, item.NextEventAt, item.Status,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt, &inserted)
	return item, inserted, err
}
