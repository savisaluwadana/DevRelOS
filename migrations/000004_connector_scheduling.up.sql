ALTER TABLE connectors
  ADD COLUMN schedule_minutes INTEGER CHECK (schedule_minutes IS NULL OR schedule_minutes BETWEEN 15 AND 10080),
  ADD COLUMN next_run_at TIMESTAMPTZ;

CREATE INDEX connectors_next_run_idx
  ON connectors(next_run_at)
  WHERE enabled = true AND schedule_minutes IS NOT NULL;
