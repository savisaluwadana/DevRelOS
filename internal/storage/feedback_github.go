package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Store) UpdateFeedbackGitHubSync(ctx context.Context, projectID, feedbackID, issueURL, issueTitle string, metadata map[string]any) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["githubIssueSyncedAt"] = time.Now().UTC().Format(time.RFC3339)
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	cmd, err := s.pool.Exec(ctx, `
		UPDATE feedback_items
		SET github_issue_url=$3,
		    github_issue_title=$4,
		    metadata=metadata || $5::jsonb,
		    updated_at=now()
		WHERE id=$1 AND project_id=$2`, feedbackID, projectID, issueURL, issueTitle, encoded)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
