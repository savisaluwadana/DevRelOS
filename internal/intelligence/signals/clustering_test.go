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
			Body:   "Our platform team finds cluster setup confusing and manual.",
			Topics: []string{"kubernetes"}, RelevanceScore: &relevance, EngagementScore: 60,
			OccurredAt: ptrTime(now.Add(-24 * time.Hour)), Status: "new",
		},
		{
			ID: "b", Provider: "github", Title: "K8s configuration is hard",
			Body:   "Deploying and configuring the cluster takes too many steps.",
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
	// Two signals per friction mode: a cluster must clear the recurring
	// threshold before it is reported, so one signal per mode would correctly
	// yield nothing and this test would not exercise separation at all.
	items := []domain.Signal{
		{ID: "a", Title: "OpenTelemetry docs are unclear", Body: "missing docs for traces", Topics: []string{"observability"}, Status: "new", CreatedAt: now},
		{ID: "b", Title: "OpenTelemetry tracing docs are undocumented", Body: "no docs for the collector pipeline", Topics: []string{"observability"}, Status: "new", CreatedAt: now},
		{ID: "c", Title: "OpenTelemetry collector keeps failing", Body: "collector error causes broken traces", Topics: []string{"observability"}, Status: "new", CreatedAt: now},
		{ID: "d", Title: "OpenTelemetry collector crash loop", Body: "the collector fails and traces break", Topics: []string{"observability"}, Status: "new", CreatedAt: now},
	}
	clusters := ClusterSignals(items, now)
	if len(clusters) != 2 {
		for _, c := range clusters {
			t.Logf("  cluster %s (%d signals)", c.Key, len(c.SignalIDs))
		}
		t.Fatalf("expected documentation and reliability to separate into 2 clusters, got %d", len(clusters))
	}
}

func ptrTime(value time.Time) *time.Time { return &value }

func TestClusterSignalsRequiresFrictionEvidence(t *testing.T) {
	now := time.Now().UTC()
	// Exactly the content that used to manufacture high-severity pain points:
	// GitHub release tags and unrelated news headlines. detectFriction returned
	// a catch-all "recurring developer friction" rule for all of it.
	noise := []domain.Signal{
		{ID: "r1", Title: "v1.36.4", Topics: []string{"kubernetes"}, Status: "new", CreatedAt: now},
		{ID: "r2", Title: "Kubernetes v1.31.10", Topics: []string{"kubernetes"}, Status: "new", CreatedAt: now},
		{ID: "r3", Title: "v1.35.5", Topics: []string{"kubernetes"}, Status: "new", CreatedAt: now},
		{ID: "n1", Title: "The Helicopter with Radioactive Blades", Status: "new", CreatedAt: now},
		{ID: "n2", Title: "We built our house for LAN parties", Status: "new", CreatedAt: now},
		{ID: "n3", Title: "Mistral raises 3B", Status: "new", CreatedAt: now},
		{ID: "a1", Title: "CNCF Announces Karmada Graduation", Status: "new", CreatedAt: now},
	}
	if clusters := ClusterSignals(noise, now); len(clusters) != 0 {
		for _, c := range clusters {
			t.Logf("  unexpected cluster %q from %d signals", c.Title, len(c.SignalIDs))
		}
		t.Fatalf("content expressing no friction produced %d pain point(s), want 0", len(clusters))
	}
}

func TestClusterSignalsStillFindsRealFrictionAmongNoise(t *testing.T) {
	now := time.Now().UTC()
	items := []domain.Signal{
		// Noise that must not dilute or hide the real complaints.
		{ID: "r1", Title: "v1.36.4", Topics: []string{"kubernetes"}, Status: "new", CreatedAt: now},
		{ID: "n1", Title: "The Helicopter with Radioactive Blades", Status: "new", CreatedAt: now},
		// Two genuine complaints sharing a friction mode.
		{ID: "p1", Title: "Helm upgrade keeps failing", Body: "every helm upgrade breaks our cluster", Topics: []string{"kubernetes"}, Status: "new", CreatedAt: now},
		{ID: "p2", Title: "helm rollback is broken", Body: "the upgrade fails and rollback errors out", Topics: []string{"kubernetes"}, Status: "new", CreatedAt: now},
	}
	clusters := ClusterSignals(items, now)
	if len(clusters) != 1 {
		t.Fatalf("expected exactly the one real pain point, got %d", len(clusters))
	}
	if got := len(clusters[0].SignalIDs); got != 2 {
		t.Fatalf("cluster holds %d signals, want only the 2 real complaints", got)
	}
	for _, id := range clusters[0].SignalIDs {
		if id == "r1" || id == "n1" {
			t.Fatalf("noise signal %q was used as pain-point evidence", id)
		}
	}
}

