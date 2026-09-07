package feedback

import (
	"fmt"
	"strings"

	feedbackdomain "github.com/savisaluwadana/DevRelOS/internal/domain/feedback"
	signaldomain "github.com/savisaluwadana/DevRelOS/internal/domain/signals"
)

func FromPainPoint(p signaldomain.PainPoint, component, owner, githubRepository string) feedbackdomain.Item {
	component = strings.TrimSpace(component)
	if component == "" && len(p.Topics) > 0 {
		component = p.Topics[0]
	}

	impact := clampScore(int(float64(p.Severity)*0.60 + float64(p.TrendScore)*0.25 + float64(minInt(p.EvidenceCount*3, 15))))
	frequency := clampScore(p.EvidenceCount*8 + p.TrendScore/3)
	if p.EvidenceCount == 0 {
		frequency = clampScore(p.TrendScore / 2)
	}

	metadata := map[string]any{
		"pain_point_key": p.Key,
		"topics":         p.Topics,
		"severity":       p.Severity,
		"trendScore":     p.TrendScore,
		"evidenceCount":  p.EvidenceCount,
	}
	if p.FirstSeenAt != nil {
		metadata["firstSeenAt"] = p.FirstSeenAt
	}
	if p.LastSeenAt != nil {
		metadata["lastSeenAt"] = p.LastSeenAt
	}

	return feedbackdomain.Item{
		ProjectID:        p.ProjectID,
		SourceType:       "pain_point",
		SourceID:         p.ID,
		Title:            p.Title,
		Summary:          p.Summary,
		Persona:          p.Persona,
		Component:        component,
		ImpactScore:      impact,
		FrequencyScore:   frequency,
		Status:           "new",
		Owner:            strings.TrimSpace(owner),
		GitHubRepository: strings.TrimSpace(githubRepository),
		GitHubIssueTitle: issueTitle(p),
		GitHubIssueBody:  issueBody(p, impact, frequency, component),
		Metadata:         metadata,
	}
}

func issueTitle(p signaldomain.PainPoint) string {
	title := strings.TrimSpace(p.Title)
	if title == "" {
		title = "Developer-reported friction"
	}
	return "Developer feedback: " + title
}

func issueBody(p signaldomain.PainPoint, impact, frequency int, component string) string {
	var b strings.Builder
	b.WriteString("## Developer problem\n")
	if strings.TrimSpace(p.Summary) != "" {
		b.WriteString(strings.TrimSpace(p.Summary))
	} else {
		b.WriteString(strings.TrimSpace(p.Title))
	}
	b.WriteString("\n\n## Evidence\n")
	b.WriteString(fmt.Sprintf("- Pain point ID: `%s`\n", p.ID))
	b.WriteString(fmt.Sprintf("- Evidence records: %d\n", p.EvidenceCount))
	b.WriteString(fmt.Sprintf("- Severity: %d/100\n", p.Severity))
	b.WriteString(fmt.Sprintf("- Trend: %d/100\n", p.TrendScore))
	b.WriteString(fmt.Sprintf("- Calculated impact: %d/100\n", impact))
	b.WriteString(fmt.Sprintf("- Calculated frequency: %d/100\n", frequency))
	if strings.TrimSpace(p.Persona) != "" {
		b.WriteString("- Persona: " + strings.TrimSpace(p.Persona) + "\n")
	}
	if component != "" {
		b.WriteString("- Component / area: " + component + "\n")
	}
	if len(p.Topics) > 0 {
		b.WriteString("- Topics: " + strings.Join(p.Topics, ", ") + "\n")
	}
	b.WriteString("\n## Acceptance / investigation notes\n")
	b.WriteString("- Validate the underlying developer workflow against the linked evidence before implementation.\n")
	b.WriteString("- Record the chosen resolution, trade-offs, and verification path.\n")
	b.WriteString("- When shipped, link the release/docs change back to DevRelOS for developer follow-up.\n")
	return b.String()
}

func clampScore(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
