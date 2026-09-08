-- Reverses 000010_outreach_delivery_retries.up.sql by restoring the
-- (status, queued_at) queue index defined in 000009 and dropping the
-- retry-scheduling column.

DROP INDEX IF EXISTS outreach_deliveries_queue_idx;

CREATE INDEX outreach_deliveries_queue_idx
  ON outreach_deliveries(status, queued_at)
  WHERE status IN ('queued','failed');

ALTER TABLE outreach_deliveries DROP COLUMN IF EXISTS next_attempt_at;
