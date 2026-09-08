-- Reverses 000009_production_completion.up.sql.
-- Drop in reverse dependency order: outreach_deliveries and the connectors
-- foreign key first, then the tables they reference.

DROP TABLE IF EXISTS outreach_deliveries;

DROP INDEX IF EXISTS workspace_invitations_active_email_idx;
DROP INDEX IF EXISTS workspace_invitations_workspace_idx;
DROP TABLE IF EXISTS workspace_invitations;

DROP INDEX IF EXISTS user_sessions_active_idx;
DROP TABLE IF EXISTS user_sessions;

DROP INDEX IF EXISTS connectors_secret_idx;
ALTER TABLE connectors DROP COLUMN IF EXISTS secret_id;

DROP TABLE IF EXISTS connector_secrets;
