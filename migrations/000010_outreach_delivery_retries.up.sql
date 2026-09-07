ALTER TABLE outreach_deliveries
  ADD COLUMN next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now();

DROP INDEX IF EXISTS outreach_deliveries_queue_idx;

CREATE INDEX outreach_deliveries_queue_idx
  ON outreach_deliveries(next_attempt_at, queued_at)
  WHERE status IN ('queued','failed');
