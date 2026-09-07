# DevRelOS

DevRelOS is an open developer-relations operating system for discovering developer pain points, finding speaking and content opportunities, managing community relationships, running DevRel workflows, repurposing media, and measuring outcomes from one place.

The product is designed around three loops:

1. **Discover** — ingest developer signals, topics, pain points, communities, events and CFPs.
2. **Engage** — plan talks, outreach, community activity, content, demos, office hours and campaigns.
3. **Measure** — connect activity to developer engagement, community growth, product feedback and business outcomes.

## Current beta

The repository now contains a runnable self-hosted beta with:

- PostgreSQL workspace/project foundation and tracked migrations
- events, CFPs, talk library and submission workflows
- community opportunities, contacts, relationships, touchpoints and approval-gated outreach
- connector scheduling and run telemetry
- Signal Radar and pain-point clustering
- work-item automation and Content Studio
- developer feedback/product-learning workflow
- Media Studio with transcript import, clip approval and FFmpeg rendering
- authenticated MCP server integration over the domain API
- Next.js operator dashboard
- Go API and background worker
- Docker Compose startup and full-stack CI smoke tests

## Product surfaces

- **Command Center** — priorities, deadlines, activity and DevRel scorecard.
- **Signal Radar** — developer signals from pluggable providers such as GitHub, Bluesky, Hacker News and RSS.
- **Topic & Pain-Point Intelligence** — evidence-backed recurring problems and trends.
- **Events & CFPs** — discovery, deadlines, talk library and submission pipeline.
- **Community Graph & Outreach** — communities, organizers, relationships, touchpoints and human-approved outreach.
- **Content Studio** — briefs, drafts, publishing workflow and repurposing.
- **Media Studio** — recording metadata, transcripts, clip candidates and FFmpeg rendering.
- **Developer Feedback** — convert community evidence into product feedback and follow-up.
- **Automation & MCP** — scheduled connectors plus authenticated agent/IDE domain tools.

## Repository layout

```text
apps/
  web/                 Next.js operator dashboard and same-origin API proxy
services/
  api/                 Go domain REST API
  worker/              connector/background/media worker
  mcp/                 MCP server
internal/
  connectors/          provider contracts and registry
  domain/              DevRel domain models
  intelligence/        scoring, clustering and workflow logic
  providers/           external source adapters
  storage/             PostgreSQL repositories
migrations/            tracked PostgreSQL schema
```

PostgreSQL is the system of record. Optional cache/queue, object storage and dedicated search infrastructure should be introduced only when scale requires them rather than becoming parallel sources of truth.

## Run locally

Requirements:

- Docker with Compose v2
- Go 1.25+ for native Go development
- Node.js 22+ for native web development

Start the full stack:

```bash
cp .env.example .env
make up
```

Open `http://localhost:3000`.

Health/readiness endpoints:

```text
http://localhost:8080/healthz
http://localhost:8080/readyz
```

Useful commands:

```bash
make ps
make logs
make down
```

See [Running DevRelOS](docs/RUNNING.md) for native development, Media Studio and MCP instructions.

## Secure operator mode

Local development can run without authentication. For a shared/self-hosted operator deployment, configure a long random API token and require it:

```text
DEVRELOS_API_TOKEN=<secret managed outside git>
DEVRELOS_REQUIRE_AUTH=true
```

The Next.js server forwards that bearer token to the Go API; the token is never exposed through a `NEXT_PUBLIC_*` environment variable. Optional `DEVRELOS_WEB_USERNAME` and `DEVRELOS_WEB_PASSWORD` protect the operator console with HTTP Basic authentication until full multi-user identity/RBAC is implemented.

PostgreSQL and the Go API bind to localhost in the default Compose deployment. Put TLS and a trusted reverse proxy in front of the web service before exposing DevRelOS across a network.

See [Security](docs/SECURITY.md) for the current trust model and production boundary.

## Design principles

- Provider interfaces instead of hard-coded scrapers.
- Store source provenance and fetched-at timestamps for external evidence.
- Respect API terms, robots directives, rate limits and content licensing.
- Human approval for outbound outreach.
- Make automation observable: runs have inputs, outputs, cost and status.
- MCP exposes domain actions rather than direct database access.
- Keep the core useful without an LLM; AI should enhance prioritization, clustering and drafting.
- Scope writes to the active project/workspace on the server rather than trusting IDs supplied by a client.

## Key docs

- [Product scope](docs/PRODUCT.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Integrations and data sources](docs/INTEGRATIONS.md)
- [Running DevRelOS](docs/RUNNING.md)
- [Security](docs/SECURITY.md)
- [MCP](docs/MCP.md)
- [Delivery roadmap](docs/ROADMAP.md)

## Next production tranche

The current secure operator mode is intentionally not a substitute for a multi-user SaaS identity layer. The next hardening work should add user/session authentication, workspace membership and RBAC, encrypted connector secret storage, remote object storage and backup/restore, TLS/reverse-proxy deployment templates, structured observability, and horizontal worker coordination.
