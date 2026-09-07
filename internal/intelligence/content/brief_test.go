package content

import (
	"strings"
	"testing"

	workdomain "github.com/savisaluwadana/DevRelOS/internal/domain/workitems"
)

func TestFromWorkItemBuildsTraceableBrief(t *testing.T) {
	item := workdomain.WorkItem{
		ID:          "work-123",
		ProjectID:   "project-123",
		SourceType:  "pain_point",
		SourceID:    "pain-123",
		Kind:        "content_brief",
		Title:       "Content brief: Kubernetes deployment feedback",
		Description: "Developers repeatedly report difficulty understanding deployment failures.",
		Priority:    82,
		Status:      "planned",
		Metadata: map[string]any{
			"persona":       "platform engineers",
			"topics":        []any{"Kubernetes", "Developer Experience"},
			"evidenceCount": float64(14),
			"severity":      float64(78),
			"trendScore":    float64(64),
		},
	}

	asset, err := FromWorkItem(item, "blog", "article", "")
	if err != nil {
		t.Fatalf("FromWorkItem returned error: %v", err)
	}
	if asset.WorkItemID != item.ID || asset.ProjectID != item.ProjectID {
		t.Fatalf("provenance not preserved: %#v", asset)
	}
	if asset.Audience != "platform engineers" {
		t.Fatalf("expected persona-derived audience, got %q", asset.Audience)
	}
	if asset.Status != "brief" {
		t.Fatalf("expected brief status, got %q", asset.Status)
	}
	if len(asset.Topics) != 2 || asset.Topics[0] != "kubernetes" || asset.Topics[1] != "developer-experience" {
		t.Fatalf("unexpected normalized topics: %#v", asset.Topics)
	}
	for _, expected := range []string{"Evidence records: 14", "Pain severity: 78/100", "Verify every technical claim"} {
		if !strings.Contains(asset.Brief, expected) {
			t.Fatalf("brief missing %q: %s", expected, asset.Brief)
		}
	}
}

func TestFromWorkItemRejectsInvalidChannelFormat(t *testing.T) {
	_, err := FromWorkItem(workdomain.WorkItem{ID: "work-1", ProjectID: "project-1", Title: "Test"}, "linkedin", "article", "developers")
	if err == nil {
		t.Fatal("expected invalid channel/format combination to fail")
	}
}

func TestValidChannelFormat(t *testing.T) {
	cases := []struct {
		channel string
		format  string
		want    bool
	}{
		{"blog", "article", true},
		{"x", "thread", true},
		{"docs", "documentation", true},
		{"short_video", "short_script", true},
		{"newsletter", "video_script", false},
		{"unknown", "article", false},
	}
	for _, tc := range cases {
		if got := ValidChannelFormat(tc.channel, tc.format); got != tc.want {
			t.Errorf("ValidChannelFormat(%q, %q)=%v want %v", tc.channel, tc.format, got, tc.want)
		}
	}
}
