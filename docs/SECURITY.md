# DevRelOS security model

DevRelOS now supports authenticated self-hosted operator mode plus a first-class user/workspace authorization foundation.

## API authentication

Set both:

```text
DEVRELOS_API_TOKEN=<long random secret>
DEVRELOS_REQUIRE_AUTH=true
```

The static operator token remains the bootstrap/break-glass credential. It has broad operator access and should be treated like a root secret.

DevRelOS also supports revocable user API keys with the `drk_` prefix. API key plaintext is returned once at creation, while only a SHA-256 hash is stored in PostgreSQL. Keys can expire, be revoked, and record last-used timestamps.

Liveness and readiness endpoints remain unauthenticated so container and orchestration probes continue to work.

Do not put API credentials in `NEXT_PUBLIC_*`, source control, connector JSON, screenshots, logs, or client-side storage.

## Workspace roles

Workspace membership roles are:

- `owner` — full workspace administration, including owner grants
- `admin` — membership/security administration and normal writes
- `editor` — normal DevRel workflow writes
- `viewer` — read-only access

User API keys must resolve to an active user and active workspace membership. Read requests require workspace membership; non-read domain operations require at least `editor`.

The static operator token bypasses workspace role checks intentionally as the bootstrap/break-glass path.

## Web-to-API boundary

Browser code calls `/api/devrelos/...` on the Next.js application. The Next.js server forwards those requests to `DEVRELOS_API_URL` and injects `DEVRELOS_API_TOKEN` server-side. Server-rendered pages use the same private backend configuration.

This design prevents the Go API credential from being bundled into browser JavaScript.

## Operator console authentication

For small shared/self-hosted deployments, set both:

```text
DEVRELOS_WEB_USERNAME=<operator>
DEVRELOS_WEB_PASSWORD=<strong secret>
```

The Next.js proxy then requires HTTP Basic authentication for the operator UI and same-origin API proxy. This gate should only be used over TLS.

Basic authentication remains an interim browser gate. It is separate from the API user/RBAC model. The next identity tranche should replace this with browser sessions backed by OIDC/passwordless identity and workspace membership.

## Access & Security workspace

The `/access` workspace lets the bootstrap operator:

- create users;
- assign workspace roles;
- create revocable API keys;
- inspect API key metadata and revoke keys;
- inspect recent security audit events.

API key secrets are shown only once when created.

## Audit events

Security-sensitive operations append audit events with workspace, actor, action, resource and metadata. Current identity events include user creation, membership changes, API key creation and API key revocation.

The audit stream should be extended to high-risk product actions such as connector credential changes, outreach approvals/sends, publishing, user/session changes and destructive administration.

## Project and workspace isolation

Write handlers resolve the active project/workspace on the server rather than trusting `projectId` or `workspaceId` supplied in a JSON body. Linked writes such as CFP creation, submission creation/status changes, and connector-run operations validate ownership in SQL.

User API keys are additionally checked against workspace membership before domain requests are allowed.

## Network exposure

The default Compose configuration binds PostgreSQL and the Go API to `127.0.0.1`. The Next.js web service is the intended network entry point.

Before exposing DevRelOS outside a trusted host/network:

1. terminate HTTPS at a trusted reverse proxy or ingress;
2. enable `DEVRELOS_REQUIRE_AUTH=true` and configure an operator token;
3. protect the web surface;
4. store credentials in a deployment secret manager rather than committed environment files;
5. restrict database and media-volume access to the application host/workers;
6. configure backups and restore testing.

## Connector and provider secrets

Provider credentials should be supplied at runtime through environment variables or a secret manager. Connector records should contain references/configuration, not copied secret values.

A later production tranche should add encrypted connector-secret storage with per-workspace access controls, rotation metadata and provider-specific secret references.

## Media security

FFmpeg jobs resolve local source/output paths inside configured media roots. The worker rejects paths escaping those roots and the base renderer does not fetch arbitrary remote media URLs.

Treat uploaded/ingested media as untrusted input. A public upload surface should add MIME validation, file-size quotas, storage isolation, malware scanning where appropriate, and resource limits before being exposed to external users.

## Outbound outreach

Outbound outreach remains approval-gated. DevRelOS should not become an autonomous mass-messaging system. Maintain do-not-contact state, source provenance and human review before sending.

## Remaining gaps before multi-user production

The largest remaining identity/security gaps are browser user sessions, OIDC/passwordless login, invitations, workspace switching, CSRF/session protections, encrypted connector-secret storage, rate limiting, TLS deployment templates, backup/restore, remote object storage policies and horizontal worker coordination.

Security changes should continue to be validated with Go tests, Next.js production builds and the authenticated Compose smoke test.
