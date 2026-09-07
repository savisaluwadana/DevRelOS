# DevRelOS security model

DevRelOS currently supports a secure single-operator/self-hosted mode. This closes the unauthenticated API boundary without pretending the product already has a complete multi-user authorization system.

## API authentication

Set both:

```text
DEVRELOS_API_TOKEN=<long random secret>
DEVRELOS_REQUIRE_AUTH=true
```

The Go API requires the token as a Bearer credential for `/api/v1/*`. Liveness and readiness endpoints remain unauthenticated so container and orchestration probes continue to work.

Do not put the API token in `NEXT_PUBLIC_*`, source control, connector JSON, screenshots, logs, or client-side storage.

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

Basic authentication is an interim operator-mode control. It does not provide users, sessions, invitations, per-user audit identity, SSO, or RBAC.

## Project and workspace isolation

Write handlers resolve the active project/workspace on the server rather than trusting `projectId` or `workspaceId` supplied in a JSON body. Linked writes such as CFP creation, submission creation/status changes, and connector-run operations validate ownership in SQL.

This reduces accidental or malicious cross-project writes, but the current operator token still has broad operator-level access. Full membership-aware authorization remains future work.

## Network exposure

The default Compose configuration binds PostgreSQL and the Go API to `127.0.0.1`. The Next.js web service is the intended network entry point.

Before exposing DevRelOS outside a trusted host/network:

1. terminate HTTPS at a trusted reverse proxy or ingress;
2. enable `DEVRELOS_REQUIRE_AUTH=true` and configure an API token;
3. protect the operator web surface;
4. store credentials in a deployment secret manager rather than committed environment files;
5. restrict database and media-volume access to the application host/workers;
6. configure backups and restore testing.

## Connector and provider secrets

Provider credentials should be supplied at runtime through environment variables or a secret manager. Connector records should contain references/configuration, not copied secret values.

The next production security tranche should add encrypted connector-secret storage with per-workspace access controls and rotation metadata.

## Media security

FFmpeg jobs resolve local source/output paths inside configured media roots. The worker rejects paths escaping those roots and the base renderer does not fetch arbitrary remote media URLs.

Treat uploaded/ingested media as untrusted input. A public upload surface should add MIME validation, file-size quotas, storage isolation, malware scanning where appropriate, and resource limits before being exposed to external users.

## Outbound outreach

Outbound outreach remains approval-gated. DevRelOS should not become an autonomous mass-messaging system. Maintain do-not-contact state, source provenance and human review before sending.

## Current gaps before multi-user production

The remaining security work includes user/session authentication, workspace membership and RBAC, per-user audit logs, SSO/OIDC where needed, CSRF/session protections for the future cookie-based identity layer, encrypted secret storage, TLS deployment templates, rate limiting, backup/restore, remote object storage policies, and horizontal worker coordination.

Security issues should be fixed in the smallest reproducible scope and validated with Go tests, Next.js production builds and the authenticated Compose smoke test.
