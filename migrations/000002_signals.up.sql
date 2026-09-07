CREATE TABLE signals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  source_record_id UUID REFERENCES source_records(id) ON DELETE SET NULL,
  provider TEXT NOT NULL,
  external_id TEXT,
  canonical_url TEXT,
  author_handle TEXT,
  author_name TEXT,
  title TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL,
  occurred_at TIMESTAMPTZ,
  topics TEXT[] NOT NULL DEFAULT '{}',
  engagement_score INTEGER NOT NULL DEFAULT 0 CHECK (engagement_score >= 0),
  relevance_score INTEGER CHECK (relevance_score BETWEEN 0 AND 100),
  status TEXT NOT NULL DEFAULT 'new' CHECK (status IN ('new','reviewed','ignored','converted')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (project_id, provider, external_id)
);

CREATE INDEX signals_project_occurred_idx ON signals(project_id, occurred_at DESC);
CREATE INDEX signals_topics_gin_idx ON signals USING GIN(topics);
CREATE INDEX signals_status_idx ON signals(project_id, status);

CREATE TABLE pain_points (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  key TEXT NOT NULL,
  title TEXT NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  persona TEXT NOT NULL DEFAULT '',
  severity INTEGER NOT NULL DEFAULT 0 CHECK (severity BETWEEN 0 AND 100),
  trend_score INTEGER NOT NULL DEFAULT 0 CHECK (trend_score BETWEEN -100 AND 100),
  evidence_count INTEGER NOT NULL DEFAULT 0 CHECK (evidence_count >= 0),
  topics TEXT[] NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','watching','addressed','archived')),
  first_seen_at TIMESTAMPTZ,
  last_seen_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (project_id, key)
);

CREATE INDEX pain_points_project_score_idx ON pain_points(project_id, severity DESC, evidence_count DESC);
CREATE INDEX pain_points_topics_gin_idx ON pain_points USING GIN(topics);

CREATE TABLE pain_point_signals (
  pain_point_id UUID NOT NULL REFERENCES pain_points(id) ON DELETE CASCADE,
  signal_id UUID NOT NULL REFERENCES signals(id) ON DELETE CASCADE,
  weight NUMERIC(5,2) NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (pain_point_id, signal_id)
);
