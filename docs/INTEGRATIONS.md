# Integrations and Data Sources

DevRelOS should use a provider architecture. A source can be enabled, disabled or replaced without changing the core domain model.

## Event and CFP sources

### developers.events

Public conference and CFP feeds are available from:

- `https://developers.events/all-events.json`
- `https://developers.events/all-cfps.json`

Use cases:
- event discovery
- CFP discovery
- closing-soon monitors
- event metadata enrichment

Important licensing note: the developers.events site/code and its data do not use the same license. Treat ingestion/redistribution of its data as a licensing-policy concern, especially for a commercial DevRelOS deployment. Keep this connector optional and preserve attribution/source links.

### CNCF Open Community Groups (OCG)

Relevant properties:
- public group and event discovery
- CNCF community groups are now powered by OCG
- OCG is open source and can be inspected/contributed to

Use cases:
- discover cloud-native communities
- group activity/cadence
- upcoming events
- public organizer/group metadata when legitimately exposed
- speaking-fit scoring

Do not assume a stable public API until the deployed interface is verified. Implement this connector behind the same provider contract so its ingestion method can evolve without touching the product model.

### CNCF event pages

Use for CNCF-hosted event/KCD discovery and enrichment where permitted.

### Sessionize

Useful because many developer events use Sessionize. Support event-specific public feeds when organizers expose them. Do not assume every event exposes the same fields.

### Other event sources

Potential adapters:
- event organizer RSS/Atom feeds
- community calendars / iCal feeds
- event websites with explicit structured data
- user-imported CSV/ICS
- Eventbrite/Luma or other platforms through official APIs/authorized integrations where available

## Developer signal sources

### Reddit

Use approved Reddit developer/data APIs. Do not make unauthorized scraping a core dependency.

Useful signals:
- recurring questions
- complaints/friction
- comparisons
- migration stories
- release reactions
- project mentions
- unanswered technical threads

Connector requirements:
- approval/config status
- rate-limit tracking
- source permalink
- subreddit/community
- author handling consistent with policy
- deletion/retention handling

### X

Use the official X API for supported search/read/write operations.

Because reads are paid, DevRelOS should expose:
- per-connector budget
- estimated/actual cost
- request deduplication/caching
- narrow saved searches
- daily caps
- kill switch when budget is reached

### Bluesky

Useful for public developer conversation monitoring because public API endpoints are broadly available and the AT Protocol supports streaming/firehose patterns.

Use cases:
- keyword/topic monitors
- project mentions
- developer discussion trends
- public author/list tracking

### Mastodon / ActivityPub

Support instance-aware ingestion through public/authorized APIs. Respect each instance's rate limits and policies.

### Hacker News

Support public item/search feeds for:
- launch reactions
- developer-tool pain points
- architecture discussions
- competitor/project mentions

### GitHub

First-class integration:
- issues
- discussions when accessible
- pull requests
- releases
- repository activity
- contributors
- issue labels/topics

Use cases:
- product feedback
- docs friction
- OSS contribution tracking
- launch/release triggers
- developer questions
- ecosystem relationship signals

### Stack Exchange / Stack Overflow

Useful for recurring technical questions and docs gaps via supported APIs.

### DEV / Forem

Use supported APIs/RSS for developer articles and discussions.

### RSS / Atom

High-value low-friction source type for:
- project blogs
- vendor engineering blogs
- release feeds
- community newsletters
- conference updates

### Web monitor

A generic web monitor should be conservative:
- user supplies URLs/domains or a search task
- obey robots/terms
- store canonical URL and minimal extract needed for the workflow
- throttle aggressively
- identify the DevRelOS crawler
- support domain allow/deny lists
- do not bypass authentication/paywalls/anti-bot measures

## Content and distribution integrations

### GitHub

- docs repositories
- sample application repos
- issue/PR feedback loop
- release triggers

### YouTube

- video metadata
- channel/video analytics when authorized
- upload/publish integration later
- transcript/description linkage

### Blog/CMS

Provider adapters can support:
- Git-based content repos
- WordPress
- Ghost
- Contentful/Sanity-style APIs
- generic webhook publishing

### Social publishing

Treat each channel independently. Publishing permissions and policies differ from read/search access.

Initial design:
- create distribution jobs in DevRelOS
- render channel-specific copy/assets
- require approval
- publish through an authorized connector
- record returned post ID/URL
- ingest metrics later

## Media integrations

### FFmpeg

Use as the initial server-side rendering/transcoding engine for:
- aspect-ratio variants
- clipping
- loudness normalization
- caption burn-in
- thumbnails/contact sheets
- audio extraction

### Whisper-compatible transcription

Use a provider interface for transcription so local/open models and hosted services can be swapped.

### Desktop/open editor handoff

Possible integrations/export formats for:
- Kdenlive
- Shotcut
- other editors that can consume media, captions and edit decision metadata

DevRelOS should orchestrate and hand off assets; it does not need to reproduce a full timeline editor in the browser for MVP.

## Automation integrations

### Temporal

Candidate for durable workflow execution when workflows become long-running/retry-heavy.

### Node-RED / generic webhooks

Support webhooks so users can plug DevRelOS into visual automation tools without making those tools the source of truth.

### GitHub Actions

