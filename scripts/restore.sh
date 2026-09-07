#!/bin/sh
set -eu

if [ "$#" -ne 1 ]; then
  echo "Usage: DEVRELOS_CONFIRM_RESTORE=YES $0 <backup-directory>" >&2
  exit 2
fi
if [ "${DEVRELOS_CONFIRM_RESTORE:-}" != "YES" ]; then
  echo "Restore is destructive. Set DEVRELOS_CONFIRM_RESTORE=YES to continue." >&2
  exit 2
fi

SOURCE="$1"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.production.yml}"
POSTGRES_DB="${POSTGRES_DB:-devrelos}"
POSTGRES_USER="${POSTGRES_USER:-devrelos}"

test -s "$SOURCE/database.dump" || { echo "missing database.dump" >&2; exit 1; }
test -s "$SOURCE/media.tar.gz" || { echo "missing media.tar.gz" >&2; exit 1; }
if [ -f "$SOURCE/SHA256SUMS" ]; then
  (cd "$SOURCE" && sha256sum -c SHA256SUMS)
fi

SOURCE_ABS="$(cd "$SOURCE" && pwd)"

echo "Stopping application services before restore"
docker compose -f "$COMPOSE_FILE" stop api worker web caddy >/dev/null 2>&1 || true

echo "Restoring PostgreSQL"
docker compose -f "$COMPOSE_FILE" exec -T postgres \
  pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" --clean --if-exists --no-owner --no-privileges < "$SOURCE/database.dump"

echo "Restoring media volume"
docker compose -f "$COMPOSE_FILE" run --rm --no-deps \
  -v "$SOURCE_ABS:/restore:ro" \
  --entrypoint /bin/sh worker \
  -c 'find /data/media -mindepth 1 -maxdepth 1 -exec rm -rf {} + && tar -xzf /restore/media.tar.gz -C /data/media'

echo "Starting application services"
docker compose -f "$COMPOSE_FILE" up -d api worker web caddy

echo "Restore complete"
