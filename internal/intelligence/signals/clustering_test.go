package signals

import (
	"testing"
	"time"

	domain "github.com/savisaluwadana/DevRelOS/internal/domain/signals"
)

func TestClusterSignalsGroupsRelatedFriction(t *testing.T) {
	now := time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC)
	relevance := 90
	items := []domain.Signal{
		{
			ID: "a", Provider: "reddit", Title: "Kubernetes setup is too complex",
			Body: "Our platform team finds cluster setup confusing and manual.",
			Topics: []string{"kubernetes"}, RelevanceScore: &relevance, EngagementScore: 60,
			OccurredAt: ptrTime(now.Add(-24 * time.Hour)), Status: "new",
		},
		{
			ID: "b", Provider: "github", Title: "K8s configuration is hard",
			Body: "Deploying and configuring the cluster takes too many steps.",
			Topics: []string{"kubernetes"}, RelevanceScore: &relevance, EngagementScore: 40,
			OccurredAt: ptrTime(now.Add(-48 * time.Hour)), Status: "reviewed",
		},
	}

	clusters := ClusterSignals(items, now)
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}
	cluster := clusters[0]
	if cluster.Key != "kubernetes:complexity" {
		t.Fatalf("unexpected key: %s", cluster.Key)
	}
	if len(cluster.SignalIDs) != 2 {
		t.Fatalf("expected 2 evidence signals, got %d", len(cluster.SignalIDs))
	}
	if cluster.Persona != "Platform engineer" {
		t.Fatalf("unexpected persona: %s", cluster.Persona)
	}
	if cluster.TrendScore <= 0 {
		t.Fatalf("expected positive trend score, got %d", cluster.TrendScore)
	}
}

func TestClusterSignalsIgnoresIgnoredSignals(t *testing.T) {
	now := time.Now().UTC()
	items := []domain.Signal{{ID: "ignored", Title: "Docs are confusing", Body: "No docs", Status: "ignored"}}
	if clusters := ClusterSignals(items, now); len(clusters) != 0 {
		t.Fatalf("expected ignored signal to be excluded, got %d clusters", len(clusters))
	}
}

func TestClusterSignalsSeparatesDifferentFrictionModes(t *testing.T) {
	now := time.Now().UTC()
	items := []domain.Signal{
		{ID: "a", Title: "OpenTelemetry docs are unclear", Body: "missing docs for traces", Topics: []string{"observability"}, Status: "new", CreatedAt: now},
		{ID: "b", Title: "OpenTelemetry collector keeps failing", Body: "collector error causes broken traces", Topics: []string{"observability"}, Status: "new", CreatedAt: now},
	}
	clusters := ClusterSignals(items, now)
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}
}

func ptrTime(value time.Time) *time.Time { return &value }
