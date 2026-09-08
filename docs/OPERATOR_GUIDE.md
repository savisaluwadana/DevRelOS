# DevRelOS Operator Guide

> This guide is the per-workspace **reference**. For how the surfaces connect, the two operating modes and the end-to-end workflow chains, start with the [Workflow Guide](WORKFLOW.md).

This guide explains how to install, configure and use DevRelOS as a day-to-day developer-relations operating system. It covers the supported self-hosted deployment, identity and access, every major workspace in the operator console, the recommended operating workflows, connectors, MCP, media, campaign attribution, security, backups and troubleshooting.

DevRelOS is designed around three operating loops:

1. **Discover** — collect developer signals, recurring pain points, events, CFPs and communities.
2. **Engage** — turn evidence into work, content, talks, relationships, outreach, feedback and campaigns.
3. **Measure** — record outcomes and attribute real work back to campaigns and developer impact.

The intended workflow is not "collect as much data as possible." The intended workflow is:

```text
source evidence
    -> signal
    -> recurring pain point / opportunity
    -> prioritized action
    -> human-reviewed execution
    -> measurable outcome
    -> follow-up / learning
```

---

## 1. What is included

The current self-hosted beta includes:

- Command Center
- Unified DevRel Calendar
- Signal Radar and recurring pain-point clustering
- event and CFP tracking
- reusable talk library and submission pipeline
- opportunity intelligence for CFPs and community speaking
- communities, contacts, relationships and touchpoints
- Relationship Radar
- approval-gated outreach and optional SMTP delivery
- Action Queue
- Content Studio
- Developer Feedback workflow
- GitHub feedback issue prefill and linked public-issue state sync
- Media Studio with transcript/clip workflow and FFmpeg rendering
- Campaigns & Attribution
- connector scheduling and run telemetry
- encrypted connector credentials
- workspace users, RBAC, API keys, browser sessions and invitations
- authenticated MCP server
- Docker Compose local and single-node production deployment
- metrics, request IDs, health/readiness, backup and restore

PostgreSQL is the system of record. Redis, Kafka, Elasticsearch and other infrastructure are intentionally not required for the base deployment.

---

## 2. Fastest way to run DevRelOS locally

### Requirements

Install:

- Docker with Compose v2
- Git

For native development also install:

- Go 1.25+
- Node.js 22+

### Clone and start

```bash
git clone https://github.com/savisaluwadana/DevRelOS.git
cd DevRelOS
cp .env.example .env
make up
```

Open:

```text
http://localhost:3000
```

Health checks:

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

The Compose startup path runs all unapplied tracked database migrations automatically.

### What you should see first

The left navigation contains the main operator workspaces. In local mode you can start with the default workspace/project and immediately add events, talks, communities, signals or connectors.

For a first test, use this sequence:

1. Add one talk in **Talk Library**.
2. Add one event and CFP in **Events & CFPs**.
3. Open **Opportunities** to see CFP/talk fit.
4. Create a submission draft.
5. Add a community and relationship.
6. Add a follow-up date and open **Calendar**.
7. Add a manual developer signal or configure a connector.
8. Convert a pain point into an **Action Queue** item or **Developer Feedback** item.
9. Create a content asset from the Action Queue.
10. Create a campaign and attribute the content/work/outreach to it.

That walkthrough exercises the main DevRelOS loop without requiring any external write integration.

---

## 3. Production deployment

The supported v1 production topology is a **single-node self-hosted deployment** with:

- Caddy for HTTPS
- Next.js web console
- Go API
- Go worker
- PostgreSQL
- persistent media storage

Create a deployment `.env` with at least:

```text
DEVRELOS_DOMAIN=devrel.example.com
POSTGRES_DB=devrelos
POSTGRES_USER=devrelos
POSTGRES_PASSWORD=<long-random-password>
DEVRELOS_API_TOKEN=<long-random-break-glass-token>
DEVRELOS_SECRET_KEY=<base64-32-byte-key>
```

Generate the encryption key:

```bash
openssl rand -base64 32
```

Validate and start:

```bash
make prod-config
make prod-up
```

Production logs:

```bash
make prod-logs
```

Stop:

```bash
make prod-down
```

