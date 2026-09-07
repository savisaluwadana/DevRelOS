# Running DevRelOS

DevRelOS supports two deployment profiles:

- **local/self-hosted development** with `docker-compose.yml`;
- **single-node production** with `docker-compose.production.yml` and Caddy-managed HTTPS.

## Local stack

Prerequisites:

- Docker with Compose v2;
- port 3000 available;
- ports 5432 and 8080 available on localhost if direct database/API access is useful.

```bash
cp .env.example .env
make up
```

The stack starts PostgreSQL, applies every unapplied migration, starts the Go API, starts the Go worker with FFmpeg, and starts the Next.js console.

Open:

- Web: http://localhost:3000
- API liveness: http://localhost:8080/healthz
- API readiness: http://localhost:8080/readyz

Useful commands:

```bash
make ps
make logs
make down
```

The local Compose file binds PostgreSQL and the API to `127.0.0.1`; only the web UI is intended for normal operator use.

## Production stack

The production profile is intentionally stricter. It requires a public DNS name plus real database, operator and encryption secrets. PostgreSQL and the Go API have no host ports; Caddy is the only public entry point and automatically obtains/renews TLS certificates.

Create a deployment `.env` containing at least:

```text
DEVRELOS_DOMAIN=devrel.example.com
POSTGRES_DB=devrelos
POSTGRES_USER=devrelos
POSTGRES_PASSWORD=<long random database password>
DEVRELOS_API_TOKEN=<long random break-glass/operator token>
DEVRELOS_SECRET_KEY=<base64 32-byte key>
```

Generate the encryption key with:

```bash
openssl rand -base64 32
```

The production profile automatically enables required API authentication, browser sessions, Secure cookies, request rate limiting and trusted-proxy handling.

Validate configuration before deployment:

```bash
make prod-config
```

Start:

```bash
make prod-up
```

Logs and shutdown:

```bash
make prod-logs
make prod-down
```

DNS for `DEVRELOS_DOMAIN` must point to the host and ports 80/443 must be reachable so Caddy can provision HTTPS.

## Bootstrap identity

`DEVRELOS_API_TOKEN` is the root/bootstrap credential. Treat it like a break-glass secret rather than a normal user credential.

Use it to provision the first `owner` from **Access & Security** or the API, then mint a `drk_...` API key for that owner. User API keys are stored only as SHA-256 hashes and can be revoked or expired.

When browser sessions are enabled, `/login` accepts a `drk_...` key once and exchanges it for a dedicated short-lived `ds_...` session. Only the `ds_` session is stored in the HttpOnly, SameSite=Strict browser cookie. Production cookies are also Secure.

Sign-out revokes the server-side session. A revoked/expired session fails the next protected request.

## Workspace invitations and switching

Owners/admins can create one-time `di_...` workspace invitations from **Access & Security**. The invitation link is shown once and can be revoked before acceptance. Invitations expire and carry a specific workspace role.

Accepted invitations create or attach the user membership and issue a short-lived browser session immediately.

Users with access to multiple workspaces can switch explicitly from the console. Dedicated sessions are pinned to one active workspace; manually changing `workspaceId` or `projectId` cannot escape that workspace boundary.

## Encrypted connector credentials

Set `DEVRELOS_SECRET_KEY` before using encrypted workspace secrets. The key must decode to exactly 32 bytes and must remain stable for the life of the stored secrets.

In **Access & Security**:

1. create an encrypted secret for a provider;
2. attach it to a compatible connector;
3. rotate the secret when required.

DevRelOS encrypts secret values with AES-256-GCM before PostgreSQL storage. List APIs return metadata only—not plaintext, ciphertext or nonces. The worker decrypts an attached value only in memory for the active connector run.

The older `token_env`/`GITHUB_TOKEN` path remains available for backwards compatibility, but encrypted workspace secrets are the preferred production path.

## API rate limiting and metrics

API rate limiting defaults to 120 requests/minute per bearer identity or client IP:

```text
DEVRELOS_RATE_LIMIT_ENABLED=true
DEVRELOS_RATE_LIMIT_RPM=120
```

