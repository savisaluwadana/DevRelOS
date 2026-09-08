package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	mediadomain "github.com/savisaluwadana/DevRelOS/internal/domain/media"
)

func (s *Store) ListMediaAssets(ctx context.Context, projectID string, limit int) ([]mediadomain.Asset, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, project_id::text, COALESCE(content_asset_id::text,''), title, source_path, source_url,
		       media_type, duration_ms, status, transcript_text, transcript_language, transcript_segments,
		       metadata, created_at, updated_at
		FROM media_assets WHERE project_id=$1 ORDER BY updated_at DESC LIMIT $2`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]mediadomain.Asset, 0)
	for rows.Next() {
		item, err := scanMediaAsset(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateMediaAsset(ctx context.Context, item mediadomain.Asset) (mediadomain.Asset, error) {
	if item.MediaType == "" {
		item.MediaType = "video"
	}
	if item.Status == "" {
		item.Status = "ready"
	}
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	metadata, err := json.Marshal(item.Metadata)
	if err != nil {
		return item, err
	}
	segments, err := json.Marshal(item.TranscriptSegments)
	if err != nil {
		return item, err
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO media_assets(project_id,content_asset_id,title,source_path,source_url,media_type,duration_ms,status,
		 transcript_text,transcript_language,transcript_segments,metadata)
		VALUES($1,NULLIF($2,'')::uuid,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12::jsonb)
		RETURNING id::text, created_at, updated_at`, item.ProjectID, item.ContentAssetID, item.Title, item.SourcePath,
		item.SourceURL, item.MediaType, item.DurationMS, item.Status, item.TranscriptText, item.TranscriptLanguage,
		segments, metadata).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) GetMediaAsset(ctx context.Context, projectID, id string) (mediadomain.Asset, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id::text, project_id::text, COALESCE(content_asset_id::text,''), title, source_path, source_url,
		       media_type, duration_ms, status, transcript_text, transcript_language, transcript_segments,
		       metadata, created_at, updated_at
		FROM media_assets WHERE project_id=$1 AND id=$2`, projectID, id)
	return scanMediaAsset(row.Scan)
}

func (s *Store) UpdateMediaTranscript(ctx context.Context, projectID, id string, update mediadomain.TranscriptUpdate) (mediadomain.Asset, error) {
	segments, err := json.Marshal(update.Segments)
	if err != nil {
		return mediadomain.Asset{}, err
	}
	row := s.pool.QueryRow(ctx, `
		UPDATE media_assets SET transcript_text=$3, transcript_language=$4, transcript_segments=$5::jsonb,
		 status='segmented', updated_at=now() WHERE project_id=$1 AND id=$2
		RETURNING id::text, project_id::text, COALESCE(content_asset_id::text,''), title, source_path, source_url,
		 media_type, duration_ms, status, transcript_text, transcript_language, transcript_segments, metadata, created_at, updated_at`,
		projectID, id, update.Text, update.Language, segments)
	return scanMediaAsset(row.Scan)
}

func (s *Store) ListMediaClips(ctx context.Context, projectID, mediaAssetID string, page Page) ([]mediadomain.Clip, error) {
	limit, limitArgs := page.clause(3)
	args := append([]any{projectID, mediaAssetID}, limitArgs...)
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, project_id::text, media_asset_id::text, COALESCE(content_asset_id::text,''), title,
		 start_ms,end_ms,aspect_ratio,score,rationale,caption_text,status,output_path,metadata,created_at,updated_at
		FROM media_clips WHERE project_id=$1 AND ($2='' OR media_asset_id=$2::uuid)
		ORDER BY score DESC, created_at DESC`+limit, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]mediadomain.Clip, 0)
	for rows.Next() {
		var item mediadomain.Clip
		var metadata []byte
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.MediaAssetID, &item.ContentAssetID, &item.Title, &item.StartMS, &item.EndMS,
			&item.AspectRatio, &item.Score, &item.Rationale, &item.CaptionText, &item.Status, &item.OutputPath, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Metadata = map[string]any{}
		_ = json.Unmarshal(metadata, &item.Metadata)
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetMediaClip looks one clip up by id, project-scoped.
//
// Both the render queue check and the worker previously listed every clip in the
// project and scanned in Go for a single id. That is a full-table read for a
// primary-key lookup, and it only stayed correct because the list query had no
// LIMIT - adding one would have made renders fail with "clip not found".
func (s *Store) GetMediaClip(ctx context.Context, projectID, clipID string) (*mediadomain.Clip, error) {
	var item mediadomain.Clip
	var metadata []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, project_id::text, media_asset_id::text, COALESCE(content_asset_id::text,''), title,
		 start_ms,end_ms,aspect_ratio,score,rationale,caption_text,status,output_path,metadata,created_at,updated_at
		FROM media_clips WHERE project_id=$1 AND id=$2`, projectID, clipID).
		Scan(&item.ID, &item.ProjectID, &item.MediaAssetID, &item.ContentAssetID, &item.Title, &item.StartMS, &item.EndMS,
			&item.AspectRatio, &item.Score, &item.Rationale, &item.CaptionText, &item.Status, &item.OutputPath,
			&metadata, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	item.Metadata = map[string]any{}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return &item, nil
}

func (s *Store) CreateMediaClip(ctx context.Context, item mediadomain.Clip) (mediadomain.Clip, error) {
	if item.AspectRatio == "" {
		item.AspectRatio = "9:16"
	}
	if item.Status == "" {
		item.Status = "candidate"
	}
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	metadata, err := json.Marshal(item.Metadata)
	if err != nil {
		return item, err
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO media_clips(project_id,media_asset_id,content_asset_id,title,start_ms,end_ms,aspect_ratio,score,rationale,caption_text,status,output_path,metadata)
		VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb)
		RETURNING id::text,created_at,updated_at`, item.ProjectID, item.MediaAssetID, item.ContentAssetID, item.Title, item.StartMS, item.EndMS,
		item.AspectRatio, item.Score, item.Rationale, item.CaptionText, item.Status, item.OutputPath, metadata).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (s *Store) UpdateMediaClipStatus(ctx context.Context, projectID, id, status string) error {
	cmd, err := s.pool.Exec(ctx, `UPDATE media_clips SET status=$3,updated_at=now() WHERE project_id=$1 AND id=$2`, projectID, id, status)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) QueueMediaRender(ctx context.Context, projectID, clipID string) (mediadomain.Job, error) {
	var job mediadomain.Job
	err := s.pool.QueryRow(ctx, `
		INSERT INTO media_jobs(project_id,media_asset_id,clip_id,job_type,status)
		SELECT $1,c.media_asset_id,c.id,'render','queued' FROM media_clips c WHERE c.id=$2 AND c.project_id=$1
		RETURNING id::text,project_id::text,media_asset_id::text,clip_id::text,job_type,status,attempts,error,payload,started_at,finished_at,created_at,updated_at`,
		projectID, clipID).Scan(&job.ID, &job.ProjectID, &job.MediaAssetID, &job.ClipID, &job.JobType, &job.Status, &job.Attempts, &job.Error,
		&job.Payload, &job.StartedAt, &job.FinishedAt, &job.CreatedAt, &job.UpdatedAt)
	return job, err
}

func (s *Store) ClaimNextMediaJob(ctx context.Context) (*mediadomain.Job, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var job mediadomain.Job
	var payload []byte
	err = tx.QueryRow(ctx, `
		SELECT id::text,project_id::text,media_asset_id::text,COALESCE(clip_id::text,''),job_type,status,attempts,error,payload,started_at,finished_at,created_at,updated_at
		FROM media_jobs WHERE status='queued' ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1`).Scan(&job.ID, &job.ProjectID, &job.MediaAssetID, &job.ClipID, &job.JobType, &job.Status, &job.Attempts, &job.Error, &payload, &job.StartedAt, &job.FinishedAt, &job.CreatedAt, &job.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, tx.Commit(ctx)
	}
	if err != nil {
		return nil, err
	}
	job.Payload = map[string]any{}
	_ = json.Unmarshal(payload, &job.Payload)
	now := time.Now().UTC()
	job.StartedAt = &now
	job.Status = "running"
	job.Attempts++
	if _, err = tx.Exec(ctx, `UPDATE media_jobs SET status='running',attempts=attempts+1,started_at=$2,updated_at=now() WHERE id=$1`, job.ID, now); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *Store) FinishMediaJob(ctx context.Context, job mediadomain.Job, outputPath string) error {
	now := time.Now().UTC()
	job.FinishedAt = &now
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `UPDATE media_jobs SET status=$2,error=$3,finished_at=$4,updated_at=now() WHERE id=$1`, job.ID, job.Status, job.Error, now); err != nil {
		return err
	}
	if job.ClipID != "" {
		clipStatus := "failed"
		if job.Status == "succeeded" {
			clipStatus = "rendered"
		}
		if _, err = tx.Exec(ctx, `UPDATE media_clips SET status=$2,output_path=CASE WHEN $3<>'' THEN $3 ELSE output_path END,updated_at=now() WHERE id=$1`, job.ClipID, clipStatus, outputPath); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

type mediaAssetScanner func(dest ...any) error

func scanMediaAsset(scan mediaAssetScanner) (mediadomain.Asset, error) {
	var item mediadomain.Asset
	var segments, metadata []byte
	err := scan(&item.ID, &item.ProjectID, &item.ContentAssetID, &item.Title, &item.SourcePath, &item.SourceURL, &item.MediaType, &item.DurationMS,
		&item.Status, &item.TranscriptText, &item.TranscriptLanguage, &segments, &metadata, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, err
	}
	item.TranscriptSegments = []mediadomain.Segment{}
	item.Metadata = map[string]any{}
	_ = json.Unmarshal(segments, &item.TranscriptSegments)
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}
