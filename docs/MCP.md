# DevRelOS MCP Server

DevRelOS exposes its domain workflows to agents through an MCP server in `services/mcp`.

The MCP process is intentionally a thin client of the authenticated DevRelOS API. It does **not** connect to PostgreSQL and does not bypass the domain validation/state machines implemented by the API.

## Transport

The initial transport is stdio so the server can be launched by local IDEs, coding agents and desktop MCP clients without opening another network listener.

Run it from the repository root:

```bash
DEVRELOS_API_URL=http://127.0.0.1:8080 go run ./services/mcp
```

Configuration:

- `DEVRELOS_API_URL` — DevRelOS API base URL. Defaults to `http://127.0.0.1:8080`.
- `DEVRELOS_API_TOKEN` — optional bearer token forwarded to the DevRelOS API. The MCP process does not persist it.

Production deployments should use an authenticated API and provide the token through the process environment or secret manager.

## Read tools

- `signals_search`
- `pain_points_list`
- `pain_points_evidence`
- `work_items_list`
- `content_assets_list`
- `feedback_list`
- `events_list`
- `cfps_list`
- `talks_list`
- `communities_list`
- `outreach_list`
- `speaking_opportunities_list`
- `cfp_opportunities_list`

Tool names use underscores for compatibility with MCP clients that still apply stricter historical name validation.

## Internal workflow actions

The first write-capable tools only create internal DevRelOS state:

- `pain_points_create_work_item` — pain point → Action Queue
- `pain_points_create_feedback` — pain point → product feedback + reviewable GitHub issue draft
- `work_items_create_content` — Action Queue → Content Studio brief
- `submissions_create_draft` — CFP/talk pair → internal submission draft

These tools do **not** send outreach, create a GitHub issue, publish content, or submit a CFP externally. External effects remain behind explicit approval/delivery integrations.

## Design rules

1. MCP calls the DevRelOS API; it never receives database credentials.
2. Existing API validation, deduplication and workflow state machines remain authoritative.
3. Secrets are supplied through environment variables, not MCP tool arguments or database-backed connector config.
4. Search/list operations are bounded to avoid returning unbounded context.
5. External side effects must be modeled separately with authorization, approval, idempotency and auditability.

## Planned MCP expansion

Later tranches can add domain resources such as `devrel://pain-points/{id}`, `devrel://communities/{id}` and `devrel://talks/{id}`, Streamable HTTP for remote agents, OAuth/service authentication, additional approved mutation tools, and analytics scorecards.
