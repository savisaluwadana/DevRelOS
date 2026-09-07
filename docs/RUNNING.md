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

## Secure operator mode

Authentication is optional for local development. To require authentication on the Go API, set a long random token in `.env` or inject it from a secret manager:

```text
DEVRELOS_API_TOKEN=<secret>
DEVRELOS_REQUIRE_AUTH=true
```

The Go API then requires:

```text
Authorization: Bearer <secret>
```

for `/api/v1/*`. `/healthz` and `/readyz` remain public for container/orchestrator probes.

Browser traffic does not receive the bearer token. Client-side actions call the same-origin path `/api/devrelos/...`; the Next.js route handler forwards the request to the internal Go API and adds `DEVRELOS_API_TOKEN` on the server.

For a small shared operator deployment, the web console can also be protected with Basic authentication:

```text
DEVRELOS_WEB_USERNAME=<operator name>
DEVRELOS_WEB_PASSWORD=<strong password>
```

Both values must be set for this gate to activate. This is an operator-mode protection layer, not the final multi-user identity/RBAC system.

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

Remote source URLs are currently stored as provenance/reference URLs only. The base renderer intentionally refuses to fetch arbitrary remote URLs.

Transcripts can be imported manually through Media Studio. A Whisper-compatible transcription adapter can be added later without changing the media asset/clip/job model.

## GitHub connectors

For authenticated GitHub Issues, Releases or Discussions monitoring, provide the provider credential to the worker environment before starting Compose:

```bash
export GITHUB_TOKEN=...
make up
```

Connector configuration stores the environment-variable reference rather than copying the secret into connector configuration.

## MCP

The MCP server is a stdio process intended for local IDE/agent clients. When API auth is enabled, provide the same API token to the MCP process:

```bash
DEVRELOS_API_URL=http://localhost:8080 \
DEVRELOS_API_TOKEN="$DEVRELOS_API_TOKEN" \
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

Secure operator mode gives DevRelOS a real authenticated service boundary and server-enforced project/workspace scoping, but it is not yet a complete multi-user SaaS security model. It still needs user sessions, workspace membership/RBAC, encrypted connector secret storage, managed TLS/reverse-proxy templates, remote object storage, backup/restore, and horizontal worker coordination before broad internet-facing production use.
