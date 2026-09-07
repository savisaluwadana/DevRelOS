# DevRelOS

DevRelOS is an open developer-relations operating system for discovering developer pain points, finding speaking and content opportunities, managing community relationships, running DevRel workflows, and measuring outcomes from one place.

The product is designed around three loops:

1. **Discover** — ingest developer signals, topics, pain points, communities, events and CFPs.
2. **Engage** — plan talks, outreach, community activity, content, demos, office hours and campaigns.
3. **Measure** — connect activity to developer engagement, community growth, product feedback and business outcomes.

## Current MVP

The first working vertical slice is now being implemented around the event and speaking workflow:

- PostgreSQL workspace/project foundation
- events and CFPs
- talk library
- submission pipeline and status transitions
- community opportunity records and speaking-fit scores
- dashboard scorecard
- connector/provider contract
- developers.events provider with explicit licensing policy
- Go API and worker runtimes
- Next.js operator dashboard

## Product surfaces

- **Command Center** — priorities, deadlines, tasks, activity feed and DevRel scorecard.
- **Signal Radar** — Reddit, X, Bluesky, Hacker News, GitHub, RSS/web and other source adapters.
- **Topic & Pain-Point Intelligence** — cluster signals, track trends, evidence and developer questions.
- **Events & CFPs** — event discovery, CFP deadlines, submission pipeline, talk library and calendar.
- **Community Graph** — CNCF/Open Community Groups, local meetups, organizers, relationships and outreach pipeline.
- **Content Studio** — briefs, articles, social posts, demos, newsletters, clips and repurposing workflows.
- **Developer Feedback** — support themes, docs friction, feature requests and product feedback routing.
- **Campaigns** — launches, webinars, workshops, hackathons, community calls and partner activities.
- **Analytics** — activity, reach, engagement, developer activation, acceptance rates and attributable outcomes.
- **Automation & MCP** — MCP tools/resources, webhooks, scheduled jobs and pluggable workflow runners.

## Repository layout

```text
apps/
  web/                 Next.js operator dashboard
services/
  api/                 Go REST API
  worker/              connector/background worker
  mcp/                 MCP server (next implementation tranche)
internal/
  connectors/          provider contracts and registry
  domain/events/       event/CFP/talk/community models
  providers/           external source adapters
  storage/             PostgreSQL repository
migrations/            PostgreSQL schema
```

PostgreSQL is the system of record. Redis can be added later for short-lived queues/cache. Object storage will hold media and exports. Search/vector capabilities should be introduced behind interfaces rather than becoming the source of truth.

## Run locally

Requirements:

- Go 1.24+
- Node.js compatible with Next.js 16
- Docker
- PostgreSQL client (`psql`) for the current migration command

```bash
cp .env.example .env
set -a && source .env && set +a

make db-up
make migrate
```

Start the API:

```bash
make api
```

Start the web dashboard in another terminal:

```bash
make web-install
make web
```

Then open `http://localhost:3000`.

The API listens on `http://localhost:8080` by default. Useful endpoints include:

```text
GET    /healthz
GET    /api/v1/dashboard
GET    /api/v1/events
POST   /api/v1/events
GET    /api/v1/cfps
POST   /api/v1/cfps
GET    /api/v1/talks
POST   /api/v1/talks
GET    /api/v1/submissions
POST   /api/v1/submissions
PATCH  /api/v1/submissions/{id}/status
GET    /api/v1/communities
POST   /api/v1/communities
```

Start the worker separately:

```bash
make worker
```

For a bounded connectivity check against the developers.events provider:

```bash
DEVRELOS_DEMO_FETCH_DEVELOPERS_EVENTS=1 make worker
```

This fetch mode does not automatically redistribute or persist the remote dataset. The provider policy marks the source as not approved for commercial redistribution by default because developers.events content/data is licensed separately from its code.

## Design principles

- Provider interfaces instead of hard-coded scrapers.
- Store source provenance and fetched-at timestamps for every external signal.
- Respect API terms, robots directives, rate limits and content licensing.
- Human approval for outbound outreach in the first releases.
- Make automation observable: every run has inputs, outputs, logs, cost and status.
- MCP exposes domain actions, not direct database access.
- Keep the core useful without an LLM; AI enhances prioritization, clustering and drafting.

## Key docs

- [Product scope](docs/PRODUCT.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Integrations and data sources](docs/INTEGRATIONS.md)
- [Delivery roadmap](docs/ROADMAP.md)

## Next implementation tranche

1. Apply migrations automatically or via a dedicated migration binary.
2. Add CRUD/detail views for events, CFPs, talks and communities in the web app.
3. Add connector persistence, scheduled runs, source records and run telemetry.
4. Normalize developers.events imports behind the policy gate.
5. Add OCG/CNCF community ingestion and relationship/touchpoint workflows.
6. Add approval-gated outreach drafts.
7. Build Signal Radar providers and pain-point clustering.
8. Add the MCP server over domain APIs.
