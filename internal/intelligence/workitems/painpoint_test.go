package workitems

import (
	"testing"

	signaldomain "github.com/savisaluwadana/DevRelOS/internal/domain/signals"
)

func TestFromPainPointBuildsTraceablePriorityWork(t *testing.T) {
	pain := signaldomain.PainPoint{
		ID:            "pain-1",
		ProjectID:     "project-1",
		Key:           "kubernetes-docs",
		Title:         "Kubernetes deployment docs are confusing",
		Summary:       "Developers repeatedly fail to find the right deployment path.",
		Persona:       "Platform engineer",
		Severity:      72,
		TrendScore:    24,
		EvidenceCount: 12,
		Topics:        []string{"kubernetes", "docs"},
	}

	item, err := FromPainPoint(pain, "docs_improvement", "savi")
	if err != nil {
		t.Fatal(err)
	}
	if item.SourceType != "pain_point" || item.SourceID != pain.ID {
		t.Fatalf("expected traceable pain point source, got %s/%s", item.SourceType, item.SourceID)
	}
	if item.Kind != "docs_improvement" || item.Status != "backlog" {
		t.Fatalf("unexpected kind/status: %s/%s", item.Kind, item.Status)
	}
	if item.Priority <= pain.Severity || item.Priority > 100 {
		t.Fatalf("expected bounded priority boosted by trend/evidence, got %d", item.Priority)
	}
	if item.Metadata["evidence_count"] != 12 {
		t.Fatalf("expected evidence metadata, got %+v", item.Metadata)
	}
}

func TestFromPainPointRejectsUnknownKind(t *testing.T) {
	_, err := FromPainPoint(signaldomain.PainPoint{}, "random_thing", "")
	if err == nil {
		t.Fatal("expected unsupported kind error")
	}
}