`DEVRELOS_TRUST_PROXY=true` should only be enabled behind a trusted reverse proxy. The production Compose profile enables it because only Caddy can reach the API network.

Every API response receives an `X-Request-ID`. API logs contain structured request fields including request ID, route pattern, status, bytes and duration.

`GET /metrics` exposes authenticated Prometheus-compatible process/request metrics. It is not public; scrape it with an operator/user bearer that has workspace access.

## Approval-gated email delivery

Email outreach remains explicitly human-approved. DevRelOS never turns a draft directly into an external message.

Enable SMTP only after configuring it:

```text
DEVRELOS_SMTP_ENABLED=true
DEVRELOS_SMTP_HOST=smtp.example.com
DEVRELOS_SMTP_PORT=587
DEVRELOS_SMTP_USERNAME=...
DEVRELOS_SMTP_PASSWORD=...
DEVRELOS_SMTP_FROM=devrel@example.com
DEVRELOS_SMTP_FROM_NAME=DevRel Team
DEVRELOS_SMTP_STARTTLS=true
DEVRELOS_SMTP_REQUIRE_TLS=true
DEVRELOS_OUTREACH_MAX_ATTEMPTS=5
```

The flow is:

`draft -> needs_approval -> approved -> queued -> sent|failed`

For email, only a successful SMTP delivery can mark the outreach `sent`. Delivery records are unique per outreach item, workers claim with `SKIP LOCKED`, failures are recorded, and retries use bounded exponential backoff. A do-not-contact contact is rejected both when queueing and when claiming a delivery.

Other outreach channels remain manual/human-executed rather than simulated automated sends.

## Media Studio

Local Compose mounts `./data/media`; production Compose uses the durable `devrelos-media` Docker volume. FFmpeg source and output paths are constrained to the configured media root.

For local development place recordings under:

```text
data/media/recordings/
```

and reference them with paths relative to the media root, for example `recordings/community-call.mp4`.

The production profile is a supported **single-node** topology: PostgreSQL plus the media volume must live on persistent storage and be backed up. A multi-host/HA deployment would require a shared/object-storage media adapter and is outside this topology.

## Backup and restore

Production backup captures PostgreSQL plus the media volume and writes SHA-256 checksums:

```bash
make backup
```

Backups are written under `./backups/<UTC timestamp>/` unless `BACKUP_ROOT` is overridden.

Restore is destructive and requires an explicit backup path:

```bash
make restore BACKUP=./backups/20260907T120000Z
```

The restore script verifies checksums when present, stops application services, restores PostgreSQL and the media volume, then starts the application again. Test restore procedures regularly and copy backups off-host.

## MCP

The MCP server remains a local stdio process and talks only to the authenticated DevRelOS domain API. Prefer a dedicated user/API key with the minimum workspace role the agent needs:

```bash
DEVRELOS_API_URL=http://localhost:8080 \
DEVRELOS_API_TOKEN="drk_..." \
go run ./services/mcp
```

See [MCP.md](MCP.md) for the tool catalog.

## Native local development

```bash
make db-up
export DATABASE_URL='postgres://devrelos:devrelos@localhost:5432/devrelos?sslmode=disable'
make migrate
```

Then use separate terminals:

```bash
make api
make worker
make web
```

`NEXT_PUBLIC_API_URL` should remain `/api/devrelos` so browser mutations continue through the same-origin Next.js proxy.

## Supported production boundary

The supported v1 production topology is a **single-node, HTTPS-terminated, authenticated self-hosted deployment** with durable PostgreSQL and media storage, workspace RBAC, dedicated short-lived browser sessions, encrypted connector secrets, audit events, rate limiting, health/readiness probes, metrics, request IDs, backup/restore, scheduled workers, approval-gated SMTP delivery and Caddy TLS.

OIDC/SSO, multi-region/high-availability storage and provider APIs that require separate commercial approval (for example Reddit/X) are optional integrations/extensions rather than prerequisites for this topology. DevRelOS intentionally does not replace them with unauthorized scraping or unsafe automation.
