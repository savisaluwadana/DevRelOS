package main

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listOutput struct {
	Items []map[string]any `json:"items" jsonschema:"DevRelOS domain objects returned by the query"`
}

type itemOutput struct {
	Item map[string]any `json:"item" jsonschema:"The DevRelOS domain object created by the action"`
}

type signalsSearchInput struct {
	Query    string `json:"query,omitempty" jsonschema:"Full-text developer signal query"`
	Provider string `json:"provider,omitempty" jsonschema:"Optional source provider filter, for example github.issues or bluesky"`
	Topic    string `json:"topic,omitempty" jsonschema:"Optional normalized topic filter"`
	Status   string `json:"status,omitempty" jsonschema:"Optional signal review status"`
	Limit    int    `json:"limit,omitempty" jsonschema:"Maximum results to return, capped at 200"`
}

type painPointsListInput struct {
	Status string `json:"status,omitempty" jsonschema:"Pain point status, normally active or archived"`
	Limit  int    `json:"limit,omitempty" jsonschema:"Maximum results to return, capped at 200"`
}

type evidenceInput struct {
	PainPointID string `json:"painPointId" jsonschema:"DevRelOS pain point UUID"`
	Limit       int    `json:"limit,omitempty" jsonschema:"Maximum evidence records to return, capped at 100"`
}

type workItemsListInput struct {
	Status string `json:"status,omitempty" jsonschema:"Optional Action Queue status filter"`
	Kind   string `json:"kind,omitempty" jsonschema:"Optional work kind filter"`
	Limit  int    `json:"limit,omitempty" jsonschema:"Maximum results to return, capped at 200"`
}

type contentListInput struct {
	Status  string `json:"status,omitempty" jsonschema:"Optional editorial status filter"`
	Channel string `json:"channel,omitempty" jsonschema:"Optional content channel filter"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Maximum results to return, capped at 200"`
}

type feedbackListInput struct {
	Status    string `json:"status,omitempty" jsonschema:"Optional product feedback status filter"`
	Component string `json:"component,omitempty" jsonschema:"Optional product component filter"`
	Limit     int    `json:"limit,omitempty" jsonschema:"Maximum results to return, capped at 200"`
}

type emptyInput struct{}

type painPointWorkInput struct {
	PainPointID string `json:"painPointId" jsonschema:"Pain point UUID to convert into work"`
	Kind        string `json:"kind" jsonschema:"Work kind such as content_brief, docs_improvement, product_feedback, talk_idea, community_research, outreach_follow_up, event_task, or engineering_task"`
	Owner       string `json:"owner,omitempty" jsonschema:"Optional work owner"`
}

type painPointFeedbackInput struct {
	PainPointID      string `json:"painPointId" jsonschema:"Pain point UUID to convert into product feedback"`
	Component        string `json:"component,omitempty" jsonschema:"Optional product component or area"`
	Owner            string `json:"owner,omitempty" jsonschema:"Optional feedback owner"`
	GitHubRepository string `json:"githubRepository,omitempty" jsonschema:"Optional target repository in owner/repo form; no issue is created automatically"`
}

type workContentInput struct {
	WorkItemID string `json:"workItemId" jsonschema:"Action Queue work item UUID"`
	Channel    string `json:"channel" jsonschema:"Content channel such as blog, linkedin, x, newsletter, youtube, short_video, docs, talk, or community"`
	Format     string `json:"format" jsonschema:"Format valid for the selected channel"`
	Audience   string `json:"audience,omitempty" jsonschema:"Optional target audience override"`
}

type submissionDraftInput struct {
	CFPID  string `json:"cfpId" jsonschema:"CFP UUID"`
	TalkID string `json:"talkId" jsonschema:"Talk UUID"`
	Notes  string `json:"notes,omitempty" jsonschema:"Optional internal submission notes"`
}

