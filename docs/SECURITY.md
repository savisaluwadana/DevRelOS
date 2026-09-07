# DevRelOS security model

DevRelOS supports authenticated self-hosted operation with users, workspace RBAC, revocable API keys, dedicated short-lived browser sessions, encrypted connector secrets, audit events and a hardened HTTPS production profile.

## Root/operator credential

Production requires:

```text
DEVRELOS_API_TOKEN=<long random secret>
DEVRELOS_REQUIRE_AUTH=true
```

The static operator token is a bootstrap/break-glass credential with broad access. Do not use it as a normal browser/user credential and do not put it in `NEXT_PUBLIC_*`, source control, screenshots, logs, localStorage or connector JSON.

## User API keys

User API keys use the `drk_` prefix. Plaintext is returned only once at creation; PostgreSQL stores only a SHA-256 hash. Keys may expire, can be revoked, and record last-used timestamps.

API keys are appropriate for CLI/MCP/automation access and as the one-time browser-session bootstrap credential.

## Dedicated browser sessions

Browser login exchanges a valid `drk_` key for a separate `ds_` session token. Only the hash of the `ds_` token is stored in PostgreSQL and only the `ds_` token is placed in the HttpOnly, SameSite=Strict browser cookie.

Production uses `Secure` cookies. Sessions have a bounded lifetime, can be revoked, and update last-seen metadata. Sign-out revokes the database session before clearing the cookie.

The Next.js edge validates the session against the Go API on protected requests. Revoked, expired, disabled-user or invalid-membership sessions are rejected.

## Workspace isolation

Roles are:

- `owner` — full workspace administration and owner grants;
- `admin` — security/membership administration plus normal writes;
- `editor` — normal workflow writes;
- `viewer` — read only.

A dedicated browser session is pinned to exactly one active workspace. Explicit workspace switching updates that server-side session after verifying membership. Query-string manipulation of `workspaceId` or `projectId` cannot move a session outside its workspace.

When no `projectId` is provided, the API resolves the default project **inside the active workspace**, not a global default project.

Linked writes (CFPs, submissions, connector operations and other scoped resources) verify ownership server-side rather than trusting body IDs.

## Invitations

Workspace invitations use one-time `di_` tokens stored only as hashes. Invitations have expiry and role metadata and can be revoked before use. Only owners can invite another owner.

Invitation acceptance atomically creates/attaches the user membership and issues a dedicated browser session. The invitation token is shown only at creation and should be transmitted through an appropriate private channel.

## Connector secrets

Set a stable 32-byte master key through:

```text
DEVRELOS_SECRET_KEY=<base64 32-byte key>
```

Generate it with `openssl rand -base64 32` and store it in the deployment secret manager. Losing this key makes encrypted connector values unrecoverable; leaking it compromises connector credentials.

Secret values are encrypted with AES-256-GCM before database storage. The associated authenticated data includes workspace, provider, secret name and key version. List/read management APIs return metadata only and never return plaintext, ciphertext or nonces.

A connector stores only a `secretId`. Cross-workspace/provider attachment is rejected. Secret deletion is blocked while still referenced. Rotation creates a new key version without changing the connector reference.

Workers decrypt attached connector credentials only in memory immediately before provider validation/fetch and remove the injected plaintext after the run path completes. Legacy environment-variable credentials remain supported but encrypted workspace secrets are preferred.

## Browser mutation boundary

Browser mutations use the same-origin `/api/devrelos/...` proxy. In session mode the proxy forwards the signed-in user's `ds_` token and never falls back to the operator token.

The proxy rejects cross-site mutation origins and emits strict browser headers. The Go API remains authoritative for RBAC.

## Request protection

The API applies:

- bounded request-body decoding;
- server read-header/read/write/idle timeouts;
- 1 MiB maximum headers;
- configurable per-client rate limiting;
- CORS limited to the configured web origin;
- no-store response caching;
- anti-framing/content-type/referrer headers;
- generated or validated `X-Request-ID` values.

Rate limiting is per process and keyed by a hash of the bearer credential when available, otherwise by client IP. `X-Forwarded-For` is trusted only when `DEVRELOS_TRUST_PROXY=true`; production Compose sets this because the API is isolated behind Caddy.

## Metrics and logs

`/healthz` and `/readyz` are public probes. `/metrics` is authenticated.

The API exports Prometheus-compatible uptime, active-request, request-count and duration-sum metrics with bounded route-pattern labels. Request logs include request ID, method, path, route pattern, status, bytes and duration.

Do not add raw credentials, message bodies or connector secret values to operational logs.

## Outreach delivery

Outbound email is approval-gated. An email draft cannot be marked `sent` by a browser/API status edit; only successful SMTP delivery by the worker may set that state.

Delivery jobs are durable and unique per outreach item. Workers claim them with PostgreSQL `SKIP LOCKED`, re-check do-not-contact state, record failures and retry with bounded exponential backoff. SMTP authentication is refused without TLS when TLS is required, and message headers are sanitized against CR/LF injection.

Non-email channels remain human/manual unless a future connector implements an explicitly authorized delivery API.

## Media security

FFmpeg jobs resolve local paths inside configured media roots; path traversal outside those roots is rejected. The base renderer does not fetch arbitrary remote media URLs.

The supported production topology uses a persistent local Docker media volume. Public upload surfaces would need additional upload quotas, MIME validation and malware/resource controls before accepting arbitrary untrusted uploads.

## Network topology and TLS

`docker-compose.production.yml` creates an internal backend network for PostgreSQL, API and worker services. Caddy is the only public service and exposes 80/443. It provides automatic TLS, HSTS and edge security headers before proxying to the Next.js web application.

Production startup requires database, API and encryption secrets instead of silently accepting demo defaults.

## Backup and recovery

`scripts/backup.sh` captures PostgreSQL plus media data and writes checksums. `scripts/restore.sh` requires explicit destructive confirmation, verifies checksums when available, restores database/media, and restarts application services.

Keep backups encrypted/off-host according to your environment, restrict access to them, and test restoration periodically. A backup containing the database does not replace protection of `DEVRELOS_SECRET_KEY`, which must also be recoverable through the deployment secret-management process.

## Audit events

Identity, membership, API-key, session, invitation and connector-secret management actions append workspace-scoped audit events. Additional high-risk side effects should continue to append audit records as integrations are added.

## Supported production boundary

The supported v1 target is a single-node, HTTPS-terminated, authenticated self-hosted deployment. High-availability/multi-host media requires a shared/object-storage adapter, and enterprise SSO/OIDC may be added for environments that require centralized identity. These are deployment extensions rather than reasons to weaken the current session/RBAC model.

Provider APIs that require separate approval/licensing (such as Reddit or X) must be integrated only under their official terms; DevRelOS should not substitute unauthorized scraping.
