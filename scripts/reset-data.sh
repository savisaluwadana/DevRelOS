#!/bin/sh
set -eu
: "${DATABASE_URL:?DATABASE_URL is required}"

# Clears DevRel content while preserving the workspace/project foundation,
# connector configuration and identity/access records. Use after a smoke run
# or demo to get back to an empty operator console without re-migrating.
#
# Preserved: workspaces, projects, connectors, connector_secrets, users,
# workspace_memberships, workspace_invitations, api_keys, user_sessions,
# schema_migrations.
if [ "${DEVRELOS_CONFIRM_RESET:-}" != "YES" ]; then
  echo "Reset is destructive and deletes ingested signals. Set DEVRELOS_CONFIRM_RESET=YES to continue." >&2
  echo "Take a backup first: pg_dump \"\$DATABASE_URL\" -Fc > database.dump" >&2
  exit 2
fi

# One TRUNCATE so foreign keys between these tables never see a partial state.
# RESTART IDENTITY is harmless here (ids are UUIDs) but keeps the reset total
# if a sequence-backed column is ever added.
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -1 <<'SQL'
TRUNCATE TABLE
  pain_point_signals,
  pain_points,
  signals,
  source_records,
  connector_runs,
  submissions,
  talks,
  cfps,
  events,
  touchpoints,
  relationships,
  community_contacts,
  contacts,
  communities,
  campaign_metrics,
  campaign_items,
  campaigns,
  outreach_deliveries,
  outreach,
  media_jobs,
  media_clips,
  media_assets,
  feedback_items,
  content_assets,
  work_items,
  audit_events,
  audit_log
RESTART IDENTITY;
SQL

echo "Reset complete"