DNS for `DEVRELOS_DOMAIN` must point to the server. Ports 80 and 443 must be reachable so Caddy can provision and renew certificates.

Do not commit the deployment `.env` file.

---

## 4. Identity, users and workspaces

### Bootstrap credential

`DEVRELOS_API_TOKEN` is the root/bootstrap credential. Treat it as a break-glass secret, not a normal user login.

Use it to provision the first owner from **Access & Security**. After that, create a normal user API key.

### API keys

User API keys start with:

```text
drk_
```

The plaintext key is shown only when created. DevRelOS stores only its hash.

Use API keys for:

- MCP
- CLI/API access
- automation
- starting a browser session

### Browser sessions

When browser sessions are enabled, the login page exchanges a valid `drk_...` API key for a short-lived session token beginning with:

```text
ds_
```

The session is stored in an HttpOnly, SameSite=Strict cookie. Production cookies are Secure.

### Roles

Workspace roles are:

- **owner** — full administration, including owner grants
- **admin** — membership/security administration and normal writes
- **editor** — normal DevRel workflow writes
- **viewer** — read only

### Invitations

Owners/admins can create one-time workspace invitations from **Access & Security**.

Invitation tokens begin with:

```text
di_
```

Send invitation links through a private channel. Invitations have a role, expiry and revocation state.

### Multiple workspaces

If a user belongs to more than one workspace, use the workspace switcher in the sidebar. Sessions are pinned to the active workspace, so query-string changes cannot be used to escape workspace boundaries.

---

## 5. Command Center

The **Command Center** is the daily starting point.

Use it to answer:

- What needs attention today?
- Which CFPs are closing soon?
- Which submissions are in flight?
- Which communities are high fit?
- Which relationships are at risk?
- Which campaigns are active?
- Are campaigns producing measurable outcomes?

Recommended routine:

1. Open Command Center at the start of the day.
2. Review deadline risk.
3. Review relationship risk.
4. Move urgent work into **Action Queue**.
5. Use **Calendar** for chronological planning.
6. Use **Campaigns & Attribution** for outcome review.

The dashboard is intentionally an operating view, not a vanity-metrics page.

---

## 6. Unified DevRel Calendar

The **Calendar** combines dates from multiple DevRelOS domains into one operating timeline.

It currently includes:

- event start/end dates
- CFP closing dates
- Action Queue due dates
- scheduled content
- campaign start dates
- campaign target end dates
- relationship follow-up dates

### Why use it

DevRel work usually fails because deadlines live in separate systems. A CFP may close while content is due and a community follow-up is overdue. The calendar exposes those conflicts in one place.

### Recommended use

For every operational item that has a real deadline, add a date in its source workspace instead of creating a duplicate calendar-only object.

Examples:

- Set a CFP close time in **Events & CFPs**.
- Set a due date on an **Action Queue** item.
- Schedule an approved content asset in **Content Studio**.
- Add `next follow-up` to a relationship.
- Set campaign start/end dates.

The calendar then derives its timeline from those authoritative records.

### Priority

Calendar entries also carry an operating priority. CFP deadlines and overdue relationship follow-ups are intentionally surfaced more aggressively than low-risk planning dates.

---

## 7. Signal Radar

**Signal Radar** is where developer evidence enters the operating system.

Signals may come from:

- GitHub Issues
- GitHub Discussions
- GitHub Releases
- Bluesky
- Hacker News
- RSS/Atom
- manually captured evidence
- other provider adapters added later

### What a signal should contain

A useful signal has:

- a source/provider
- source URL/provenance
- title/body
- occurrence time
- topics
- engagement score where available

DevRelOS stores source provenance so downstream decisions can be traced back to evidence.

### Pain-point clustering

The pain-point engine groups recurring signals into stable pain-point records.

Use pain points to answer:

- What keeps recurring?
- Which developer persona is affected?
- Is severity increasing?
- How much evidence supports the conclusion?

Do not treat one social post as a validated market problem. Use the evidence view before converting a pain point into work.

### Converting pain into action

From a pain point you can create:

- content work
- documentation improvement
- product feedback
- talk idea
- community research
- engineering work
- other Action Queue items

You can also create a structured **Developer Feedback** item.

---