func TestClusterSignalsHonoursMinEvidence(t *testing.T) {
	now := time.Now().UTC()
	// One genuine complaint: real friction, but not recurring.
	single := []domain.Signal{
		{ID: "s1", Title: "OpenTelemetry docs are unclear", Body: "missing docs for traces", Topics: []string{"observability"}, Status: "new", CreatedAt: now},
	}

	if clusters := ClusterSignals(single, now); len(clusters) != 0 {
		t.Fatalf("a single signal is not recurring, got %d cluster(s)", len(clusters))
	}

	// An operator who wants every signal surfaced can lower the threshold.
	rules := DefaultRuleset()
	rules.MinEvidence = 1
	clusters := ClusterSignalsWith(rules, single, now)
	if len(clusters) != 1 {
		t.Fatalf("minEvidence=1 should surface the single signal, got %d", len(clusters))
	}
	if rules.MinEvidence == DefaultMinEvidence {
		t.Fatal("test is not exercising a non-default threshold")
	}

	// A zero or negative value falls back rather than reporting everything.
	rules.MinEvidence = 0
	if got := ClusterSignalsWith(rules, single, now); len(got) != 0 {
		t.Fatalf("an unset threshold must fall back to the default, got %d cluster(s)", len(got))
	}
}

func TestClusterSignalsExcludesAnnouncementSources(t *testing.T) {
	now := time.Now().UTC()
	// Real CNCF and Kubernetes blog prose. Every one of these matches a friction
	// term ("configure", "setup", "complex", "manual") while complaining about
	// nothing, which is why 30 announcement posts once formed a pain point.
	announcements := []domain.Signal{
		{ID: "a1", SourceShape: "announcement", Topics: []string{"kubernetes"}, Status: "new", CreatedAt: now,
			Title: "Kubernetes access via an identity provider", Body: "Configure the public client and set up the manual flow."},
		{ID: "a2", SourceShape: "announcement", Topics: []string{"kubernetes"}, Status: "new", CreatedAt: now,
			Title: "CNCF Announces Karmada Graduation", Body: "Multi-cluster, multi-cloud orchestration for complex hybrid setup."},
		{ID: "a3", SourceShape: "announcement", Topics: []string{"kubernetes"}, Status: "new", CreatedAt: now,
			Title: "Kubernetes v1.37: DRA Updates", Body: "Rootless mode graduates; configure the manual driver setup."},
	}
	if clusters := ClusterSignalsWith(DefaultRuleset(), announcements, now); len(clusters) != 0 {
		for _, c := range clusters {
			t.Logf("  unexpected cluster %q from %d announcement(s)", c.Title, len(c.SignalIDs))
		}
		t.Fatalf("announcement sources produced %d pain point(s), want 0", len(clusters))
	}
}

func TestClusterSignalsUsesReportsAlongsideAnnouncements(t *testing.T) {
	now := time.Now().UTC()
	items := []domain.Signal{
		// Announcements that would otherwise dominate the cluster.
		{ID: "a1", SourceShape: "announcement", Topics: []string{"kubernetes"}, Status: "new", CreatedAt: now,
			Title: "Kubernetes v1.37 Sneak Peek", Body: "New setup flow, simpler to configure."},
		{ID: "a2", SourceShape: "announcement", Topics: []string{"kubernetes"}, Status: "new", CreatedAt: now,
			Title: "CNCF welcomes new members", Body: "Enterprises scale complex hybrid infrastructure."},
		// Actual developers reporting the same friction.
		{ID: "p1", SourceShape: "report", Topics: []string{"kubernetes"}, Status: "new", CreatedAt: now,
			Title: "Helm upgrade keeps failing", Body: "every helm upgrade breaks our cluster"},
		{ID: "p2", SourceShape: "report", Topics: []string{"kubernetes"}, Status: "new", CreatedAt: now,
			Title: "helm rollback is broken", Body: "the upgrade fails and rollback errors out"},
	}
	clusters := ClusterSignalsWith(DefaultRuleset(), items, now)
	if len(clusters) != 1 {
		t.Fatalf("expected one pain point from the two reports, got %d", len(clusters))
	}
	if got := len(clusters[0].SignalIDs); got != 2 {
		t.Fatalf("cluster holds %d signals, want only the 2 reports", got)
	}
	for _, id := range clusters[0].SignalIDs {
		if id == "a1" || id == "a2" {
			t.Fatalf("announcement %q was counted as developer pain", id)
		}
	}
}

func TestClusterSignalsTreatsUnknownShapeAsEligible(t *testing.T) {
	now := time.Now().UTC()
	// Manually entered signals and rows predating the column carry no shape,
	// and must keep behaving as before rather than silently disappearing.
	items := []domain.Signal{
		{ID: "u1", Topics: []string{"kubernetes"}, Status: "new", CreatedAt: now,
			Title: "Helm upgrade keeps failing", Body: "every upgrade breaks the cluster"},
		{ID: "u2", SourceShape: "unknown", Topics: []string{"kubernetes"}, Status: "new", CreatedAt: now,
			Title: "helm rollback broken", Body: "the upgrade fails and rollback errors"},
	}
	if clusters := ClusterSignalsWith(DefaultRuleset(), items, now); len(clusters) != 1 {
		t.Fatalf("unshaped signals should still cluster, got %d cluster(s)", len(clusters))
	}
}
