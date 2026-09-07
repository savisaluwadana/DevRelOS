# DevRelOS Architecture

## Architectural approach

Start as a **modular monolith with independent workers and an MCP edge**. DevRelOS has many domains, but splitting them into microservices immediately would create operational cost without product value. Domain boundaries should still be explicit so high-load ingestion, media processing or MCP workloads can be separated later.

## Runtime components

```text
Browser
  |
  v
Next.js Web App
  |
  v
Go API -----------------------------------------------------+
  |                                                         |
  +--> PostgreSQL                                           |
  +--> object storage                                       |
  +--> Redis (optional cache/queue)                         |
  +--> search/vector adapter (optional)                     |
                                                            |
Go Worker <--- scheduler/workflow engine <--- connector jobs+
  |
  +--> social/community/event/content providers
  +--> clustering/scoring/intelligence
  +--> media pipeline

MCP Server
  |
  +--> authenticated DevRelOS domain API
  +--> tools/resources/prompts for agents and IDEs
```

## Repository layout

```text
apps/
  web/
services/
  api/
  worker/
  mcp/
packages/
  contracts/
  ui/
  connectors/
internal/
  auth/
  domain/
    signals/
    intelligence/
    events/
    communities/
    outreach/
    content/
    feedback/
    campaigns/
    analytics/
    automations/
  providers/
  workflows/
  storage/
  observability/
docs/
```

## Backend

Use Go for the core API, connector orchestration and worker processes. Reasons:
- good fit for concurrent ingestion and polling
- predictable memory/runtime footprint
- strong HTTP/networking ecosystem
- simple static binaries for self-hosting
- easy separation of workers later

The first API can be REST/JSON. Internal boundaries should use typed domain services rather than routing business logic through HTTP inside the same deployment.

## Frontend

Use Next.js with a component system suitable for dense operator dashboards. The UI should prioritize:
- command palette
- global search
- saved views
- table/kanban/calendar layouts
- side-panel detail views
- bulk tagging without bulk outbound sending
- activity timelines
- filters that can be saved as monitors

## Persistence

### PostgreSQL

System of record for all durable domain data.

Core table groups:
- identity/workspaces/projects
- sources/connectors/credentials metadata
- signals/topics/pain points
- communities/contacts/relationships/touchpoints
- events/cfps/submissions/talks
- content/assets/campaigns
- feedback/tasks
- automation definitions/runs
- metrics/attribution

Use JSONB only for provider-specific payloads and snapshots. Important fields needed for filtering, joins or policy must be normalized.

### Raw provider snapshots

For external data, retain:
- provider
- external ID
- canonical URL
- fetched at
- source timestamp
- content hash
- normalized fields
- minimal raw payload where licensing/terms permit storage

This makes connector behavior auditable and enables reprocessing without pretending scraped/third-party content is owned by DevRelOS.

### Search

Start with PostgreSQL full-text/trigram search. Introduce a dedicated search engine only when query volume or faceting justifies it.

### Vector/semantic layer

Embeddings can improve clustering and retrieval but must not become the canonical store. Store embedding model/version and regenerate safely.

## Connector framework

Every source implements a common contract conceptually similar to:

```text
Provider
  ID()
  Capabilities()
  ValidateConfig()
  Fetch(cursor, filters)
  Normalize(raw)
  RateLimitPolicy()
  RetentionPolicy()
```

Capabilities can include:
- search
- stream
- timeline
- event feed
- CFP feed
- community directory
- publishing
- analytics
- webhooks

Each connector run records:
- input configuration
- cursor/window
- requests made
- items fetched
- items created/updated/skipped
- provider cost when known
- rate-limit state
- warnings/errors
- duration

## Ingestion pipeline

```text
Provider fetch
  -> validate
  -> normalize
  -> canonicalize URLs/entities
  -> deduplicate
  -> policy/licensing gate
  -> persist signal/event/community
  -> enrich
  -> cluster/score
  -> emit domain event
  -> update monitors/alerts
```

Enrichment jobs should be idempotent and retryable.

## Intelligence layer

AI should be used for bounded operations with evidence links:
- classify topic/persona
- extract pain-point statements
- cluster semantically similar signals
- summarize a cluster
- score relevance against project goals
- suggest content angles
- suggest talk/CFP fit
- draft personalized outreach
- extract questions/actions from transcripts