## 8. Integrations and connectors

Open **Integrations** to configure data-source connectors.

Each connector has:

- provider
- name
- enabled/disabled state
- provider-specific configuration
- optional encrypted credential reference
- schedule cadence
- run history

### Scheduling

Supported recurring schedules range from approximately every 15 minutes through weekly cadence depending on the selected option.

Use faster schedules only when the information changes quickly enough to justify the provider cost and rate-limit pressure.

### Run history

A connector run records operational data such as:

- queued/running/succeeded/failed state
- items fetched
- created/updated/skipped counts
- request count
- warnings
- provider cost where available
- failure details

Use run history before assuming a missing signal is an intelligence problem. It may simply be a connector/auth/config problem.

### GitHub Issues

Configure repository owner/repo, issue state, optional labels/query and optional authentication.

Pull requests are excluded from issue ingestion by default.

### GitHub Discussions

Requires authenticated GitHub GraphQL access. Use a workspace secret or supported legacy token configuration.

### GitHub Releases

Can read public repositories without a token, although authenticated access improves rate limits.

### RSS/Atom

The RSS connector accepts a feed URL. Production ingestion is HTTPS-only and includes SSRF protections. Do not weaken the URL validation to make internal/private URLs work; add an explicitly trusted integration path instead.

### developers.events

This source is useful for event/CFP discovery but its data licensing is not the same as its code licensing. DevRelOS keeps the connector policy-gated. Review the intended commercial/non-commercial use before enabling ingestion.

### Bluesky and Hacker News

These are useful for public developer conversation and launch/reaction signals. Use narrow queries rather than collecting unrelated public content.

### Reddit and X

Do not add unauthorized scraping. Use official APIs and comply with approval, pricing and platform terms.

---

## 9. Encrypted connector secrets

For production, use encrypted workspace secrets rather than plaintext connector JSON.

Set:

```text
DEVRELOS_SECRET_KEY=<base64-32-byte-key>
```

In **Access & Security**:

1. Create a secret.
2. Select the provider.
3. Give the secret an identifiable name.
4. Enter the credential value.
5. Attach the secret to the compatible connector.

Secret values are AES-256-GCM encrypted before database storage.

The UI/list APIs never return plaintext credentials. The worker decrypts an attached credential only in memory for the connector run.

To rotate a credential, rotate the workspace secret rather than putting a new token into connector JSON.

---

## 10. Events & CFPs

Use **Events & CFPs** to manage the event pipeline.

### Event statuses

Typical lifecycle:

```text
discovered -> tracking -> attending -> completed -> archived
```

Track:

- event name
- website
- location/timezone
- start/end
- event type
- topics

### CFP statuses

```text
upcoming -> open -> closed
```

A CFP can store:

- open/close times
- submission URL
- tracks
- requirements
- fit score

### Best practice

Record the actual closing timestamp, not only a date. It feeds the calendar and deadline intelligence.

---

## 11. Talk Library

The **Talk Library** is the reusable inventory of talks you can submit or pitch.

Track:

- title
- abstract
- description
- audience level
- duration
- topics
- demo/slides/recording links
- readiness state

Talk states:

```text
draft -> ready -> retired
```

A ready talk receives a stronger opportunity score than an unfinished draft.

Keep talk topics accurate. Topic overlap is part of CFP/community matching.

---

## 12. Opportunity Intelligence

Open **Opportunities** after you have talks plus communities and/or CFPs.

### CFP opportunities

DevRelOS scores CFP/talk pairs using inspectable factors such as:

- topic fit
- talk readiness
- deadline timing
- existing project fit
- duplicate submission penalty

Use **Create submission draft** to create an internal submission. DevRelOS does not automatically submit to an external CFP.

### Speaking opportunities

Community/talk ranking uses factors such as:

- topic overlap
- community quality/activity
- relationship strength
- talk readiness
- timing
- existing outreach penalties

Use the score as prioritization evidence, not as a guarantee that an organizer will accept a pitch.

---

## 13. Submission pipeline

A submission tracks a selected talk against a CFP.

Lifecycle:

```text
draft -> needs_work -> ready -> submitted -> accepted/rejected
```

Also supported:

```text
withdrawn
```

