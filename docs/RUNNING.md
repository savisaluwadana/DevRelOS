# Running DevRelOS

## Fastest path: Docker Compose

Prerequisites:

- Docker with Compose v2
- ports 3000, 5432 and 8080 available

Start the complete beta stack:

```bash
make up
```

This starts PostgreSQL, applies every unapplied SQL migration, starts the Go API, starts the Go worker with FFmpeg installed, and starts the Next.js operator console.

Open:

- Web: http://localhost:3000
- API health: http://localhost:8080/healthz

Useful commands:

```bash
make ps
make logs
make down
```

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

DevRelOS will only allow the worker to resolve paths inside `DEVRELOS_MEDIA_ROOT`. Approved clips are rendered by FFmpeg into:

```text
data/media/outputs/
```

Remote source URLs are currently stored as provenance/reference URLs only. The base renderer intentionally refuses to fetch arbitrary remote URLs.

Transcripts can be imported manually through Media Studio. A Whisper-compatible transcription adapter can be added later without changing the media asset/clip/job model.

## GitHub connectors

For authenticated GitHub Issues, Releases or Discussions monitoring, export a token before starting Compose:

```bash
export GITHUB_TOKEN=...
make up
```

Connector configuration stores only the environment-variable name, not the secret value.

## MCP

The MCP server is a stdio process intended for local IDE/agent clients:

```bash
DEVRELOS_API_URL=http://localhost:8080 go run ./services/mcp
```

See [MCP.md](MCP.md) for the tool catalog and client configuration guidance.

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

## Current beta boundary

The Docker Compose path is intended for local/self-hosted beta operation. It does not yet include multi-user login/session management, production TLS termination, managed secrets, remote object storage, or horizontal worker orchestration. Do not expose the current beta API directly to the public internet without adding an authenticated reverse proxy or the planned application auth layer.
