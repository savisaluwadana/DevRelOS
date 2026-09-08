package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	campaigndomain "github.com/savisaluwadana/DevRelOS/internal/domain/campaigns"
)

func (s *Store) ListCampaigns(ctx context.Context, projectID string, page Page) ([]campaigndomain.Campaign, error) {
	limit, limitArgs := page.clause(2)
	args := append([]any{projectID}, limitArgs...)
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, project_id::text, name, objective, status, starts_at, ends_at,
		       budget_usd::float8, target, created_at, updated_at
		FROM campaigns WHERE project_id=$1
		ORDER BY CASE status WHEN 'active' THEN 0 WHEN 'planning' THEN 1 WHEN 'paused' THEN 2 WHEN 'completed' THEN 3 ELSE 4 END,
		         updated_at DESC`+limit, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]campaigndomain.Campaign, 0)
	for rows.Next() {
		var item campaigndomain.Campaign
		var target []byte
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Name, &item.Objective, &item.Status, &item.StartsAt, &item.EndsAt,
			&item.BudgetUSD, &target, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Target = map[string]any{}
		_ = json.Unmarshal(target, &item.Target)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateCampaign(ctx context.Context, item campaigndomain.Campaign) (campaigndomain.Campaign, error) {
	if item.Status == "" {
		item.Status = "planning"
	}
	if item.Target == nil {
		item.Target = map[string]any{}
	}
	target, err := json.Marshal(item.Target)
	if err != nil {
		return item, err
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO campaigns(project_id,name,objective,status,starts_at,ends_at,budget_usd,target)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8::jsonb)
		RETURNING id::text, created_at, updated_at`, item.ProjectID, item.Name, item.Objective, item.Status,
		item.StartsAt, item.EndsAt, item.BudgetUSD, target).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) GetCampaign(ctx context.Context, projectID, id string) (campaigndomain.Campaign, error) {
	var item campaigndomain.Campaign
	var target []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, project_id::text, name, objective, status, starts_at, ends_at,
		       budget_usd::float8, target, created_at, updated_at
		FROM campaigns WHERE id=$1 AND project_id=$2`, id, projectID).Scan(
		&item.ID, &item.ProjectID, &item.Name, &item.Objective, &item.Status, &item.StartsAt, &item.EndsAt,
		&item.BudgetUSD, &target, &item.CreatedAt, &item.UpdatedAt)
	if err == nil {
		item.Target = map[string]any{}
		_ = json.Unmarshal(target, &item.Target)
	}
	return item, err
}

func (s *Store) UpdateCampaignStatus(ctx context.Context, projectID, id, status string) error {
	cmd, err := s.pool.Exec(ctx, `UPDATE campaigns SET status=$3, updated_at=now() WHERE id=$1 AND project_id=$2`, id, projectID, status)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) CampaignEntityBelongsToProject(ctx context.Context, projectID, entityType, entityID string) (bool, error) {
	queries := map[string]string{
		"content_asset": `SELECT EXISTS(SELECT 1 FROM content_assets WHERE id=$1 AND project_id=$2)`,
		"work_item":     `SELECT EXISTS(SELECT 1 FROM work_items WHERE id=$1 AND project_id=$2)`,
		"event":         `SELECT EXISTS(SELECT 1 FROM events WHERE id=$1 AND project_id=$2)`,
		"cfp":           `SELECT EXISTS(SELECT 1 FROM cfps c JOIN events e ON e.id=c.event_id WHERE c.id=$1 AND e.project_id=$2)`,
		"submission":    `SELECT EXISTS(SELECT 1 FROM submissions s JOIN talks t ON t.id=s.talk_id WHERE s.id=$1 AND t.project_id=$2)`,
		"outreach":      `SELECT EXISTS(SELECT 1 FROM outreach WHERE id=$1 AND project_id=$2)`,
		"feedback":      `SELECT EXISTS(SELECT 1 FROM feedback_items WHERE id=$1 AND project_id=$2)`,
		"media_asset":   `SELECT EXISTS(SELECT 1 FROM media_assets WHERE id=$1 AND project_id=$2)`,
		"community":     `SELECT EXISTS(SELECT 1 FROM communities WHERE id=$1 AND project_id=$2)`,
		"talk":          `SELECT EXISTS(SELECT 1 FROM talks WHERE id=$1 AND project_id=$2)`,
	}
	query, ok := queries[entityType]
	if !ok {
		return false, fmt.Errorf("unsupported campaign entity type %q", entityType)
	}
	var exists bool
	err := s.pool.QueryRow(ctx, query, entityID, projectID).Scan(&exists)
	return exists, err
}

func (s *Store) LinkCampaignItem(ctx context.Context, projectID string, item campaigndomain.Item) (campaigndomain.Item, error) {
	var campaignProject string
	if err := s.pool.QueryRow(ctx, `SELECT project_id::text FROM campaigns WHERE id=$1 AND project_id=$2`, item.CampaignID, projectID).Scan(&campaignProject); err != nil {
		return item, err
	}
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	metadata, err := json.Marshal(item.Metadata)
	if err != nil {
		return item, err
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO campaign_items(campaign_id,entity_type,entity_id,channel,cost_usd,metadata)
		VALUES($1,$2,$3,$4,$5,$6::jsonb)
		ON CONFLICT(campaign_id,entity_type,entity_id) DO UPDATE SET channel=EXCLUDED.channel, cost_usd=EXCLUDED.cost_usd, metadata=EXCLUDED.metadata
		RETURNING id::text, created_at`, item.CampaignID, item.EntityType, item.EntityID, item.Channel, item.CostUSD, metadata).Scan(&item.ID, &item.CreatedAt)
	return item, err
}