Record the real external state. Do not mark a draft as submitted just because the abstract is complete.

Accepted submissions feed campaign attribution and Command Center outcome metrics.

---

## 14. Communities, contacts and relationships

### Communities

A community record may include:

- platform/source
- website
- city/country/timezone
- topics
- member/activity data when available
- speaking-fit score
- next/last event

### Contacts

Use contacts for organizers and other relevant people.

A contact can be marked **do not contact**. Respect that state; outbound workflows reject prohibited contact attempts.

### Relationships

A relationship links the project to a community and/or contact.

Stages:

```text
cold -> warm -> engaged -> partner -> dormant
```

Track:

- relationship strength
- last touch
- next follow-up
- notes

### Touchpoints

Log meaningful interactions:

- email
- community chat
- call
- event interaction
- social conversation
- internal note

Touchpoints provide the relationship history used by Relationship Radar.

---

## 15. Relationship Radar

**Relationship Radar** converts relationship history into an action-oriented risk view.

It considers:

- relationship strength
- time since last touch
- overdue next follow-up
- lifecycle stage

Health states:

- **healthy**
- **watch**
- **critical**

Recommended actions include:

- follow up now
- reconnect with a useful update
- create a concrete collaboration next step
- maintain partner momentum
- research and make a useful first touch

Use the recommendation as a prompt for human judgment. Do not automate generic relationship spam.

---

## 16. Outreach

The outreach workspace is deliberately approval-gated.

Lifecycle:

```text
draft
 -> needs_approval
 -> approved
 -> queued
 -> sent | failed
 -> replied
```

### Drafting

An outreach item can link:

- community
- contact
- talk
- channel
- subject/body
- selection rationale

When coming from Opportunity Intelligence, keep the score/rationale but personalize the message.

### Approval

External delivery should happen only after explicit approval.

### Email delivery

Optional SMTP delivery is supported when configured.

Example environment variables:

```text
DEVRELOS_SMTP_ENABLED=true
DEVRELOS_SMTP_HOST=smtp.example.com
DEVRELOS_SMTP_PORT=587
DEVRELOS_SMTP_USERNAME=...
DEVRELOS_SMTP_PASSWORD=...
DEVRELOS_SMTP_FROM=devrel@example.com
DEVRELOS_SMTP_FROM_NAME=DevRel Team
DEVRELOS_SMTP_STARTTLS=true
DEVRELOS_SMTP_REQUIRE_TLS=true
DEVRELOS_OUTREACH_MAX_ATTEMPTS=5
```

Only successful SMTP delivery may mark email outreach as sent.

Non-email channels remain manual unless an explicitly authorized write connector is implemented.

---

## 17. Action Queue

The **Action Queue** is the execution layer between intelligence and work.

Kinds include:

- content brief
- documentation improvement
- product feedback
- talk idea
- community research
- outreach follow-up
- event task
- engineering task

Statuses:

```text
backlog -> planned -> in_progress -> done
```

Also:

```text
blocked
cancelled
```

Each work item can track:

- provenance/source
- priority
- owner
- due date
- metadata/topics

### Recommended workflow

1. Convert evidence into a work item.
2. Give it an owner and due date.
3. Move only committed work to `planned`.
4. Keep WIP limited.
5. Convert suitable work into Content Studio.
6. Attribute campaign-related work to a campaign.

Due dates automatically appear in the Unified Calendar.

---

## 18. Content Studio

**Content Studio** turns evidence-backed work into reviewable content assets.

Supported channels include:

- blog
- LinkedIn
- X
- newsletter
- YouTube
- short video
- docs
- talks
- community content

The content model separates:

- **brief** — evidence/context/editorial intent
- **draft** — actual copy

This distinction is intentional. Evidence assembly can be deterministic without pretending that generated prose is ready to publish.

Lifecycle:

```text
brief -> drafting -> review -> approved -> published -> archived
```

### Recommended flow

1. Start from an Action Queue item when possible.
2. Review the inherited evidence/provenance.
3. Draft the content.
4. Perform technical/editorial review.
5. Approve.
6. Schedule or publish externally.
7. Record the final published URL and state.
8. Attribute the asset to the relevant campaign.

Scheduled content appears in the Unified Calendar.

