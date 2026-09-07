# DevRelOS Delivery Roadmap

The roadmap prioritizes an end-to-end DevRel loop before breadth. A narrow workflow that moves from discovery to measurable outcome is more valuable than dozens of disconnected integrations.

## Phase 0 — Foundation

Goal: establish the product skeleton and policies.

Deliverables:
- monorepo structure
- Next.js web shell
- Go API
- Go worker
- PostgreSQL
- migrations
- workspace/project model
- auth/RBAC skeleton
- connector registry
- background job/run model
- audit log
- source provenance model
- operations page for job status
- Docker Compose development environment

Exit criteria:
- a user can create a workspace/project
- a connector can be registered and run
- normalized records can be persisted with provenance
- failures/retries are observable

## Phase 1 — Events, CFPs and talk pipeline

Goal: replace the CFP spreadsheet first.

Deliverables:
- event entity
- CFP entity
- developers.events connector behind licensing/config policy
- manual event/CFP entry
- CSV/ICS import
- CFP filters/search
- deadline views
- calendar view
- talk library
- submission pipeline
- fit scoring rules
- reminders
- acceptance/rejection tracking

Dashboard widgets:
- closing in 7 days
- high-fit open CFPs
- submissions awaiting work
- accepted talks
- upcoming deliveries

Exit criteria:
- a user can discover/import an event, evaluate it, attach a talk, submit, and track the outcome.

## Phase 2 — Signal Radar and pain-point intelligence

Goal: turn public developer conversation into an evidence-backed research queue.

Initial sources:
- GitHub
- RSS/Atom
- Hacker News
- Bluesky
- Reddit when approved/configured
- X when configured with budget controls

Deliverables:
- saved monitors
- scheduled ingestion
- deduplication
- canonical URLs
- topic tags
- pain-point extraction
- semantic clustering
- evidence panel
- trend velocity
- watchlists
- alert rules
- convert cluster to content idea or developer feedback

Exit criteria:
- a user can define a monitor and receive a ranked, inspectable set of developer pain points with source evidence.

## Phase 3 — CNCF/community graph and speaking outreach

Goal: operationalize community relationship building.

Deliverables:
- community model
- contact model
- relationship/touchpoint model
- OCG/CNCF community ingestion
- geography/timezone filters
- event cadence/activity scoring
- topic-fit scoring
- organizer/public contact provenance
- talk-to-community matching
- outreach draft generation
- approval queue
- follow-up reminders
- no-contact/deduplication controls
- delivery history

Dashboard widgets:
- best community opportunities this month
- warm relationships with upcoming events
- outreach needing approval
- follow-ups due
- talks that match active groups

Exit criteria:
- a user can find a relevant group, inspect why it is a fit, prepare a personalized pitch, approve it, track the relationship and record the speaking outcome.

## Phase 4 — Content Studio and media automation

Goal: connect research/events to repeatable content output.

Deliverables:
- content backlog
- content briefs
- editorial statuses
- content calendar
- source-evidence linkage
- campaign linkage
- transcript ingestion
- clip candidate detection
- subtitle generation
- FFmpeg render jobs
- aspect-ratio variants
- metadata/title/description drafts
- asset review/download
- distribution jobs

Exit criteria:
- a delivered talk or research cluster can become an article/clip/social package without duplicating planning work.

## Phase 5 — MCP and automation control plane

Goal: make DevRelOS programmable from agents and external workflows.

Deliverables:
- authenticated MCP server
- project-scoped authorization
- resources for events, pain points, communities, talks and campaigns
- tools for search/scoring/draft creation
- approval-aware write tools
- webhook triggers/actions
- scheduled workflows
- run history
- cost/usage visibility

Initial MCP tools:
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
- `analytics.get_scorecard`

Exit criteria:
- an authenticated agent can inspect DevRelOS context and safely create approved-domain work without direct DB access.

## Phase 6 — Developer feedback and product loop

Goal: connect advocacy to engineering/product decisions.

Deliverables:
- feedback inbox
- duplicate clustering
- persona/component mapping
- evidence attachments
- impact scoring
- GitHub issue linking
- status sync where permitted
- docs-gap workflow
- "you asked, we shipped" follow-up tasks

Exit criteria:
- a recurring community issue can be turned into an engineering action and closed with a developer-facing follow-up.

## Phase 7 — Campaigns, launches and community operations

Deliverables:
- launch campaign templates
- community call workflows
- office-hours workflows
- webinar workflows
- hackathon/workshop tracking
- ambassador/champion records
- partner campaigns
- sponsorship/booth tracking
- task dependencies
- calendar integrations

## Phase 8 — Analytics and attribution

Goal: measure outcomes, not activity volume alone.

Deliverables:
- scorecard framework
- activity metrics
- reach/engagement metrics
- CFP acceptance and speaking conversion
- content performance
- community relationship health
- developer activation funnel adapters
- product feedback closure rate
- campaign attribution
- cost per outcome

Recommended scorecard categories:
- Reach
- Engagement
- Education
- Activation
- Adoption
- Contribution
- Retention
- Advocacy
- Product learning
- Ecosystem influence

## Suggested first vertical slice

Build this before expanding broadly:

```text
CNCF/Developer event discovered
 -> CFP scored
 -> talk selected
 -> submission tracked
 -> event accepted/delivered
 -> recording/transcript attached
 -> clips/content tasks generated
 -> community relationship updated
 -> outcome recorded in scorecard
```

This demonstrates why DevRelOS is an operating system rather than another social scheduler.

## Non-goals for the first releases

- building a full social network
- replacing a full CRM
- replacing Jira/GitHub Issues
- building a full browser-based nonlinear video editor
- fully autonomous mass outreach
- scraping authenticated/private communities
- premature microservices
- an AI chat box with no structured domain workflow
