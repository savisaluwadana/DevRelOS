package opportunities

import (
	"testing"
	"time"

	"github.com/savisaluwadana/DevRelOS/internal/domain/events"
)

func TestRankCFPsPrefersReadyMatchingTalk(t *testing.T) {
	now := time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC)
	closes := now.Add(10 * 24 * time.Hour)
	fit := 80
	cfps := []events.CFP{{ID: "cfp-1", EventID: "event-1", Status: "open", Tracks: []string{"platform-engineering", "kubernetes"}, ClosesAt: &closes, FitScore: &fit}}
	allEvents := []events.Event{{ID: "event-1", Name: "KCD Test", Topics: []string{"cloud-native", "kubernetes"}}}
	talks := []events.Talk{
		{ID: "talk-good", Title: "Platform Engineering", Topics: []string{"platform-engineering", "kubernetes"}, Status: "ready", Abstract: "abstract", DemoURL: "https://example.com"},
		{ID: "talk-bad", Title: "Unrelated", Topics: []string{"database"}, Status: "draft"},
	}

	got := RankCFPs(cfps, allEvents, talks, nil, now)
	if len(got) != 2 {
		t.Fatalf("expected 2 pairings, got %d", len(got))
	}
	if got[0].Talk.ID != "talk-good" {
		t.Fatalf("expected matching talk first, got %s", got[0].Talk.ID)
	}
	if got[0].Breakdown.TopicFit <= 0 || got[0].Breakdown.Readiness <= 0 || got[0].Breakdown.Deadline <= 0 {
		t.Fatalf("expected positive fit/readiness/deadline scores, got %+v", got[0].Breakdown)
	}
}

func TestRankCFPsPenalizesExistingSubmission(t *testing.T) {
	now := time.Now().UTC()
	cfps := []events.CFP{{ID: "cfp-1", EventID: "event-1", Status: "open", Tracks: []string{"kubernetes"}}}
	allEvents := []events.Event{{ID: "event-1", Name: "Event", Topics: []string{"kubernetes"}}}
	talks := []events.Talk{{ID: "talk-1", Title: "Kubernetes", Topics: []string{"kubernetes"}, Status: "ready"}}

	without := RankCFPs(cfps, allEvents, talks, nil, now)
	with := RankCFPs(cfps, allEvents, talks, []events.Submission{{CFPID: "cfp-1", TalkID: "talk-1", Status: "submitted"}}, now)
	if with[0].Score >= without[0].Score {
		t.Fatalf("expected existing submission penalty, without=%d with=%d", without[0].Score, with[0].Score)
	}
	if with[0].ExistingSubmission == nil {
		t.Fatal("expected existing submission in result")
	}
}

func TestRankCFPsSkipsClosedAndRetired(t *testing.T) {
	cfps := []events.CFP{{ID: "closed", EventID: "event-1", Status: "closed"}, {ID: "open", EventID: "event-1", Status: "open"}}
	allEvents := []events.Event{{ID: "event-1", Name: "Event"}}
	talks := []events.Talk{{ID: "retired", Status: "retired"}, {ID: "ready", Status: "ready"}}
	got := RankCFPs(cfps, allEvents, talks, nil, time.Now().UTC())
	if len(got) != 1 || got[0].CFP.ID != "open" || got[0].Talk.ID != "ready" {
		t.Fatalf("unexpected results: %+v", got)
	}
}
