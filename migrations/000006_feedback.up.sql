CREATE TABLE feedback_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  source_type TEXT NOT NULL DEFAULT 'manual' CHECK (source_type IN ('manual','pain_point','signal','work_item')),
  source_id UUID,
  title TEXT NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  persona TEXT NOT NULL DEFAULT '',
  component TEXT NOT NULL DEFAULT '',
  impact_score INTEGER NOT NULL DEFAULT 50 CHECK (impact_score BETWEEN 0 AND 100),
  frequency_score INTEGER NOT NULL DEFAULT 50 CHECK (frequency_score BETWEEN 0 AND 100),
  status TEXT NOT NULL DEFAULT 'new' CHECK (status IN ('new','triaged','planned','in_progress','shipped','closed','wont_fix')),
  owner TEXT NOT NULL DEFAULT '',
  github_repository TEXT NOT NULL DEFAULT '',
  github_issue_number INTEGER,
  github_issue_url TEXT NOT NULL DEFAULT '',
  github_issue_title TEXT NOT NULL DEFAULT '',
  github_issue_body TEXT NOT NULL DEFAULT '',
  follow_up_note TEXT NOT NULL DEFAULT '',
  shipped_at TIMESTAMPTZ,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX feedback_items_project_status_idx ON feedback_items(project_id, status, impact_score DESC, updated_at DESC);
CREATE INDEX feedback_items_source_idx ON feedback_items(project_id, source_type, source_id) WHERE source_id IS NOT NULL;
CREATE UNIQUE INDEX feedback_items_pain_point_uidx
  ON feedback_items(project_id, source_type, source_id)
  WHERE source_type='pain_point' AND source_id IS NOT NULL;
