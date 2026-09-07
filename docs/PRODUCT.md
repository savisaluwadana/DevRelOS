# DevRelOS Product Scope

## Product thesis

DevRel teams usually operate across disconnected tools: spreadsheets for CFPs, social feeds for research, Notion for content, CRM-like sheets for organizers, calendars for events, video tools for clips, GitHub for technical work, and analytics dashboards for reporting. DevRelOS should become the system that connects those activities into one operating loop.

The unit of value is not a post, event or task. It is a **developer relationship or developer outcome** that can be traced from signal -> action -> engagement -> learning -> result.

## Primary users

- Developer advocates and DevRel engineers
- Developer marketing and community teams
- Open-source project maintainers
- Platform/devtool companies running technical communities
- Founder-led developer-tool companies without a dedicated DevRel team

## Core workflows

### 1. Signal Radar

Collect and normalize public developer signals from approved sources.

Capabilities:
- saved searches and watchlists
- source filters and date windows
- deduplication and canonical URLs
- topic/entity extraction
- pain-point detection
- sentiment and urgency hints
- evidence-backed clusters
- trend velocity
- competitor/project mentions
- alerts when a tracked theme spikes
- convert a signal or cluster into a content brief, feedback item, campaign or outreach opportunity

### 2. Topic and pain-point intelligence

A topic is a durable theme such as "Kubernetes multi-tenancy". A pain point is an evidence-backed problem such as "platform teams cannot give application teams self-service without exposing raw Kubernetes complexity".

Each pain point should retain:
- supporting signals
- affected persona
- products/projects mentioned
- frequency and recency
- severity/urgency
- suggested content angles
- suggested demos/tutorials
- product-feedback implications
- events or communities where the topic is relevant

### 3. Events and CFP operating system

Capabilities:
- ingest conference/event sources
- open/closing-soon CFP views
- filters by region, topic, audience, cost, format and date
- CFP fit score against project goals and talk inventory
- deadline reminders
- submission pipeline
- submission owner
- abstract/version history
- acceptance/rejection tracking
- travel/logistics status
- calendar sync
- post-event follow-up and content repurposing

Suggested CFP states:

`discovered -> shortlisted -> preparing -> submitted -> accepted/rejected/waitlisted -> scheduled -> delivered -> follow-up -> archived`

### 4. Talk library

Store reusable talk assets rather than rewriting submissions from scratch.

Each talk can contain:
- title variants
- abstract variants
- target personas
- topics/tags
- length variants (5/15/30/45/60 minutes)
- level
- demo requirements
- speaker notes
- slide/repository/video links
- conferences submitted to
- acceptance rate
- delivery history
- feedback
- clips and derivative content

### 5. Community Graph and speaking outreach

Track communities and organizers as relationships, not scraped leads.

Community record:
- community/group name
- ecosystem (CNCF, Kubernetes, platform engineering, language, local tech, etc.)
- location/timezone
- public URL
- organizer contacts when legitimately public or user-supplied
- member/attendance indicators when available
- activity recency
- event cadence
- recurring topics
- recent speakers
- preferred formats
- upcoming events
- relationship owner
- relationship health
- last/next touchpoint
- outreach status

Speaking opportunity scoring should consider:
- topic fit
- audience fit
- geography/timezone
- activity recency
- event frequency
- prior relationship strength
- past acceptance
- available talk fit
- expected impact
- effort/cost

Suggested relationship pipeline:

`discovered -> researched -> warm -> pitched -> follow-up -> accepted -> scheduled -> delivered -> nurture`

DevRelOS should not mass-message organizers. It should prepare evidence-based, personalized outreach and require human approval by default.

### 6. Content Studio

Manage the full lifecycle from idea to distribution and repurposing.

Content types:
- tutorials
- technical blogs
- documentation improvements
- release explainers
- comparison/architecture articles
- social posts
- newsletters
- livestreams
- webinars
- demos
- podcasts
- conference talks
- short-form clips
- community updates
- case studies

