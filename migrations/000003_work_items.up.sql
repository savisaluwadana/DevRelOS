CREATE TABLE work_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  source_type TEXT NOT NULL DEFAULT 'manual' CHECK (source_type IN ('manual','pain_point','signal','community','cfp','outreach','campaign')),
  source_id UUID,
  kind TEXT NOT NULL CHECK (kind IN ('content_brief','docs_improvement','product_feedback','talk_idea','community_research','outreach_follow_up','event_task','engineering_task')),
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  priority INTEGER NOT NULL DEFAULT 50 CHECK (priority BETWEEN 0 AND 100),
  status TEXT NOT NULL DEFAULT 'backlog' CHECK (status IN ('backlog','planned','in_progress','blocked','done','cancelled')),
  owner TEXT NOT NULL DEFAULT '',
  due_at TIMESTAMPTZ,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX work_items_project_status_idx ON work_items(project_id, status, priority DESC);
CREATE INDEX work_items_due_idx ON work_items(project_id, due_at) WHERE due_at IS NOT NULL;
CREATE INDEX work_items_source_idx ON work_items(project_id, source_type, source_id);
CREATE INDEX work_items_kind_idx ON work_items(project_id, kind);
CREATE UNIQUE INDEX work_items_source_kind_unique_idx
  ON work_items(project_id, source_type, source_id, kind)
  WHERE source_id IS NOT NULL;
