# DevRelOS MCP Server

DevRelOS exposes its domain workflows to agents through an MCP server in `services/mcp`.

The MCP process is intentionally a thin client of the authenticated DevRelOS API. It does **not** connect to PostgreSQL and does not bypass the domain validation/state machines implemented by the API.

## Transport

The initial transport is stdio so the server can be launched by local IDEs, coding agents and desktop MCP clients without opening another network listener.

Run it from the repository root:

```bash
DEVRELOS_API_URL=http://127.0.0.1:8080 \
DEVRELOS_API_TOKEN="drk_..." \
go run ./services/mcp
```

Configuration:

- `DEVRELOS_API_URL` — DevRelOS API base URL. Defaults to `http://127.0.0.1:8080`.
- `DEVRELOS_API_TOKEN` — optional bearer token forwarded to the DevRelOS API. The MCP process does not persist it.

Production deployments should use an authenticated API and a dedicated user/API key with the minimum workspace role required by the agent.

## Read tools

- `signals_search` — search normalized developer signals.
- `pain_points_list` — list recurring evidence-backed pain points.
- `pain_points_evidence` — inspect supporting evidence for one pain point.
- `work_items_list` — inspect the Action Queue.
- `content_assets_list` — inspect Content Studio assets and editorial state.
- `feedback_list` — inspect product feedback and GitHub linkage.
- `events_list` — list tracked events.
- `cfps_list` — list CFPs and deadlines.
- `talks_list` — list reusable talks.
- `communities_list` — list communities.
- `outreach_list` — list outreach state.
- `speaking_opportunities_list` — rank community/talk opportunities.
- `cfp_opportunities_list` — rank CFP/talk opportunities.
- `calendar_list` — read the unified calendar across CFPs, events, work, content, campaign milestones and relationship follow-ups. Optional `from`/`to` inputs use RFC3339.
- `campaigns_list` — list campaigns and lifecycle state.
- `campaign_report` — inspect one campaign's attribution, spend and derived outcome metrics.
- `relationship_radar_list` — read relationship health, risk and recommended actions.

Tool names use underscores for compatibility with MCP clients that still apply stricter historical name validation.

## Internal workflow actions

The write-capable tools only create internal DevRelOS state:

- `pain_points_create_work_item` — pain point → Action Queue.
- `pain_points_create_feedback` — pain point → product feedback + reviewable GitHub issue draft.
- `work_items_create_content` — Action Queue → Content Studio brief.
- `submissions_create_draft` — CFP/talk pair → internal submission draft.

These tools do **not** send outreach, create a GitHub issue, publish content, or submit a CFP externally. External effects remain behind explicit approval/delivery integrations.

## Example agent workflows

An agent can safely use the read tools to answer questions such as:

- "What are the five highest-evidence pain points and which already have work attached?"
- "What is due during the next two weeks?"
- "Which relationships are critical and overdue for follow-up?"
- "Which campaigns have activity but weak outcomes?"
- "Which CFP/talk pair should I prepare next?"

It can then use approved internal mutation tools to create a work item, feedback item, content brief or submission draft. The resulting object remains in DevRelOS for operator review.

## Design rules

1. MCP calls the DevRelOS API; it never receives database credentials.
2. Existing API validation, workspace isolation, deduplication and workflow state machines remain authoritative.
3. Secrets are supplied through process environment/secret management, not MCP tool arguments.
4. Search/list operations are bounded to avoid returning unbounded context.
5. External side effects must be modeled separately with authorization, approval, idempotency and auditability.
6. Agents should prefer read/plan/create-draft operations and leave irreversible external actions behind explicit approval gates.

## Planned MCP expansion

Later tranches can add domain resources such as `devrel://pain-points/{id}`, `devrel://communities/{id}` and `devrel://talks/{id}`, Streamable HTTP for remote agents, OAuth/service authentication, additional approved mutation tools, and richer analytics scorecards.
