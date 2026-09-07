CREATE TABLE media_assets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  content_asset_id UUID REFERENCES content_assets(id) ON DELETE SET NULL,
  title TEXT NOT NULL,
  source_path TEXT NOT NULL DEFAULT '',
  source_url TEXT NOT NULL DEFAULT '',
  media_type TEXT NOT NULL DEFAULT 'video' CHECK (media_type IN ('video','audio')),
  duration_ms BIGINT NOT NULL DEFAULT 0 CHECK (duration_ms >= 0),
  status TEXT NOT NULL DEFAULT 'ready' CHECK (status IN ('ready','transcribing','segmented','rendering','completed','failed','archived')),
  transcript_text TEXT NOT NULL DEFAULT '',
  transcript_language TEXT NOT NULL DEFAULT '',
  transcript_segments JSONB NOT NULL DEFAULT '[]'::jsonb,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (source_path <> '' OR source_url <> '')
);

CREATE INDEX media_assets_project_status_idx ON media_assets(project_id, status, updated_at DESC);
CREATE INDEX media_assets_content_idx ON media_assets(content_asset_id) WHERE content_asset_id IS NOT NULL;

CREATE TABLE media_clips (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  media_asset_id UUID NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
  content_asset_id UUID REFERENCES content_assets(id) ON DELETE SET NULL,
  title TEXT NOT NULL,
  start_ms BIGINT NOT NULL CHECK (start_ms >= 0),
  end_ms BIGINT NOT NULL CHECK (end_ms > start_ms),
  aspect_ratio TEXT NOT NULL DEFAULT '9:16' CHECK (aspect_ratio IN ('9:16','1:1','16:9')),
  score INTEGER NOT NULL DEFAULT 50 CHECK (score BETWEEN 0 AND 100),
  rationale TEXT NOT NULL DEFAULT '',
  caption_text TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'candidate' CHECK (status IN ('candidate','approved','queued','rendering','rendered','rejected','failed')),
  output_path TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX media_clips_asset_status_idx ON media_clips(media_asset_id, status, score DESC);

CREATE TABLE media_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  media_asset_id UUID NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
  clip_id UUID REFERENCES media_clips(id) ON DELETE CASCADE,
  job_type TEXT NOT NULL CHECK (job_type IN ('transcribe','segment','render')),
  status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued','running','succeeded','failed','cancelled')),
  attempts INTEGER NOT NULL DEFAULT 0,
  error TEXT NOT NULL DEFAULT '',
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX media_jobs_queue_idx ON media_jobs(status, created_at) WHERE status IN ('queued','running');
CREATE UNIQUE INDEX media_jobs_active_clip_render_uidx ON media_jobs(clip_id, job_type)
  WHERE clip_id IS NOT NULL AND job_type='render' AND status IN ('queued','running');