func registerTools(server *mcp.Server, api *apiClient) {
	mcp.AddTool(server, &mcp.Tool{Name: "signals_search", Description: "Search normalized developer signals in DevRelOS. Read-only."}, func(ctx context.Context, _ *mcp.CallToolRequest, in signalsSearchInput) (*mcp.CallToolResult, listOutput, error) {
		q := url.Values{}
		setQuery(q, "q", in.Query)
		setQuery(q, "provider", in.Provider)
		setQuery(q, "topic", in.Topic)
		setQuery(q, "status", in.Status)
		q.Set("limit", strconv.Itoa(boundedLimit(in.Limit, 50, 200)))
		items, err := api.list(ctx, "/api/v1/signals", q)
		return nil, listOutput{Items: items}, err
	})

	mcp.AddTool(server, &mcp.Tool{Name: "pain_points_list", Description: "List evidence-backed developer pain point clusters. Read-only."}, func(ctx context.Context, _ *mcp.CallToolRequest, in painPointsListInput) (*mcp.CallToolResult, listOutput, error) {
		q := url.Values{}
		setQuery(q, "status", in.Status)
		q.Set("limit", strconv.Itoa(boundedLimit(in.Limit, 50, 200)))
		items, err := api.list(ctx, "/api/v1/pain-points", q)
		return nil, listOutput{Items: items}, err
	})

	mcp.AddTool(server, &mcp.Tool{Name: "pain_points_evidence", Description: "Inspect the underlying signal evidence for a pain point. Read-only."}, func(ctx context.Context, _ *mcp.CallToolRequest, in evidenceInput) (*mcp.CallToolResult, listOutput, error) {
		q := url.Values{"limit": {strconv.Itoa(boundedLimit(in.Limit, 50, 100))}}
		items, err := api.list(ctx, "/api/v1/pain-points/"+url.PathEscape(strings.TrimSpace(in.PainPointID))+"/evidence", q)
		return nil, listOutput{Items: items}, err
	})

	mcp.AddTool(server, &mcp.Tool{Name: "work_items_list", Description: "List prioritized DevRel Action Queue work. Read-only."}, func(ctx context.Context, _ *mcp.CallToolRequest, in workItemsListInput) (*mcp.CallToolResult, listOutput, error) {
		q := url.Values{}
		setQuery(q, "status", in.Status)
		setQuery(q, "kind", in.Kind)
		q.Set("limit", strconv.Itoa(boundedLimit(in.Limit, 50, 200)))
		items, err := api.list(ctx, "/api/v1/work-items", q)
		return nil, listOutput{Items: items}, err
	})

	mcp.AddTool(server, &mcp.Tool{Name: "content_assets_list", Description: "List Content Studio assets and editorial states. Read-only."}, func(ctx context.Context, _ *mcp.CallToolRequest, in contentListInput) (*mcp.CallToolResult, listOutput, error) {
		q := url.Values{}
		setQuery(q, "status", in.Status)
		setQuery(q, "channel", in.Channel)
		q.Set("limit", strconv.Itoa(boundedLimit(in.Limit, 50, 200)))
		items, err := api.list(ctx, "/api/v1/content-assets", q)
		return nil, listOutput{Items: items}, err
	})

	mcp.AddTool(server, &mcp.Tool{Name: "feedback_list", Description: "List product feedback, engineering state, impact and GitHub linkage. Read-only."}, func(ctx context.Context, _ *mcp.CallToolRequest, in feedbackListInput) (*mcp.CallToolResult, listOutput, error) {
		q := url.Values{}
		setQuery(q, "status", in.Status)
		setQuery(q, "component", in.Component)
		q.Set("limit", strconv.Itoa(boundedLimit(in.Limit, 50, 200)))
		items, err := api.list(ctx, "/api/v1/feedback", q)
		return nil, listOutput{Items: items}, err
	})

	registerSimpleList(server, api, "events_list", "List tracked developer events. Read-only.", "/api/v1/events")
	registerSimpleList(server, api, "cfps_list", "List tracked CFPs and deadlines. Read-only.", "/api/v1/cfps")
	registerSimpleList(server, api, "talks_list", "List reusable talks in the Talk Library. Read-only.", "/api/v1/talks")
	registerSimpleList(server, api, "communities_list", "List discovered and managed communities. Read-only.", "/api/v1/communities")
	registerSimpleList(server, api, "outreach_list", "List speaking/community outreach records. Read-only.", "/api/v1/outreach")
	registerSimpleList(server, api, "speaking_opportunities_list", "Rank community and talk speaking matches. Read-only.", "/api/v1/opportunities/speaking")
	registerSimpleList(server, api, "cfp_opportunities_list", "Rank CFP and talk submission matches. Read-only.", "/api/v1/opportunities/cfps")

	mcp.AddTool(server, &mcp.Tool{Name: "pain_points_create_work_item", Description: "Create an internal Action Queue item from a pain point. This changes DevRelOS state but has no external side effect."}, func(ctx context.Context, _ *mcp.CallToolRequest, in painPointWorkInput) (*mcp.CallToolResult, itemOutput, error) {
		item, err := api.create(ctx, "/api/v1/pain-points/"+url.PathEscape(strings.TrimSpace(in.PainPointID))+"/work-items", map[string]any{"kind": in.Kind, "owner": in.Owner})
		return nil, itemOutput{Item: item}, err
	})

	mcp.AddTool(server, &mcp.Tool{Name: "pain_points_create_feedback", Description: "Create internal product feedback and a reviewable GitHub issue draft from a pain point. No GitHub issue is submitted."}, func(ctx context.Context, _ *mcp.CallToolRequest, in painPointFeedbackInput) (*mcp.CallToolResult, itemOutput, error) {
		item, err := api.create(ctx, "/api/v1/pain-points/"+url.PathEscape(strings.TrimSpace(in.PainPointID))+"/feedback", map[string]any{"component": in.Component, "owner": in.Owner, "githubRepository": in.GitHubRepository})
		return nil, itemOutput{Item: item}, err
	})

	mcp.AddTool(server, &mcp.Tool{Name: "work_items_create_content", Description: "Create an internal evidence-backed Content Studio brief from an Action Queue item. No content is published."}, func(ctx context.Context, _ *mcp.CallToolRequest, in workContentInput) (*mcp.CallToolResult, itemOutput, error) {
		item, err := api.create(ctx, "/api/v1/work-items/"+url.PathEscape(strings.TrimSpace(in.WorkItemID))+"/content-assets", map[string]any{"channel": in.Channel, "format": in.Format, "audience": in.Audience})
		return nil, itemOutput{Item: item}, err
	})

	mcp.AddTool(server, &mcp.Tool{Name: "submissions_create_draft", Description: "Create an internal CFP submission draft for an existing CFP/talk pair. This does not submit to an external CFP service."}, func(ctx context.Context, _ *mcp.CallToolRequest, in submissionDraftInput) (*mcp.CallToolResult, itemOutput, error) {
		item, err := api.create(ctx, "/api/v1/submissions", map[string]any{"cfpId": in.CFPID, "talkId": in.TalkID, "notes": in.Notes, "status": "draft"})
		return nil, itemOutput{Item: item}, err
	})
}

func registerSimpleList(server *mcp.Server, api *apiClient, name, description, path string) {
	mcp.AddTool(server, &mcp.Tool{Name: name, Description: description}, func(ctx context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, listOutput, error) {
		items, err := api.list(ctx, path, nil)
		return nil, listOutput{Items: items}, err
	})
}

func setQuery(values url.Values, key, value string) {
	if value = strings.TrimSpace(value); value != "" {
		values.Set(key, value)
	}
}