Never persist only an AI summary when source evidence is available. The UI should allow users to inspect the supporting signals.

## Event and CFP model

Separate Event from CFP because:
- an event can have zero, one or multiple submission windows
- CFP dates can change independently
- one event can contain multiple tracks/programs

Suggested structure:

```text
Event
  -> CFP(s)
  -> Submission(s)
  -> Talk(s)
  -> EventParticipation
```

EventParticipation tracks travel/logistics, booth/sponsor status, attendance and post-event actions separately from speaking submissions.

## Community graph

Use graph-like relationships in relational tables first:

```text
Community <- CommunityContact -> Contact
Project <- Relationship -> Community/Contact
Relationship -> Touchpoint[]
Community -> Event[]
Talk -> Delivery[] -> Event
```

This supports questions such as:
- Which CNCF groups have not been contacted in 90 days?
- Which organizers have previously accepted our speakers?
- Which communities discuss platform engineering frequently?
- Which relationships are warm enough for a co-hosted session?

A graph database is unnecessary for MVP.

## Outreach safety model

Outbound actions have states:

`draft -> needs_approval -> approved -> queued -> sent -> replied/failed/cancelled`

Rules:
- no autonomous bulk outreach in early releases
- deduplicate recipients and active threads
- enforce per-workspace and per-channel limits
- support do-not-contact state
- retain why a contact was selected
- require a legitimate/public/user-provided contact source
- log generated versus user-edited copy

## Automation engine

Keep automation definitions in DevRelOS while allowing execution adapters.

Core trigger types:
- schedule
- webhook
- domain event
- monitor condition
- manual

Core actions:
- run connector
- create/update entity
- call domain action
- notify
- request approval
- enqueue media job
- call webhook
- invoke MCP-capable agent

A workflow engine such as Temporal can be introduced for durable long-running execution. Node-RED-style webhook integrations can be supported without making an external automation product the DevRelOS database of record.

## MCP design

The MCP server is an authenticated control plane over DevRelOS domain actions.

Example tools:
- `signals.search`
- `pain_points.list`
- `pain_points.get_evidence`
- `events.search`
- `cfps.list_open`
- `cfps.score_fit`
- `submissions.create_draft`
- `communities.search`
- `communities.score_speaking_fit`
- `relationships.log_touchpoint`
- `outreach.create_draft`
- `content.create_brief`
- `content.repurpose`
- `feedback.create`
- `campaigns.create`
- `analytics.get_scorecard`

Example resources:
- `devrel://projects/{id}`
- `devrel://pain-points/{id}`
- `devrel://events/{id}`
- `devrel://communities/{id}`
- `devrel://talks/{id}`
- `devrel://campaigns/{id}`

The MCP server should enforce the same authorization and approval policies as the UI/API.

## Media architecture

DevRelOS should own orchestration and metadata, not reinvent editing UX.

```text
Recording
 -> object storage
 -> transcription
 -> transcript segmentation
 -> candidate clip scoring
 -> review
 -> FFmpeg render job
 -> captions/thumbnails/metadata
 -> exported asset variants
```

Adapters can later hand assets to Kdenlive/Shotcut-compatible workflows or commercial editors without coupling the core model to any one editor.

## Observability

Every connector, automation and media job needs:
- status
- structured logs
- attempt count
- duration
- provider cost if available
- items processed
- error category
- trace/correlation ID

Expose an operations view in the dashboard from the beginning.

## Security

- OAuth/OIDC for users
- workspace/project RBAC
- connector credentials encrypted at rest
- never expose provider secrets through MCP resources
- scoped connector permissions
- outbound approval policy
- audit log for writes and sends
- SSRF-safe URL fetching
- allow/deny lists for web ingestion
- HTML sanitization
- file/media validation

## Deployment

Initial self-hosted shape:

```text
web
api
worker
mcp
postgres
object-store
redis (optional)
```

Containerize each runtime. Kubernetes support can come after a Compose/local path works cleanly.

## Scaling boundaries

Split components only when justified:
- ingestion workers by provider when quotas/load differ
- media workers when CPU/GPU jobs dominate
- MCP service for isolated credentials/rate limiting
- analytics pipeline when event volume grows

Until then, optimize for a solo team being able to understand and operate the whole system.
