DROP INDEX IF EXISTS connectors_next_run_idx;
ALTER TABLE connectors DROP COLUMN IF EXISTS next_run_at;
ALTER TABLE connectors DROP COLUMN IF EXISTS schedule_minutes;
