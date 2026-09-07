# DevRelOS

DevRelOS is an open developer-relations operating system for discovering developer pain points, finding speaking and content opportunities, managing community relationships, running DevRel workflows, and measuring outcomes from one place.

The product is designed around three loops:

1. **Discover** — ingest developer signals, topics, pain points, communities, events and CFPs.
2. **Engage** — plan talks, outreach, community activity, content, demos, office hours and campaigns.
3. **Measure** — connect activity to developer engagement, community growth, product feedback and business outcomes.

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

## Proposed architecture

```text
apps/
  web/                 Next.js operator dashboard
services/
  api/                 Go API and domain services
  worker/              Go ingestion/scoring/background jobs
  mcp/                 MCP server exposing DevRelOS capabilities
packages/
  contracts/           API/event schemas
  ui/                  shared UI primitives
  connectors/          connector metadata and shared fixtures
internal/
  domain/              core Go domain packages
  providers/           Reddit/X/Bluesky/GitHub/events/community adapters
  workflows/           automation workflows
  intelligence/        clustering, ranking, scoring and summarization
  storage/             PostgreSQL/search/object-store adapters
```

PostgreSQL is the system of record. Redis can be added for short-lived queues/cache. Object storage holds media and exports. Search/vector capabilities should be introduced behind interfaces rather than becoming the source of truth.

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

## Initial build order

1. Foundation: auth/workspaces, projects, PostgreSQL, connector framework and job model.
2. Event + CFP ingestion and submission pipeline.
3. Signal ingestion and topic/pain-point intelligence.
4. CNCF/community graph and speaking-outreach pipeline.
5. Content studio and repurposing jobs.
6. MCP server and workflow automation.
7. Analytics, attribution and team operations.

## Status

DevRelOS is at foundation stage. The repository is being structured around a modular monolith first, with clear service boundaries so ingestion workers and the MCP server can be split out when load or security boundaries justify it.