DevRelOS records publication; it does not currently claim universal autonomous publishing to every social/CMS destination.

---

## 19. Developer Feedback

The **Developer Feedback** workspace converts recurring developer pain into structured product learning.

Feedback stores:

- title/summary
- affected persona
- component
- impact score
- frequency score
- owner
- lifecycle status
- GitHub repository
- issue draft/linkage
- developer follow-up note
- source provenance

Lifecycle:

```text
new -> triaged -> planned -> in_progress -> shipped -> closed
```

Also supported:

```text
wont_fix
```

### Create from Signal Radar

This is the preferred flow for recurring developer pain because it preserves the evidence source and scoring.

### GitHub issue workflow

If `githubRepository` is configured as `owner/repo`:

1. Review/edit the generated issue title and body.
2. Click **Open prefilled GitHub issue**.
3. GitHub opens an issue form with the draft title/body prefilled.
4. Review and submit the issue explicitly on GitHub.
5. Return to DevRelOS and link the issue number/URL using **Edit / link GitHub**.
6. Click **Sync GitHub issue**.

For linked **public** issues, DevRelOS records synced information such as:

- open/closed state
- GitHub title/URL
- last GitHub update time
- author
- comment count
- labels
- last sync time

GitHub state does **not** automatically change the DevRelOS product-feedback lifecycle. This prevents a closed GitHub issue from being treated as a shipped developer outcome without human verification.

### Shipping feedback

Before marking feedback `shipped`, add a follow-up note explaining:

- what changed
- what developers should be told
- where the fix/docs/release can be found

That requirement protects the final loop:

```text
developer pain -> product work -> shipped change -> developer follow-up
```

---

## 20. Campaigns & Attribution

Campaigns group real DevRel activity under a measurable initiative.

A campaign has:

- name
- objective
- lifecycle status
- start/end
- optional budget
- target metadata

Lifecycle:

```text
planning -> active -> paused/completed -> archived
```

### What can be attributed

Campaigns can link records such as:

- content assets
- work items
- outreach
- feedback
- events
- CFPs/submissions
- talks
- communities
- media assets

### Quick attribution

Content Studio, Action Queue, Outreach and Feedback expose direct campaign attribution controls so operators do not need to copy internal IDs.

Optional attributed cost can be recorded at the activity link.

### Derived campaign outcomes

Campaign reports derive metrics from linked source records, including:

- published content
- completed work
- outreach sent
- outreach replies
- reply rate
- submissions
- accepted talks
- acceptance rate
- shipped feedback
- attributed spend
- budget remaining

The outcome score is deterministic and inspectable. It is not a manually entered "success" number.

### Manual metrics

You can add metrics that DevRelOS does not derive from its own records, for example:

- registrations
- workshop attendance
- product activations
- docs conversions
- qualified accounts

Use a consistent metric key across campaigns if you want meaningful comparisons.

### Removing attribution

Attribution is reversible. If activity was linked to the wrong campaign, remove the link rather than leaving polluted historical reporting.

---

## 21. Media Studio

Media Studio manages recording/transcript/clip workflows.

The initial server-side rendering engine is FFmpeg.

For local development, place media under:

```text
data/media/recordings/
```

Reference files relative to the configured media root.

Typical flow:

```text
recording
 -> transcript import
 -> segment/candidate selection
 -> clip review/approval
 -> FFmpeg render
 -> output variant
```

Use Media Studio to orchestrate clip creation and handoff. DevRelOS is not intended to reproduce every capability of a full non-linear video editor.

Production uses durable media storage. Back it up together with PostgreSQL.

---

## 22. MCP for agents and IDEs

DevRelOS exposes an MCP server in:

```text
services/mcp
```

It is a thin authenticated client of the Go domain API. It does not connect directly to PostgreSQL.

Run:

```bash
DEVRELOS_API_URL=http://localhost:8080 \
DEVRELOS_API_TOKEN="drk_..." \
go run ./services/mcp
```

Read tools include operations for:

- signals
- pain points/evidence
- work items
- content
- feedback
- events
- CFPs
- talks
- communities
- outreach
- speaking opportunities
- CFP opportunities

Internal write tools can create actions such as:

