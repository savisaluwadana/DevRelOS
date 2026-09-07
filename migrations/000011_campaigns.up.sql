CREATE TABLE campaigns (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  objective TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'planning' CHECK (status IN ('planning','active','paused','completed','archived')),
  starts_at TIMESTAMPTZ,
  ends_at TIMESTAMPTZ,
  budget_usd NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (budget_usd >= 0),
  target JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX campaigns_project_status_idx ON campaigns(project_id, status, updated_at DESC);

CREATE TABLE campaign_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  entity_type TEXT NOT NULL CHECK (entity_type IN ('content_asset','work_item','event','cfp','submission','outreach','feedback','media_asset','community','talk')),
  entity_id UUID NOT NULL,
  channel TEXT NOT NULL DEFAULT '',
  cost_usd NUMERIC(14,2) NOT NULL DEFAULT 0 CHECK (cost_usd >= 0),
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (campaign_id, entity_type, entity_id)
);

CREATE INDEX campaign_items_campaign_type_idx ON campaign_items(campaign_id, entity_type);

CREATE TABLE campaign_metrics (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  metric_key TEXT NOT NULL,
  metric_value NUMERIC(18,4) NOT NULL,
  source TEXT NOT NULL DEFAULT 'manual',
  observed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX campaign_metrics_campaign_key_idx ON campaign_metrics(campaign_id, metric_key, observed_at DESC);