Useful for technical workflows:
- build demo assets
- publish docs
- run sample repositories
- trigger release-content workflows

## Calendar and communication integrations

Potential connectors:
- Google Calendar
- Outlook Calendar
- Slack
- Discord
- email providers

Use cases:
- CFP deadlines
- speaking events
- community calls
- follow-up reminders
- approval notifications
- campaign milestones

## CRM / revenue attribution

Optional later connectors:
- HubSpot
- Salesforce
- product analytics
- warehouse/BI

The purpose is not to turn DevRelOS into a sales CRM. The connector should answer questions like:
- Did developers exposed to a campaign activate?
- Did an event influence product-qualified accounts?
- Which community partnerships create durable adoption?

## Connector policy metadata

Every connector definition should include:
- provider ID
- auth method
- capabilities
- rate-limit model
- cost model
- retention constraints
- content-license notes
- polling minimum
- deletion/update behavior
- supported write actions
- whether writes require human approval

This metadata should be visible in the admin UI so teams know the operational and policy consequences of enabling a source.

---

## Customising signal clustering

Pain-point clustering is a deterministic keyword heuristic, not an LLM, so
cluster rebuilds are reproducible and the product works offline. The trade-off
is that the vocabulary matters: the built-in topic and friction terms are tuned
for platform engineering, Kubernetes, CI/CD, observability and developer
experience. Signals from another domain will cluster into the generic
`general-developer-friction` bucket until the vocabulary is replaced.

### Which sources can produce pain points

Not every source can. Providers declare the *shape* of what they emit:

| Shape | Providers | Feeds pain points? |
| --- | --- | --- |
| `report` — the author has a problem | GitHub Issues, GitHub Discussions, Hacker News, Bluesky | yes |
| `announcement` — the author is publishing news | RSS/Atom feeds, GitHub Releases | no |
| `unknown` — unclassified, including manual entry | manual signals, rows predating this column | yes |

Announcement sources stay fully useful in Signal Radar and for content
research. They are excluded from pain-point evidence because friction
vocabulary is ordinary technical prose in a blog post: "setup", "configure",
"complex", "manual" and "hard" appear constantly in writing that *describes*
rather than complains, and keyword matching cannot tell "how to configure X"
from "configuring X is painful".

This is not a tuning problem. On a real dataset of 191 signals drawn from the
Kubernetes blog, the CNCF blog, GitHub releases and Hacker News, the clusterer
reported 18 pain points at severity up to 100 — including one backed by 73
Kubernetes version tags and another backed by 47 unrelated news headlines. With
source shapes applied, the same dataset yields the one pain point that the
data actually supports.

**If pain-point discovery matters to you, point connectors at complaint-shaped
sources**: GitHub Issues and Discussions on your own repositories, plus
whatever forum your developers actually complain in. Blog and release feeds are
for the content radar.

Two further guards apply:

- A signal must express **some** friction term to be evidence at all. Content
  matching none of the friction vocabulary produces no pain point rather than
  falling into a catch-all bucket.
- A cluster needs at least `minEvidence` signals (default **2**) before it is
  reported, because these are presented as *recurring* pain points. Single
  friction signals remain visible in Signal Radar.

Point `DEVRELOS_CLUSTERING_RULES` at a JSON file to override it:

```json
{
  "topics": [
    {
      "key": "robotics",
      "label": "Robotics",
      "persona": "Robotics engineer",
      "terms": ["ros2", "gazebo", "motion planning"]
    },
    {
      "key": "firmware",
      "label": "Firmware and embedded",
      "persona": "Embedded engineer",
      "terms": ["firmware", "rtos", "flashing", "bootloader"]
    }
  ],
  "frictions": [
    { "key": "calibration", "label": "calibration drift", "terms": ["calibration", "drifts", "recalibrate"] }
  ],
  "minEvidence": 2
}
```

Rules:

- Both sections are optional. Omitting a section keeps the built-in vocabulary
  for that section, so you can replace topics without restating every friction
  term.
- `key` is required and must be unique; `label` defaults to `key`; `persona` is
  optional and surfaces on the cluster.
- `terms` is required and non-empty. Terms are lowercased and trimmed on load
  and matched as **whole words** against the signal title and body, allowing
  common inflections: `cost` matches "costs", `configure` matches "configured"
  and "configuring", but neither matches "Costa Rica" or "Hardware".
  Multi-word terms such as `platform engineering` match as phrases.
- `minEvidence` sets how many signals a cluster needs before it is reported.
  It defaults to 2 and must be at least 1. Setting it to 1 surfaces every
  friction signal as its own pain point.
- Unknown fields are rejected, so a typo fails loudly instead of being ignored.
- **A missing or malformed file is a startup failure, not a fallback.** Silently
  reverting to the built-ins would leave you believing your vocabulary was in
  effect while every cluster was computed with the wrong terms.

When running under Compose, the path must be readable *inside* the container —
mount the file and point the variable at the mounted path:

```yaml
services:
  api:
    volumes:
      - ./clustering-rules.json:/etc/devrelos/clustering-rules.json:ro
    environment:
      DEVRELOS_CLUSTERING_RULES: /etc/devrelos/clustering-rules.json
```

Changing the vocabulary changes how future clusters are grouped. Rebuild
clustering afterwards so existing pain points are recomputed against the new
terms.
