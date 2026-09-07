# DevRelOS security model

DevRelOS now supports authenticated self-hosted operation with first-class users, workspace RBAC, revocable API keys and per-user browser sessions.

## API authentication

Set both:

```text
DEVRELOS_API_TOKEN=<long random secret>
DEVRELOS_REQUIRE_AUTH=true
```

The static operator token remains the bootstrap/break-glass credential. It has broad operator access and should be treated like a root secret.

DevRelOS also supports revocable user API keys with the `drk_` prefix. API key plaintext is returned once at creation, while only a SHA-256 hash is stored in PostgreSQL. Keys can expire, be revoked, and record last-used timestamps.

Liveness and readiness endpoints remain unauthenticated so container and orchestration probes continue to work.

Do not put API credentials in `NEXT_PUBLIC_*`, source control, connector JSON, screenshots, logs, localStorage, or client-side JavaScript state.

## Workspace roles

Workspace membership roles are:

- `owner` — full workspace administration, including owner grants;
- `admin` — membership/security administration and normal writes;
- `editor` — normal DevRel workflow writes;
- `viewer` — read-only access.

User API keys must resolve to an active user and active workspace membership. Read requests require workspace membership; non-read domain operations require at least `editor`.

The static operator token bypasses workspace role checks intentionally as the bootstrap/break-glass path.

## Browser sessions

Set:

```text
DEVRELOS_WEB_SESSIONS=true
DEVRELOS_SESSION_MAX_AGE_SECONDS=28800
```

after bootstrapping the first owner and owner API key.

The `/login` flow validates the supplied `drk_...` key against `/api/v1/identity/me` and stores it in an HttpOnly, SameSite=Strict cookie. The cookie is Secure in production builds. The key is not readable from page JavaScript after sign-in.

The Next.js edge proxy revalidates the credential on protected requests. Revoked, expired, disabled-user, or membership-invalid credentials are cleared and redirected back to `/login`.

The same-origin `/api/devrelos/...` forwarder uses the browser user's session credential whenever session mode is enabled. It does **not** fall back to the operator token. This is important because Go RBAC must remain authoritative for browser mutations.

`/access` is additionally blocked at the web edge for non-admin/non-owner users, while the Go API independently enforces the same authorization boundary.

The current browser session is backed directly by a revocable API key rather than a separate short-lived session-token table. This is acceptable for the current self-hosted beta but should be replaced by dedicated short-lived sessions when OIDC/passwordless login is added.

## Server-rendered reads

Server-rendered data loaders still use the internal server-side API credential. The edge proxy authenticates the browser before those pages render, and sensitive administration routes are role-gated. Browser writes always use the user's own credential. A future workspace-switching/OIDC tranche should propagate the selected workspace/user context into all server-rendered reads as well.

## Legacy Basic operator gate

When browser sessions are disabled, small self-hosted deployments can set:

```text
DEVRELOS_WEB_USERNAME=<operator>
DEVRELOS_WEB_PASSWORD=<strong secret>
```

The Next.js proxy then requires HTTP Basic authentication for the operator UI and same-origin API proxy. This gate should only be used over TLS. Session mode takes precedence when enabled.

## Access & Security workspace

The `/access` workspace lets owners/admins:

- provision workspace users with an initial role;
- change workspace roles;
- create revocable API keys;
- inspect API key metadata and revoke keys;
- inspect recent security audit events.

User provisioning is workspace-scoped. New users receive their initial workspace membership atomically so they cannot be left as invisible orphan records during normal administration.

API key secrets are shown only once when created.

## Audit events

Security-sensitive operations append audit events with workspace, actor, action, resource and metadata. Current identity events include user provisioning, membership changes, API key creation and API key revocation.

The audit stream should be extended to high-risk product actions such as connector credential changes, outreach approvals/sends, publishing, session changes and destructive administration.

## Project and workspace isolation

Write handlers resolve the active project/workspace on the server rather than trusting `projectId` or `workspaceId` supplied in a JSON body. Linked writes such as CFP creation, submission creation/status changes, and connector-run operations validate ownership in SQL.

User API keys are additionally checked against workspace membership before domain requests are allowed. Identity user listings are scoped to the current workspace for admin/owner callers.

## Network exposure

The default Compose configuration binds PostgreSQL and the Go API to `127.0.0.1`. The Next.js web service is the intended network entry point.

Before exposing DevRelOS outside a trusted host/network:

1. terminate HTTPS at a trusted reverse proxy or ingress;
2. enable `DEVRELOS_REQUIRE_AUTH=true` and configure a strong operator token;
3. enable per-user browser sessions after bootstrapping the first owner;
4. store credentials in a deployment secret manager rather than committed environment files;
5. restrict database and media-volume access to the application host/workers;
6. configure backups and restore testing.

## Connector and provider secrets

Provider credentials should currently be supplied at runtime through environment variables or a secret manager. Connector records should contain references/configuration, not copied secret values.

The next security tranche should add encrypted connector-secret storage with per-workspace access controls, rotation metadata and provider-specific secret references.

## Media security

FFmpeg jobs resolve local source/output paths inside configured media roots. The worker rejects paths escaping those roots and the base renderer does not fetch arbitrary remote media URLs.

Treat uploaded/ingested media as untrusted input. A public upload surface should add MIME validation, file-size quotas, storage isolation, malware scanning where appropriate, and resource limits before being exposed to external users.

## Outbound outreach

Outbound outreach remains approval-gated. DevRelOS should not become an autonomous mass-messaging system. Maintain do-not-contact state, source provenance and human review before sending.

## Remaining gaps before broad production

The largest remaining identity/security gaps are OIDC/passwordless login, dedicated short-lived web sessions, invitations, workspace switching, CSRF protections for future cookie-authenticated mutation endpoints, encrypted connector-secret storage, rate limiting, TLS deployment templates, backup/restore, remote object storage policies, stronger audit coverage and horizontal worker coordination.

Security changes should continue to be validated with Go tests, Next.js production builds and authenticated/session-aware Compose smoke tests.
