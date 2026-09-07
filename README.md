# DevRelOS

DevRelOS is an open developer-relations operating system for discovering developer pain points, finding speaking and content opportunities, managing community relationships, running DevRel workflows, repurposing media, and measuring outcomes from one place.

The product is designed around three loops:

1. **Discover** — ingest developer signals, topics, pain points, communities, events and CFPs.
2. **Engage** — plan talks, outreach, community activity, content, product feedback and campaigns.
3. **Measure** — connect real DevRel work to replies, accepted talks, shipped feedback, campaign outcomes and follow-up.

> New to DevRelOS? Start with the **[complete Operator Guide](docs/OPERATOR_GUIDE.md)**. It covers setup, every workspace, recommended workflows, connectors, MCP, security, backups and troubleshooting.

## Current beta

The repository contains a runnable self-hosted beta with:

- PostgreSQL workspace/project foundation and tracked migrations
- user identity, workspace membership/RBAC, revocable API keys, short-lived browser sessions and invitations
- encrypted connector secrets and workspace-scoped audit events
- events, CFPs, reusable talks and submission workflows
- opportunity intelligence for CFP/talk and community/talk matching
- communities, contacts, relationships, touchpoints and Relationship Radar
- approval-gated outreach with optional durable SMTP delivery
- connector scheduling, run telemetry and provider policy controls
- Signal Radar and recurring pain-point clustering
- Action Queue and evidence-backed Content Studio
- developer feedback/product-learning workflow with GitHub issue prefill and linked public-issue sync
- Campaigns & Attribution with derived outcome reporting
- Unified DevRel Calendar for deadlines, scheduled work and follow-ups
- Media Studio with transcript import, clip approval and FFmpeg rendering
- authenticated MCP server over the domain API
- Next.js operator dashboard and same-origin browser API proxy
- Go API and background worker
- Caddy-based HTTPS production profile, metrics, request IDs, backup/restore and full-stack CI smoke tests

## Product surfaces

- **Command Center** — priorities, deadline risk, relationship risk, campaign impact and DevRel scorecard.
- **Calendar** — CFP deadlines, events, campaign milestones, scheduled content, work due dates and relationship follow-ups.
- **Signal Radar** — developer signals from pluggable providers such as GitHub, Bluesky, Hacker News and RSS.
- **Pain-Point Intelligence** — evidence-backed recurring problems and trends.
- **Opportunities** — inspectable speaking and CFP/talk matching.
- **Campaigns & Attribution** — connect content, work, outreach, feedback, talks/events and other activity to measurable outcomes.
- **Relationship Radar** — relationship health, risk and recommended follow-up actions.
- **Events & CFPs** — discovery, deadlines, talk library and submission pipeline.
- **Community Graph & Outreach** — communities, organizers, relationships, touchpoints and human-approved outreach.
- **Action Queue** — prioritized source-backed DevRel execution.
- **Content Studio** — briefs, drafts, review, scheduling and publication tracking.
- **Media Studio** — recording metadata, transcripts, clip candidates and FFmpeg rendering.
- **Developer Feedback** — pain-point-to-product workflow, GitHub issue handoff/sync and developer follow-up.
- **Integrations** — scheduled provider connectors, credentials and run telemetry.
- **Access & Security** — users, roles, invitations, API keys, sessions and encrypted connector secrets.
- **Automation & MCP** — authenticated agent/IDE domain tools without direct database access.

## Repository layout

```text
apps/
  web/                 Next.js operator dashboard and same-origin API proxy
services/
  api/                 Go domain REST API
  worker/              connector/background/media/outreach worker
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

See [Running DevRelOS](docs/RUNNING.md) for native development, production deployment, Media Studio and MCP instructions.

## Secure operator mode

Local development can run without authentication. The supported production profile enables required API authentication, browser sessions, workspace RBAC, encrypted connector credentials, rate limiting and Caddy-managed HTTPS.

The root bootstrap credential is:

```text
DEVRELOS_API_TOKEN=<secret managed outside git>
```

Use the root token only to bootstrap the first owner or for break-glass access. Normal users should use revocable `drk_...` API keys and dedicated short-lived `ds_...` browser sessions.

Never expose the root token through a `NEXT_PUBLIC_*` variable or browser storage.

See [Security](docs/SECURITY.md) for the trust model and supported production boundary.

## Design principles

- Provider interfaces instead of hard-coded scrapers.
- Store source provenance and fetched-at timestamps for external evidence.
- Respect API terms, robots directives, rate limits and content licensing.
- Human approval for outbound outreach and other external side effects.
- Make automation observable: runs have inputs, outputs, cost and status.
- MCP exposes domain actions rather than direct database access.
- Keep the core useful without an LLM; AI should enhance prioritization, clustering and drafting.
- Scope writes to the active project/workspace on the server rather than trusting IDs supplied by a client.
- Do not infer product delivery from external state alone; important lifecycle transitions remain explicit operator actions.

## Key docs

- **[Complete Operator Guide](docs/OPERATOR_GUIDE.md)**
- [Running DevRelOS](docs/RUNNING.md)
- [Security](docs/SECURITY.md)
- [Product scope](docs/PRODUCT.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Integrations and data sources](docs/INTEGRATIONS.md)
- [MCP](docs/MCP.md)
- [Delivery roadmap](docs/ROADMAP.md)

## Current production boundary

The supported v1 deployment is a **single-node, HTTPS-terminated, authenticated self-hosted installation** with durable PostgreSQL/media storage, workspace RBAC, short-lived browser sessions, encrypted connector secrets, audit events, rate limiting, health/readiness probes, metrics, backup/restore, scheduled workers, approval-gated SMTP delivery and Caddy TLS.

OIDC/SSO, multi-region/high-availability media/object storage and additional official provider write APIs can be added as deployment/product extensions without weakening the current trust model.
