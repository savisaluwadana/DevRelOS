#!/bin/sh
set -eu

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.production.yml}"
BACKUP_ROOT="${BACKUP_ROOT:-./backups}"
POSTGRES_DB="${POSTGRES_DB:-devrelos}"
POSTGRES_USER="${POSTGRES_USER:-devrelos}"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
TARGET="$BACKUP_ROOT/$STAMP"

mkdir -p "$TARGET"

echo "Backing up PostgreSQL to $TARGET/database.dump"
docker compose -f "$COMPOSE_FILE" exec -T postgres \
  pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc > "$TARGET/database.dump"

test -s "$TARGET/database.dump" || { echo "database backup is empty" >&2; exit 1; }

echo "Backing up media volume to $TARGET/media.tar.gz"
docker compose -f "$COMPOSE_FILE" run --rm --no-deps \
  -v "$TARGET:/backup" \
  --entrypoint /bin/sh worker \
  -c 'cd /data/media && tar -czf /backup/media.tar.gz .'

test -s "$TARGET/media.tar.gz" || { echo "media backup is empty" >&2; exit 1; }

cat > "$TARGET/manifest.txt" <<EOF
created_at=$STAMP
compose_file=$COMPOSE_FILE
postgres_db=$POSTGRES_DB
postgres_user=$POSTGRES_USER
EOF

sha256sum "$TARGET/database.dump" "$TARGET/media.tar.gz" > "$TARGET/SHA256SUMS"
echo "Backup complete: $TARGET"