Capabilities:
- content backlog tied to source evidence
- content brief generation
- technical review state
- editorial calendar
- asset checklist
- distribution plan
- UTM/campaign tracking
- repurposing graph (talk -> article -> clips -> social -> newsletter)
- evergreen refresh reminders

### 7. Media automation

DevRelOS should orchestrate media tools rather than try to become a full nonlinear video editor.

Examples:
- ingest a talk/webinar recording
- transcribe
- detect candidate clips
- create subtitles
- normalize audio
- render vertical/landscape variants
- generate chapters
- export metadata/title/description drafts
- push render jobs to supported tools

The initial rendering engine can be FFmpeg-based. External adapters can later integrate desktop or service-based editors.

### 8. Developer feedback and advocacy-to-product loop

DevRel is often the highest-bandwidth interface between engineering teams and users. Track that deliberately.

Capabilities:
- convert community/social/support signals into feedback
- cluster duplicate requests
- map feedback to persona/product/component
- attach evidence
- severity/reach scoring
- product-team owner/status
- GitHub issue linkage
- publish back a "you asked, we shipped" follow-up task

### 9. Community support and developer success

Track:
- recurring questions
- unanswered community threads
- response SLA
- office-hours questions
- docs gaps
- onboarding friction
- sample/demo requests
- support-to-doc conversion

### 10. Release and launch campaigns

For a release, DevRelOS should create a campaign containing:
- technical narrative
- target audiences
- docs/tutorial work
- demo assets
- community briefings
- social/content schedule
- livestream/webinar
- partner amplification
- feedback watch
- post-launch report

### 11. Community calls, webinars and office hours

Capabilities:
- recurring session calendar
- agenda backlog
- speaker/guest management
- reminder workflow
- attendee registration links
- recording/transcript links
- Q&A extraction
- follow-up actions
- clip/content generation
- attendance and retention trends

### 12. Ambassador/champion programs

Track:
- champions/ambassadors
- contribution history
- interests/regions
- speaking availability
- content contributions
- referrals
- recognition/swags/tasks
- program health

### 13. OSS and ecosystem engagement

DevRel engineering work should be first-class:
- issues/PRs created
- demos and sample applications
- integrations
- docs contributions
- upstream contributions
- ecosystem partner work
- GitHub Discussions/community participation
- maintainer relationships

### 14. Hackathons, workshops and labs

Track event design, challenges, sample repos, mentors, submissions, participant activation and follow-up.

### 15. Sponsorships and booths

Optional later module:
- sponsorship inventory
- deadlines and deliverables
- booth staffing
- demos
- badge/lead capture imports
- follow-up ownership
- cost and attributable outcomes

### 16. Developer journey and activation analytics

DevRel should eventually connect activity to outcomes such as:
- documentation visits
- quickstart completion
- sign-ups
- API keys/workspaces created
- first successful API call/deployment
- GitHub stars/forks/contributors
- community joins
- returning developers
- product-qualified developer accounts

Do not collapse these into vanity reach metrics. Preserve the funnel from awareness to technical adoption.

## Command Center

The default dashboard should answer:

- What requires action today?
- Which CFPs are closing soon?
- Which communities should we contact next?
- Which developer pain points are trending?
- Which content items are blocked?
- Which outreach messages need approval?
- What launches/events are upcoming?
- What developer feedback needs product follow-up?
- What moved in our DevRel scorecard this week?

## Core entities

- Workspace
- Project/Product
- Persona
- Source
- Signal
- Topic
- PainPoint
- Community
- Contact
- Relationship
- Touchpoint
- Event
- CFP
- Submission
- Talk
- ContentItem
- Asset
- Campaign
- DeveloperFeedback
- Task
- Automation
- AutomationRun
- Metric
- Attribution
- Connector

## Outcome model

Every activity should be able to link to one or more objectives:

- awareness
- education
- activation
- adoption
- contribution
- retention
- advocacy
- ecosystem partnership
- product learning
- revenue/pipeline influence

This allows a team to compare activities on outcomes rather than simply counting posts or events.
