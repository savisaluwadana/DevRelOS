package workitems

import (
	"fmt"
	"strings"

	signaldomain "github.com/savisaluwadana/DevRelOS/internal/domain/signals"
	workdomain "github.com/savisaluwadana/DevRelOS/internal/domain/workitems"
)

var allowedKinds = map[string]bool{
	"content_brief":      true,
	"docs_improvement":   true,
	"product_feedback":   true,
	"talk_idea":          true,
	"community_research": true,
	"outreach_follow_up": true,
	"event_task":         true,
	"engineering_task":   true,
}

func ValidKind(kind string) bool {
	return allowedKinds[kind]
}

func FromPainPoint(p signaldomain.PainPoint, kind, owner string) (workdomain.WorkItem, error) {
	if !ValidKind(kind) {
		return workdomain.WorkItem{}, fmt.Errorf("unsupported work item kind %q", kind)
	}
	prefix, action := kindCopy(kind)
	title := p.Title
	if strings.TrimSpace(title) == "" {
		title = p.Key
	}
	if strings.TrimSpace(title) == "" {
		title = "Developer pain point"
	}

	description := strings.TrimSpace(p.Summary)
	if description != "" {
		description = action + "\n\nEvidence-backed context: " + description
	} else {
		description = action
	}

	priority := p.Severity
	if p.TrendScore > 0 {
		priority += p.TrendScore / 4
	}
	if p.EvidenceCount >= 10 {
		priority += 5
	}
	if priority < 20 {
		priority = 20
	}
	if priority > 100 {
		priority = 100
	}

	return workdomain.WorkItem{
		ProjectID:   p.ProjectID,
		SourceType:  "pain_point",
		SourceID:    p.ID,
		Kind:        kind,
		Title:       prefix + title,
		Description: description,
		Priority:    priority,
		Status:      "backlog",
		Owner:       owner,
		Metadata: map[string]any{
			"pain_point_key":   p.Key,
			"pain_point_title": p.Title,
			"persona":          p.Persona,
			"severity":         p.Severity,
			"trend_score":      p.TrendScore,
			"evidence_count":   p.EvidenceCount,
			"topics":           p.Topics,
		},
	}, nil
}

func kindCopy(kind string) (string, string) {
	switch kind {
	case "content_brief":
		return "Content: ", "Create an evidence-backed technical content brief that directly addresses this developer problem."
	case "docs_improvement":
		return "Docs: ", "Identify the documentation gap behind this pain point and define a concrete documentation improvement."
	case "product_feedback":
		return "Feedback: ", "Turn this recurring developer pain into structured product feedback with evidence and expected developer impact."
	case "talk_idea":
		return "Talk: ", "Develop a talk or workshop angle that teaches developers how to solve or understand this recurring problem."
	case "community_research":
		return "Community: ", "Find relevant developer communities where this pain point is discussed and identify useful engagement opportunities."
	case "outreach_follow_up":
		return "Follow-up: ", "Plan a useful relationship follow-up connected to this developer pain point; do not send automatically."
	case "event_task":
		return "Event: ", "Create an event-related action that uses this pain point to improve a CFP, session, demo or event interaction."
	case "engineering_task":
		return "Engineering: ", "Define the smallest technical investigation or fix needed to address this evidence-backed developer problem."
	default:
		return "Action: ", "Create a concrete DevRel action for this pain point."
	}
}