- pain point -> work item
- pain point -> feedback
- work item -> content brief
- CFP/talk -> submission draft

MCP writes go through the normal DevRelOS API validation and workspace permissions.

External side effects such as publishing, CFP submission or generic outreach sending are not silently bypassed through MCP.

---

## 23. A practical weekly operating rhythm

### Daily

1. Open **Command Center**.
2. Open **Calendar**.
3. Handle urgent CFP/follow-up risk.
4. Review connector failures.
5. Move committed work through Action Queue.
6. Log important relationship touchpoints.

### Twice per week

1. Review Signal Radar.
2. Rebuild/review pain-point clusters.
3. Convert meaningful pain into product/content/docs actions.
4. Review new speaking opportunities.
5. Review Content Studio editorial queue.

### Weekly

1. Review active campaign outcome reports.
2. Review critical/watch relationships.
3. Review CFP submission pipeline.
4. Review feedback planned/in-progress/shipped.
5. Sync linked GitHub feedback issues.
6. Review connector request/cost telemetry.
7. Plan the next week's calendar conflicts.

### Monthly

1. Archive stale signals/work/campaigns where appropriate.
2. Review community portfolio and dormant relationships.
3. Compare campaign outcomes, not only activity volume.
4. Rotate credentials according to your security policy.
5. Verify backups and test recovery procedures periodically.

---

## 24. Recommended end-to-end workflows

### A. Developer pain -> content

```text
GitHub/HN/Bluesky/RSS
 -> Signal Radar
 -> recurring pain point
 -> Action Queue content brief
 -> Content Studio
 -> review
 -> published URL
 -> campaign attribution
```

### B. Developer pain -> product feedback

```text
source evidence
 -> pain point
 -> Developer Feedback
 -> triage
 -> GitHub issue draft
 -> explicit GitHub issue creation
 -> linked issue sync
 -> engineering progress
 -> shipped
 -> developer follow-up
```

### C. CFP discovery -> accepted talk

```text
event connector/manual event
 -> CFP
 -> Opportunity Intelligence
 -> reusable Talk Library asset
 -> submission draft
 -> external submission
 -> submitted
 -> accepted/rejected
 -> campaign attribution
```

### D. Community -> speaking relationship

```text
community discovery
 -> speaking-fit ranking
 -> contact/research
 -> relationship
 -> useful touchpoints
 -> outreach draft
 -> approval
 -> delivery/manual send
 -> reply
 -> speaking opportunity
 -> nurture relationship
```

### E. Campaign execution

```text
campaign objective
 -> planned activities
 -> content/work/outreach/events/feedback
 -> quick attribution
 -> outcome metrics
 -> report
 -> learn and adjust
```

---

## 25. API usage

The Go API defaults to:

```text
http://localhost:8080
```

Local browser mutations should go through the Next.js same-origin proxy:

```text
/api/devrelos/...
```

Do not expose the root operator token in browser JavaScript.

Useful endpoints include:

```text
GET  /healthz
GET  /readyz
GET  /metrics
GET  /api/v1/dashboard
GET  /api/v1/calendar
GET  /api/v1/signals
GET  /api/v1/pain-points
GET  /api/v1/opportunities/speaking
GET  /api/v1/opportunities/cfps
GET  /api/v1/work-items
GET  /api/v1/content-assets
GET  /api/v1/feedback
GET  /api/v1/campaigns
GET  /api/v1/relationships/radar
GET  /api/v1/connectors
```

### Paging list endpoints

Every list endpoint accepts `limit` and `offset`, and every list query is
bounded. Omitting both returns the first 200 rows; `limit` may not exceed 500.
Out-of-range values fall back to the default rather than erroring.

```text
GET /api/v1/talks?limit=50
GET /api/v1/talks?limit=50&offset=50
```

There is no total count in the response. **A page shorter than the requested
`limit` means you have reached the end** — keep advancing `offset` by `limit`
until that happens.

This applies to `events`, `cfps`, `talks`, `submissions`, `communities`,
`contacts`, `relationships`, `outreach`, `campaigns`, campaign items, `calendar`,
media clips, `signals`, `pain-points`, `work-items`, `content-assets` and
`feedback`. Small administrative lists (workspace members, API keys, connectors,
connector secrets) are unpaged on purpose: they are bounded by team and
configuration size, and truncating them would hide members rather than page
them.

