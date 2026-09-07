package storage

import (
	"context"
	"encoding/json"
	"time"

	feedbackdomain "github.com/savisaluwadana/DevRelOS/internal/domain/feedback"
)

type FeedbackFilter struct {
	Status    string
	Component string
	Limit     int
}

func (s *Store) ListFeedback(ctx context.Context, projectID string, filter FeedbackFilter) ([]feedbackdomain.Item, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, project_id::text, source_type, COALESCE(source_id::text,''), title, summary,
		       persona, component, impact_score, frequency_score, status, owner, github_repository,
		       github_issue_number, github_issue_url, github_issue_title, github_issue_body,
		       follow_up_note, shipped_at, metadata, created_at, updated_at
		FROM feedback_items
		WHERE project_id=$1
		  AND ($2='' OR status=$2)
		  AND ($3='' OR component=$3)
		ORDER BY CASE status
		           WHEN 'in_progress' THEN 0
		           WHEN 'planned' THEN 1
		           WHEN 'triaged' THEN 2
		           WHEN 'new' THEN 3
		           WHEN 'shipped' THEN 4
		           ELSE 5
		         END,
		         impact_score DESC,
		         frequency_score DESC,
		         updated_at DESC
		LIMIT $4`, projectID, filter.Status, filter.Component, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]feedbackdomain.Item, 0)
	for rows.Next() {
		item, err := scanFeedback(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetFeedback(ctx context.Context, projectID, id string) (feedbackdomain.Item, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id::text, project_id::text, source_type, COALESCE(source_id::text,''), title, summary,
		       persona, component, impact_score, frequency_score, status, owner, github_repository,
		       github_issue_number, github_issue_url, github_issue_title, github_issue_body,
		       follow_up_note, shipped_at, metadata, created_at, updated_at
		FROM feedback_items WHERE id=$1 AND project_id=$2`, id, projectID)
	return scanFeedback(row.Scan)
}

func (s *Store) CreateFeedback(ctx context.Context, item feedbackdomain.Item) (feedbackdomain.Item, error) {
	if item.SourceType == "" {
		item.SourceType = "manual"
	}
	if item.Status == "" {
		item.Status = "new"
	}
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	metadata, err := json.Marshal(item.Metadata)
	if err != nil {
		return item, err
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO feedback_items (
			project_id, source_type, source_id, title, summary, persona, component,
			impact_score, frequency_score, status, owner, github_repository, github_issue_number,
			github_issue_url, github_issue_title, github_issue_body, follow_up_note, shipped_at, metadata
		)
		VALUES ($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19::jsonb)
		RETURNING id::text, created_at, updated_at`,
		item.ProjectID, item.SourceType, item.SourceID, item.Title, item.Summary, item.Persona,
		item.Component, item.ImpactScore, item.FrequencyScore, item.Status, item.Owner,
		item.GitHubRepository, item.GitHubIssueNumber, item.GitHubIssueURL, item.GitHubIssueTitle,
		item.GitHubIssueBody, item.FollowUpNote, item.ShippedAt, metadata,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) UpdateFeedback(ctx context.Context, projectID, id string, update feedbackdomain.Update) (feedbackdomain.Item, error) {
	var shippedAt any
	if update.Status != nil && *update.Status == "shipped" {
		shippedAt = time.Now().UTC()
	}
	row := s.pool.QueryRow(ctx, `
		UPDATE feedback_items
		SET title=COALESCE($3,title),
		    summary=COALESCE($4,summary),
		    persona=COALESCE($5,persona),
		    component=COALESCE($6,component),
		    impact_score=COALESCE($7,impact_score),
		    frequency_score=COALESCE($8,frequency_score),
		    status=COALESCE($9,status),
		    owner=COALESCE($10,owner),
		    github_repository=COALESCE($11,github_repository),
		    github_issue_number=COALESCE($12,github_issue_number),
		    github_issue_url=COALESCE(NULLIF($13,''),github_issue_url),
		    github_issue_title=COALESCE($14,github_issue_title),
		    github_issue_body=COALESCE($15,github_issue_body),
		    follow_up_note=COALESCE($16,follow_up_note),
		    shipped_at=COALESCE($17,shipped_at),
		    updated_at=now()
		WHERE id=$1 AND project_id=$2
		RETURNING id::text, project_id::text, source_type, COALESCE(source_id::text,''), title, summary,
		          persona, component, impact_score, frequency_score, status, owner, github_repository,
		          github_issue_number, github_issue_url, github_issue_title, github_issue_body,
		          follow_up_note, shipped_at, metadata, created_at, updated_at`,
		id, projectID, update.Title, update.Summary, update.Persona, update.Component,
		update.ImpactScore, update.FrequencyScore, update.Status, update.Owner,
		update.GitHubRepository, update.GitHubIssueNumber, update.GitHubIssueURL,
		update.GitHubIssueTitle, update.GitHubIssueBody, update.FollowUpNote, shippedAt)
	return scanFeedback(row.Scan)
}

func (s *Store) FeedbackExistsForPainPoint(ctx context.Context, projectID, painPointID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM feedback_items
			WHERE project_id=$1 AND source_type='pain_point' AND source_id=$2::uuid
		)`, projectID, painPointID).Scan(&exists)
	return exists, err
}

type feedbackScanner func(dest ...any) error

func scanFeedback(scan feedbackScanner) (feedbackdomain.Item, error) {
	var item feedbackdomain.Item
	var metadata []byte
	err := scan(
		&item.ID, &item.ProjectID, &item.SourceType, &item.SourceID, &item.Title, &item.Summary,
		&item.Persona, &item.Component, &item.ImpactScore, &item.FrequencyScore, &item.Status,
		&item.Owner, &item.GitHubRepository, &item.GitHubIssueNumber, &item.GitHubIssueURL,
		&item.GitHubIssueTitle, &item.GitHubIssueBody, &item.FollowUpNote, &item.ShippedAt,
		&metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, err
	}
	item.Metadata = map[string]any{}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}
