CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE workspaces (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE projects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  slug TEXT NOT NULL,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  topics TEXT[] NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (workspace_id, slug)
);

CREATE TABLE connectors (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  provider TEXT NOT NULL,
  name TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT true,
  config JSONB NOT NULL DEFAULT '{}'::jsonb,
  policy JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE connector_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  connector_id UUID NOT NULL REFERENCES connectors(id) ON DELETE CASCADE,
  status TEXT NOT NULL CHECK (status IN ('queued','running','succeeded','failed','cancelled')),
  cursor TEXT,
  requests_made INTEGER NOT NULL DEFAULT 0,
  items_fetched INTEGER NOT NULL DEFAULT 0,
  items_created INTEGER NOT NULL DEFAULT 0,
  items_updated INTEGER NOT NULL DEFAULT 0,
  items_skipped INTEGER NOT NULL DEFAULT 0,
  provider_cost_usd NUMERIC(12,4),
  warnings JSONB NOT NULL DEFAULT '[]'::jsonb,
  error TEXT,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE source_records (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  provider TEXT NOT NULL,
  external_id TEXT,
  canonical_url TEXT,
  source_timestamp TIMESTAMPTZ,
  fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  content_hash TEXT,
  raw_payload JSONB,
  provenance JSONB NOT NULL DEFAULT '{}'::jsonb,
  UNIQUE (workspace_id, provider, external_id)
);

CREATE TABLE events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  source_record_id UUID REFERENCES source_records(id) ON DELETE SET NULL,
  name TEXT NOT NULL,
  slug TEXT,
  description TEXT NOT NULL DEFAULT '',
  website_url TEXT,
  city TEXT,
  country TEXT,
  timezone TEXT,
  starts_at TIMESTAMPTZ,
  ends_at TIMESTAMPTZ,
  event_type TEXT NOT NULL DEFAULT 'conference',
  topics TEXT[] NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'discovered' CHECK (status IN ('discovered','tracking','attending','completed','archived')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX events_project_start_idx ON events(project_id, starts_at);
CREATE INDEX events_topics_gin_idx ON events USING GIN(topics);

CREATE TABLE cfps (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  name TEXT NOT NULL DEFAULT 'Main CFP',
  submission_url TEXT,
  opens_at TIMESTAMPTZ,
  closes_at TIMESTAMPTZ,
  tracks TEXT[] NOT NULL DEFAULT '{}',
  requirements TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('upcoming','open','closed','cancelled')),
  fit_score INTEGER CHECK (fit_score BETWEEN 0 AND 100),
  score_reason JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX cfps_deadline_idx ON cfps(closes_at);

CREATE TABLE talks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  abstract TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  level TEXT NOT NULL DEFAULT 'intermediate',
  duration_minutes INTEGER NOT NULL DEFAULT 30,
  topics TEXT[] NOT NULL DEFAULT '{}',
  demo_url TEXT,
  slides_url TEXT,
  recording_url TEXT,
  status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','ready','retired')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE submissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  cfp_id UUID NOT NULL REFERENCES cfps(id) ON DELETE CASCADE,
  talk_id UUID NOT NULL REFERENCES talks(id) ON DELETE RESTRICT,
  title_override TEXT,
  abstract_override TEXT,
  status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','needs_work','ready','submitted','accepted','rejected','withdrawn')),
  submitted_at TIMESTAMPTZ,
  decision_at TIMESTAMPTZ,
  notes TEXT NOT NULL DEFAULT '',
  fit_score INTEGER CHECK (fit_score BETWEEN 0 AND 100),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (cfp_id, talk_id)
);

CREATE TABLE communities (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  source_record_id UUID REFERENCES source_records(id) ON DELETE SET NULL,
  name TEXT NOT NULL,
  platform TEXT NOT NULL DEFAULT 'manual',
  external_id TEXT,
  website_url TEXT,
  city TEXT,
  country TEXT,
  timezone TEXT,
  topics TEXT[] NOT NULL DEFAULT '{}',
  member_count INTEGER,
  activity_score INTEGER CHECK (activity_score BETWEEN 0 AND 100),
  speaking_fit_score INTEGER CHECK (speaking_fit_score BETWEEN 0 AND 100),
  last_event_at TIMESTAMPTZ,
  next_event_at TIMESTAMPTZ,
  status TEXT NOT NULL DEFAULT 'discovered' CHECK (status IN ('discovered','researching','warm','active','paused','do_not_contact')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (project_id, platform, external_id)
);

CREATE TABLE contacts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  role TEXT,
  email TEXT,
  public_profile_url TEXT,
  source_url TEXT,
  do_not_contact BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE community_contacts (
  community_id UUID NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
  contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
  role TEXT NOT NULL DEFAULT 'organizer',
  PRIMARY KEY (community_id, contact_id)
);

CREATE TABLE relationships (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  community_id UUID REFERENCES communities(id) ON DELETE CASCADE,
  contact_id UUID REFERENCES contacts(id) ON DELETE CASCADE,
  stage TEXT NOT NULL DEFAULT 'cold' CHECK (stage IN ('cold','warm','engaged','partner','dormant')),
  strength INTEGER NOT NULL DEFAULT 0 CHECK (strength BETWEEN 0 AND 100),
  last_touch_at TIMESTAMPTZ,
  next_follow_up_at TIMESTAMPTZ,
  notes TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (community_id IS NOT NULL OR contact_id IS NOT NULL)
);

CREATE TABLE touchpoints (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  relationship_id UUID NOT NULL REFERENCES relationships(id) ON DELETE CASCADE,
  channel TEXT NOT NULL,
  direction TEXT NOT NULL CHECK (direction IN ('inbound','outbound','internal')),
  summary TEXT NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE outreach (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  community_id UUID REFERENCES communities(id) ON DELETE SET NULL,
  contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,
  talk_id UUID REFERENCES talks(id) ON DELETE SET NULL,
  channel TEXT NOT NULL,
  subject TEXT,
  body TEXT NOT NULL,
  rationale TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','needs_approval','approved','queued','sent','replied','failed','cancelled')),
  approved_at TIMESTAMPTZ,
  sent_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE audit_log (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  actor TEXT NOT NULL,
  action TEXT NOT NULL,
  entity_type TEXT NOT NULL,
  entity_id UUID,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO workspaces (slug, name) VALUES ('default', 'Default Workspace');
INSERT INTO projects (workspace_id, slug, name, description, topics)
SELECT id, 'default', 'DevRel Program', 'Default DevRelOS project', ARRAY['developer-relations','cloud-native']
FROM workspaces WHERE slug = 'default';