Opportunity ranking (`/api/v1/opportunities/*`) deliberately reads the complete
candidate set rather than a page, because it scores a cross product of
communities and talks — a truncated input would silently produce wrong
rankings. This is the one place scale will bite first on a very large project.

The web UI should be preferred for routine operations because it uses the same workflow constraints and makes state visible.

---

## 26. Observability

### Health

```text
GET /healthz
```

Confirms the API process is alive.

### Readiness

```text
GET /readyz
```

Checks database readiness.

### Metrics

```text
GET /metrics
```

The metrics endpoint is authenticated and exposes Prometheus-compatible request/process metrics.

API responses include `X-Request-ID`. Use it when correlating a UI error with API logs.

### Connector troubleshooting

For connector problems check, in order:

1. connector enabled state
2. provider configuration
3. secret attachment/credential validity
4. latest run state
5. run error/warnings
6. rate limits/provider policy
7. source record creation
8. normalized domain record creation

---

## 27. Backup and restore

Create a production backup:

```bash
make backup
```

Backups include PostgreSQL and media data with checksums.

Restore:

```bash
make restore BACKUP=./backups/<timestamp>
```

Restore is destructive and requires explicit confirmation.

Keep backups off-host according to your environment's policy.

Backups do not replace protecting `DEVRELOS_SECRET_KEY`. If the database is restored but the encryption key is lost, encrypted connector credentials cannot be recovered.

---

## 28. Troubleshooting

### Web UI says API unavailable

Check:

```bash
make ps
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
make logs
```

### Database migration failure

Check migration logs and confirm PostgreSQL is healthy. Do not manually skip a failed migration in production without understanding the schema state.

### Connector succeeds but no signals appear

A connector run can store source records without producing a signal if normalization/project resolution fails. Review run warnings and provider configuration.

### GitHub Discussion connector fails

GitHub Discussions uses GraphQL and requires authentication. Attach a valid GitHub credential.

### GitHub feedback sync says issue is not public

The current sync endpoint intentionally uses public GitHub issue reads. For a private repository, use the prefilled issue workflow and manually maintain linkage until an explicitly authenticated private-repository write/read adapter is configured.

### RSS URL rejected

Production RSS ingestion blocks localhost, private, loopback, link-local and other unsafe targets. This is an SSRF protection, not a bug.

### Outreach will not send

Check:

- status is approved/queued
- contact is not do-not-contact
- SMTP is enabled
- SMTP TLS/auth is valid
- delivery attempts have not exceeded the retry limit

### Media render path rejected

Media input/output must remain inside the configured media root. Do not use path traversal or arbitrary remote URLs.

### Browser session stops working

The session may be expired, revoked, user-disabled or no longer valid for the workspace. Sign in again with an active `drk_...` key.

---

## 29. Security rules operators should not bypass

1. Do not put `DEVRELOS_API_TOKEN` in `NEXT_PUBLIC_*` variables.
2. Do not store provider plaintext credentials in connector JSON for production.
3. Do not weaken RSS SSRF protections to fetch private/internal URLs.
4. Do not turn outreach into autonomous mass messaging.
5. Do not treat public APIs as permission to redistribute copyrighted/licensed data without reviewing terms.
6. Do not make GitHub issue closure automatically equal product delivery.
7. Do not expose PostgreSQL directly to the public internet.
8. Do not expose the Go API directly when the supported production topology expects Caddy/web as the edge.
9. Keep the encryption key stable and separately recoverable.
10. Test backup restoration before you need it.

---

## 30. Where to go next

Reference documents:

- `README.md` — product overview and fastest start
- `docs/RUNNING.md` — deployment/runtime instructions
- `docs/SECURITY.md` — trust model, RBAC, sessions and secrets
- `docs/INTEGRATIONS.md` — provider/data-source guidance
- `docs/MCP.md` — MCP server and tool catalog
- `docs/ARCHITECTURE.md` — system architecture
- `docs/PRODUCT.md` — product scope
- `docs/ROADMAP.md` — delivery direction

For routine operation, use this guide as the primary manual and the narrower documents above as implementation references.
