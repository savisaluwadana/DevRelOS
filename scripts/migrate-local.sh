#!/bin/sh
set -eu
: "${DATABASE_URL:?DATABASE_URL is required}"

psql "$DATABASE_URL" -v ON_ERROR_STOP=1 <<'SQL'
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
SQL

for file in migrations/*.up.sql; do
  [ -f "$file" ] || continue
  version=$(basename "$file" .up.sql)
  applied=$(psql "$DATABASE_URL" -Atqc "SELECT 1 FROM schema_migrations WHERE version='${version}'")
  if [ "$applied" = "1" ]; then
    echo "migration $version already applied"
    continue
  fi
  echo "applying migration $version"
  # Apply the migration and record it in ONE transaction. Running these as two
  # psql invocations left a window where a crash in between applied the schema
  # change without recording it, so the next run re-applied it and failed on
  # CREATE TABLE / ADD COLUMN. Same fix as scripts/migrate.sh.
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -1 <<SQL
\i ${file}
INSERT INTO schema_migrations(version) VALUES ('${version}');
SQL
done
