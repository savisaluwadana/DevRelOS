package opportunities

import (
	"testing"
	"time"

	"github.com/savisaluwadana/DevRelOS/internal/domain/events"
	outreachdomain "github.com/savisaluwadana/DevRelOS/internal/domain/outreach"
)

func TestRankSpeakingPrefersTopicFitAndWarmRelationship(t *testing.T) {
	now := time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC)
	activity := 80
	members := 900
	nextEvent := now.Add(20 * 24 * time.Hour)

	communities := []events.Community{
		{
			ID:            "community-a",
			Name:          "Cloud Native A",
			Topics:        []string{"kubernetes", "platform-engineering"},
			ActivityScore: &activity,
			MemberCount:   &members,
			NextEventAt:   &nextEvent,
			Status:        "warm",
		},
		{
			ID:     "community-b",
			Name:   "Cloud Native B",
			Topics: []string{"observability"},
			Status: "discovered",
		},
	}
	talks := []events.Talk{
		{ID: "talk-1", Title: "Platform engineering with Kubernetes", Topics: []string{"kubernetes", "platform-engineering"}, Status: "ready", DemoURL: "https://example.com/demo"},
		{ID: "talk-2", Title: "Tracing systems", Topics: []string{"observability"}, Status: "draft"},
	}
	relationships := []outreachdomain.Relationship{{CommunityID: "community-a", Stage: "engaged", Strength: 75}}

	got := RankSpeaking(communities, talks, relationships, nil, now)
	if len(got) != 4 {
		t.Fatalf("expected 4 pairings, got %d", len(got))
	}
	if got[0].Community.ID != "community-a" || got[0].Talk.ID != "talk-1" {
		t.Fatalf("expected community-a/talk-1 to rank first, got %s/%s", got[0].Community.ID, got[0].Talk.ID)
	}
	if got[0].Breakdown.TopicFit <= 0 || got[0].Breakdown.Relationship <= 0 {
		t.Fatalf("expected topic and relationship contributions, got %+v", got[0].Breakdown)
	}
}

func TestRankSpeakingPenalizesExistingOutreach(t *testing.T) {
	now := time.Now().UTC()
	communities := []events.Community{{ID: "community-a", Name: "A", Topics: []string{"kubernetes"}, Status: "discovered"}}
	talks := []events.Talk{{ID: "talk-1", Title: "Kubernetes", Topics: []string{"kubernetes"}, Status: "ready"}}
	without := RankSpeaking(communities, talks, nil, nil, now)
	with := RankSpeaking(communities, talks, nil, []outreachdomain.Outreach{{CommunityID: "community-a", Status: "sent", UpdatedAt: now}}, now)

	if len(without) != 1 || len(with) != 1 {
		t.Fatal("expected one result in both rankings")
	}
	if with[0].Score >= without[0].Score {
		t.Fatalf("expected sent outreach to lower score: without=%d with=%d", without[0].Score, with[0].Score)
	}
	if with[0].ExistingOutreachState != "sent" {
		t.Fatalf("expected sent state, got %q", with[0].ExistingOutreachState)
	}
}

func TestRankSpeakingSkipsPausedAndRetired(t *testing.T) {
	communities := []events.Community{
		{ID: "paused", Status: "paused"},
		{ID: "active", Status: "discovered"},
	}
	talks := []events.Talk{
		{ID: "retired", Status: "retired"},
		{ID: "ready", Status: "ready"},
	}
	got := RankSpeaking(communities, talks, nil, nil, time.Now().UTC())
	if len(got) != 1 || got[0].Community.ID != "active" || got[0].Talk.ID != "ready" {
		t.Fatalf("unexpected results: %+v", got)
	}
}
