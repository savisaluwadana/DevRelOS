package feedback

import (
	"strings"
	"testing"

	signaldomain "github.com/savisaluwadana/DevRelOS/internal/domain/signals"
)

func TestFromPainPointBuildsTriageableFeedback(t *testing.T) {
	p := signaldomain.PainPoint{
		ID:            "pain-1",
		ProjectID:     "project-1",
		Key:           "deployment-errors",
		Title:         "Deployment errors are hard to diagnose",
		Summary:       "Developers repeatedly cannot tell why deployments failed.",
		Persona:       "platform engineers",
		Severity:      82,
		TrendScore:    70,
		EvidenceCount: 12,
		Topics:        []string{"kubernetes", "observability"},
	}

	item := FromPainPoint(p, "", "platform-team", "example/project")
	if item.SourceType != "pain_point" || item.SourceID != p.ID {
		t.Fatalf("feedback provenance was not preserved: %#v", item)
	}
	if item.Component != "kubernetes" {
		t.Fatalf("expected first topic as fallback component, got %q", item.Component)
	}
	if item.ImpactScore <= 0 || item.ImpactScore > 100 || item.FrequencyScore <= 0 || item.FrequencyScore > 100 {
		t.Fatalf("scores out of range: impact=%d frequency=%d", item.ImpactScore, item.FrequencyScore)
	}
	if !strings.Contains(item.GitHubIssueBody, "Evidence records: 12") || !strings.Contains(item.GitHubIssueBody, "Pain point ID") {
		t.Fatalf("GitHub issue draft is missing evidence context: %s", item.GitHubIssueBody)
	}
}

func TestFeedbackScoresClamp(t *testing.T) {
	p := signaldomain.PainPoint{ID: "pain-2", ProjectID: "project-1", Severity: 100, TrendScore: 100, EvidenceCount: 1000}
	item := FromPainPoint(p, "api", "", "")
	if item.ImpactScore != 100 || item.FrequencyScore != 100 {
		t.Fatalf("expected clamped scores, got impact=%d frequency=%d", item.ImpactScore, item.FrequencyScore)
	}
}