func (s *Store) RecordCampaignMetric(ctx context.Context, projectID string, metric campaigndomain.Metric) (campaigndomain.Metric, error) {
	if metric.Source == "" {
		metric.Source = "manual"
	}
	if metric.ObservedAt.IsZero() {
		metric.ObservedAt = time.Now().UTC()
	}
	if metric.Metadata == nil {
		metric.Metadata = map[string]any{}
	}
	metadata, err := json.Marshal(metric.Metadata)
	if err != nil {
		return metric, err
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO campaign_metrics(campaign_id,metric_key,metric_value,source,observed_at,metadata)
		SELECT c.id,$3,$4,$5,$6,$7::jsonb FROM campaigns c WHERE c.id=$1 AND c.project_id=$2
		RETURNING id::text, created_at`, metric.CampaignID, projectID, metric.MetricKey, metric.MetricValue,
		metric.Source, metric.ObservedAt, metadata).Scan(&metric.ID, &metric.CreatedAt)
	return metric, err
}

// campaignReportItemLimit caps the attributed-item list embedded in a report.
const campaignReportItemLimit = 200

func (s *Store) CampaignReport(ctx context.Context, projectID, campaignID string) (campaigndomain.Report, error) {
	campaign, err := s.GetCampaign(ctx, projectID, campaignID)
	if err != nil {
		return campaigndomain.Report{}, err
	}
	report := campaigndomain.Report{Campaign: campaign, LinkedByType: map[string]int{}, Metrics: map[string]float64{}, RecentMetrics: []campaigndomain.Metric{}}

	rows, err := s.pool.Query(ctx, `SELECT entity_type, count(*)::int, COALESCE(sum(cost_usd),0)::float8 FROM campaign_items WHERE campaign_id=$1 GROUP BY entity_type`, campaignID)
	if err != nil {
		return report, err
	}
	for rows.Next() {
		var kind string
		var count int
		var cost float64
		if err := rows.Scan(&kind, &count, &cost); err != nil {
			rows.Close()
			return report, err
		}
		report.LinkedByType[kind] = count
		report.SpendUSD += cost
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return report, err
	}
	rows.Close()
	report.BudgetRemainingUSD = math.Max(0, campaign.BudgetUSD-report.SpendUSD)

	// Populate the attributed items. The report declared an "items" field and
	// never filled it, so every API and MCP consumer saw an empty list while
	// linkedByType showed the real counts, and the web app had to issue a
	// second request per campaign to work around it.
	//
	// linkedByType and spendUsd above are exact (computed with GROUP BY over
	// every row); this list is capped, so treat it as a display sample rather
	// than the authoritative set for large campaigns.
	items, err := s.ListCampaignItems(ctx, projectID, campaignID, Page{Limit: campaignReportItemLimit})
	if err != nil {
		return report, err
	}
	report.Items = items

	err = s.pool.QueryRow(ctx, `
		SELECT
		  count(*) FILTER (WHERE ci.entity_type='content_asset' AND ca.status='published')::int,
		  count(*) FILTER (WHERE ci.entity_type='work_item' AND wi.status='done')::int,
		  count(*) FILTER (WHERE ci.entity_type='outreach' AND o.status IN ('sent','replied'))::int,
		  count(*) FILTER (WHERE ci.entity_type='outreach' AND o.status='replied')::int,
		  count(*) FILTER (WHERE ci.entity_type='submission')::int,
		  count(*) FILTER (WHERE ci.entity_type='submission' AND sub.status='accepted')::int,
		  count(*) FILTER (WHERE ci.entity_type='feedback' AND f.status='shipped')::int
		FROM campaign_items ci
		LEFT JOIN content_assets ca ON ci.entity_type='content_asset' AND ca.id=ci.entity_id
		LEFT JOIN work_items wi ON ci.entity_type='work_item' AND wi.id=ci.entity_id
		LEFT JOIN outreach o ON ci.entity_type='outreach' AND o.id=ci.entity_id
		LEFT JOIN submissions sub ON ci.entity_type='submission' AND sub.id=ci.entity_id
		LEFT JOIN feedback_items f ON ci.entity_type='feedback' AND f.id=ci.entity_id
		WHERE ci.campaign_id=$1`, campaignID).Scan(&report.PublishedContent, &report.CompletedWork, &report.OutreachSent,
		&report.OutreachReplies, &report.Submissions, &report.AcceptedTalks, &report.FeedbackShipped)
	if err != nil {
		return report, err
	}
	if report.OutreachSent > 0 {
		report.ReplyRate = float64(report.OutreachReplies) / float64(report.OutreachSent) * 100
	}
	if report.Submissions > 0 {
		report.AcceptanceRate = float64(report.AcceptedTalks) / float64(report.Submissions) * 100
	}

	metricRows, err := s.pool.Query(ctx, `
		SELECT id::text, campaign_id::text, metric_key, metric_value::float8, source, observed_at, metadata, created_at
		FROM campaign_metrics WHERE campaign_id=$1 ORDER BY observed_at DESC LIMIT 100`, campaignID)
	if err != nil {
		return report, err
	}
	for metricRows.Next() {
		var m campaigndomain.Metric
		var metadata []byte
		if err := metricRows.Scan(&m.ID, &m.CampaignID, &m.MetricKey, &m.MetricValue, &m.Source, &m.ObservedAt, &metadata, &m.CreatedAt); err != nil {
			metricRows.Close()
			return report, err
		}
		m.Metadata = map[string]any{}
		_ = json.Unmarshal(metadata, &m.Metadata)
		report.Metrics[m.MetricKey] += m.MetricValue
		if len(report.RecentMetrics) < 20 {
			report.RecentMetrics = append(report.RecentMetrics, m)
		}
	}
	if err := metricRows.Err(); err != nil {
		metricRows.Close()
		return report, err
	}
	metricRows.Close()

	score := math.Min(25, float64(report.PublishedContent*5)) + math.Min(20, float64(report.CompletedWork*4)) +
		math.Min(20, report.ReplyRate*0.2) + math.Min(20, report.AcceptanceRate*0.2) + math.Min(15, float64(report.FeedbackShipped*5))
	report.OutcomeScore = int(math.Round(math.Min(100, score)))
	return report, nil
}

func (s *Store) RelationshipRadar(ctx context.Context, projectID string) ([]campaigndomain.RelationshipRadarItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id::text, r.project_id::text, COALESCE(r.community_id::text,''), COALESCE(r.contact_id::text,''),
		       COALESCE(c.name, ct.name, 'Unknown relationship'),
		       CASE WHEN r.contact_id IS NOT NULL THEN 'contact' ELSE 'community' END,
		       r.stage, r.strength, r.last_touch_at, r.next_follow_up_at
		FROM relationships r
		LEFT JOIN communities c ON c.id=r.community_id
		LEFT JOIN contacts ct ON ct.id=r.contact_id
		WHERE r.project_id=$1
		ORDER BY r.next_follow_up_at NULLS LAST, r.strength DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	now := time.Now().UTC()
	items := make([]campaigndomain.RelationshipRadarItem, 0)
	for rows.Next() {
		var item campaigndomain.RelationshipRadarItem
		if err := rows.Scan(&item.RelationshipID, &item.ProjectID, &item.CommunityID, &item.ContactID, &item.Name, &item.Kind,
			&item.Stage, &item.Strength, &item.LastTouchAt, &item.NextFollowUpAt); err != nil {
			return nil, err
		}
		risk := (100 - item.Strength) / 2
		if item.LastTouchAt == nil {
			risk += 25
		} else {
			days := int(now.Sub(*item.LastTouchAt).Hours() / 24)
			if days < 0 {
				days = 0
			}
			item.DaysSinceTouch = &days
			if days > 60 {
				risk += 30
			} else if days > 30 {
				risk += 20
			} else if days > 14 {
				risk += 10
			}
		}
		if item.NextFollowUpAt != nil && item.NextFollowUpAt.Before(now) {
			risk += 30
		}
		if item.Stage == "dormant" {
			risk += 20
		}
		if risk > 100 {
			risk = 100
		}
		item.RiskScore = risk
		switch {
		case risk >= 70:
			item.Health = "critical"
		case risk >= 45:
			item.Health = "watch"
		default:
			item.Health = "healthy"
		}
		switch {
		case item.NextFollowUpAt != nil && item.NextFollowUpAt.Before(now):
			item.Recommended = "Follow up now"
		case item.DaysSinceTouch == nil || *item.DaysSinceTouch > 30:
			item.Recommended = "Reconnect with a useful update"
		case item.Stage == "warm" || item.Stage == "engaged":
			item.Recommended = "Create a concrete collaboration next step"
		case item.Stage == "partner":
			item.Recommended = "Maintain momentum and log the next touchpoint"
		default:
			item.Recommended = "Research and add a meaningful first touch"
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
