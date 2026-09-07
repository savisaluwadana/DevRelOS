CREATE TABLE content_assets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  work_item_id UUID REFERENCES work_items(id) ON DELETE SET NULL,
  channel TEXT NOT NULL CHECK (channel IN ('blog','linkedin','x','newsletter','youtube','short_video','docs','talk','community')),
  format TEXT NOT NULL CHECK (format IN ('article','social_post','thread','newsletter','tutorial','video_script','short_script','documentation','talk_outline','community_post')),
  title TEXT NOT NULL,
  audience TEXT NOT NULL DEFAULT 'developers',
  objective TEXT NOT NULL DEFAULT '',
  brief TEXT NOT NULL DEFAULT '',
  draft TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'brief' CHECK (status IN ('brief','drafting','review','approved','published','archived')),
  topics TEXT[] NOT NULL DEFAULT '{}',
  source_url TEXT,
  published_url TEXT,
  scheduled_at TIMESTAMPTZ,
  published_at TIMESTAMPTZ,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX content_assets_project_status_idx ON content_assets(project_id, status, updated_at DESC);
CREATE INDEX content_assets_work_item_idx ON content_assets(work_item_id) WHERE work_item_id IS NOT NULL;
CREATE INDEX content_assets_topics_gin_idx ON content_assets USING GIN(topics);

CREATE UNIQUE INDEX content_assets_work_channel_format_uidx
  ON content_assets(work_item_id, channel, format)
  WHERE work_item_id IS NOT NULL;
