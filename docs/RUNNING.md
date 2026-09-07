# Running DevRelOS

## Fastest path: Docker Compose

Prerequisites:

- Docker with Compose v2
- port 3000 available
- ports 5432 and 8080 available on localhost if you want direct database/API access

Start the complete stack:

```bash
cp .env.example .env
make up
```

This starts PostgreSQL, applies every unapplied SQL migration, starts the Go API, starts the Go worker with FFmpeg installed, and starts the Next.js operator console.

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

PostgreSQL and the Go API are bound to `127.0.0.1` by the default Compose file. The web service is the intended operator entry point.

## API authentication

Authentication is optional for local development. To require authentication on the Go API, set a long random bootstrap/operator token in `.env` or inject it from a secret manager:

```text
DEVRELOS_API_TOKEN=<secret>
DEVRELOS_REQUIRE_AUTH=true
```

The Go API then requires a valid bearer credential for `/api/v1/*`. `/healthz` and `/readyz` remain public for container/orchestrator probes.

The static operator token is now intended as a bootstrap/break-glass credential. Normal users should use revocable `drk_...` API keys attached to workspace memberships.

## Bootstrap the first user

Before enabling browser sessions, start DevRelOS in operator mode and open **Access & Security**. Create the first user with role `owner`, then mint an API key for that user and copy it immediately. DevRelOS stores only the key hash and does not show the plaintext key again.

You can also bootstrap via the API using the operator bearer credential.

Once an owner key exists, set:

```text
DEVRELOS_WEB_SESSIONS=true
DEVRELOS_SESSION_MAX_AGE_SECONDS=28800
```

and restart the web service or stack:

```bash
make down
make up
```

Opening `http://localhost:3000` now redirects to `/login`. Sign in with the owner's `drk_...` key.

In session mode:

- the key is kept in an HttpOnly, SameSite=Strict cookie;
- the edge proxy revalidates it against `/api/v1/identity/me`;
- revoked, expired or disabled-user keys are rejected on the next protected request;
- browser API calls are forwarded with the signed-in user's key, never the operator token;
- Go RBAC therefore applies to all browser mutations;
- `/access` is available only to `owner` and `admin` users;
- `viewer` is read-only and `editor` can perform normal domain writes.

Use the **Sign out** control to clear the browser session cookie.

## Legacy Basic operator gate

For a small deployment that has not enabled user sessions, the web console can still be protected with Basic authentication:

```text
DEVRELOS_WEB_USERNAME=<operator name>
DEVRELOS_WEB_PASSWORD=<strong password>
```

Both values must be set for this gate to activate. `DEVRELOS_WEB_SESSIONS=true` takes precedence over this Basic gate.

Use TLS and a trusted reverse proxy before exposing the web service outside a trusted host/network. See [SECURITY.md](SECURITY.md).

## Media Studio

The worker mounts `./data/media` as `/data/media`.

Put local recordings under:

```text
data/media/recordings/
```

When creating a Media Studio asset, use a relative source path such as:

```text
recordings/community-call.mp4
```

DevRelOS only allows the worker to resolve paths inside `DEVRELOS_MEDIA_ROOT`. Approved clips are rendered by FFmpeg into:

```text
data/media/outputs/
```

Remote source URLs are currently stored as provenance/reference URLs only. The base renderer intentionally refuses to fetch arbitrary remote media URLs.

Transcripts can be imported manually through Media Studio. A Whisper-compatible transcription adapter can be added later without changing the media asset/clip/job model.

## GitHub connectors

For authenticated GitHub Issues, Releases or Discussions monitoring, provide the provider credential to the worker environment before starting Compose:

```bash
export GITHUB_TOKEN=...
make up
```

Connector configuration stores the environment-variable reference rather than copying the secret into connector configuration.

## MCP

The MCP server is a stdio process intended for local IDE/agent clients. Prefer a dedicated DevRelOS user/API key with only the workspace role the agent requires:

```bash
DEVRELOS_API_URL=http://localhost:8080 \
DEVRELOS_API_TOKEN="drk_..." \
go run ./services/mcp
```

The MCP process forwards the bearer token and does not persist it. See [MCP.md](MCP.md) for the tool catalog and client configuration guidance.

## Local development without full Compose

Start PostgreSQL:

```bash
make db-up
```

Set the local database URL and apply tracked migrations:

```bash
export DATABASE_URL='postgres://devrelos:devrelos@localhost:5432/devrelos?sslmode=disable'
make migrate
```

Then run the services in separate terminals:

```bash
make api
make worker
make web
```

For native Next.js development, `DEVRELOS_API_URL` should point at the native Go API and `NEXT_PUBLIC_API_URL` should remain `/api/devrelos` so browser mutations continue through the Next.js proxy.

## Current production boundary

DevRelOS now has user identities, workspace memberships, RBAC, revocable API keys, audit events, and per-user browser sessions. Remaining production work includes OIDC/SSO and invitation flows, workspace switching, encrypted connector secret storage, rate limiting, managed TLS/reverse-proxy templates, remote object storage, backup/restore, observability, and horizontal worker coordination before broad internet-facing production use.
