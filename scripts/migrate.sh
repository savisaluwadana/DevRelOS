#!/bin/sh
set -eu

: "${DATABASE_URL:?DATABASE_URL is required}"

psql "$DATABASE_URL" -v ON_ERROR_STOP=1 <<'SQL'
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
SQL

for file in /migrations/*.up.sql; do
  [ -f "$file" ] || continue
  version=$(basename "$file" .up.sql)
  applied=$(psql "$DATABASE_URL" -Atqc "SELECT 1 FROM schema_migrations WHERE version='${version}'")
  if [ "$applied" = "1" ]; then
    echo "migration $version already applied"
    continue
  fi
  echo "applying migration $version"
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -1 -f "$file"
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -c "INSERT INTO schema_migrations(version) VALUES ('${version}')"
done
