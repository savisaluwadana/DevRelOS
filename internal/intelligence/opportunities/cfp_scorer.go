package opportunities

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/savisaluwadana/DevRelOS/internal/domain/events"
)

type CFPScoreBreakdown struct {
	TopicFit      int `json:"topicFit"`
	Readiness     int `json:"readiness"`
	Deadline      int `json:"deadline"`
	ExistingFit   int `json:"existingFit"`
	SubmissionGap int `json:"submissionGap"`
	Penalty       int `json:"penalty"`
}

type CFPOpportunity struct {
	CFP                 events.CFP            `json:"cfp"`
	Event               events.Event          `json:"event"`
	Talk                events.Talk           `json:"talk"`
	Score               int                   `json:"score"`
	Breakdown           CFPScoreBreakdown     `json:"breakdown"`
	Reasons             []string              `json:"reasons"`
	ExistingSubmission  *events.Submission    `json:"existingSubmission,omitempty"`
}

func RankCFPs(cfps []events.CFP, allEvents []events.Event, talks []events.Talk, submissions []events.Submission, now time.Time) []CFPOpportunity {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	eventByID := make(map[string]events.Event, len(allEvents))
	for _, event := range allEvents {
		eventByID[event.ID] = event
	}
	submissionByPair := make(map[string]*events.Submission, len(submissions))
	for i := range submissions {
		item := &submissions[i]
		submissionByPair[item.CFPID+":"+item.TalkID] = item
	}

	items := make([]CFPOpportunity, 0)
	for _, cfp := range cfps {
		if cfp.Status != "open" {
			continue
		}
		event := eventByID[cfp.EventID]
		for _, talk := range talks {
			if talk.Status == "retired" {
				continue
			}
			items = append(items, scoreCFPPair(cfp, event, talk, submissionByPair[cfp.ID+":"+talk.ID], now))
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score == items[j].Score {
			if items[i].Event.Name == items[j].Event.Name {
				return items[i].Talk.Title < items[j].Talk.Title
			}
			return items[i].Event.Name < items[j].Event.Name
		}
		return items[i].Score > items[j].Score
	})
	return items
}

func scoreCFPPair(cfp events.CFP, event events.Event, talk events.Talk, existing *events.Submission, now time.Time) CFPOpportunity {
	breakdown := CFPScoreBreakdown{}
	reasons := make([]string, 0, 6)

	targetTopics := append([]string{}, cfp.Tracks...)
	targetTopics = append(targetTopics, event.Topics...)
	overlap, talkCoverage, targetCoverage := topicOverlap(talk.Topics, targetTopics)
	if overlap > 0 {
		breakdown.TopicFit = clamp(int(math.Round((0.7*talkCoverage+0.3*targetCoverage)*55)), 0, 55)
		reasons = append(reasons, fmt.Sprintf("%d topic/track match(es) with the CFP", overlap))
	} else if len(targetTopics) > 0 && len(talk.Topics) > 0 {
		reasons = append(reasons, "no explicit talk-to-track overlap")
	}

	if talk.Status == "ready" {
		breakdown.Readiness += 8
		reasons = append(reasons, "talk is marked ready")
	}
	if strings.TrimSpace(talk.Abstract) != "" {
		breakdown.Readiness += 3
	}
	if strings.TrimSpace(talk.DemoURL) != "" {
		breakdown.Readiness += 2
	}
	if strings.TrimSpace(talk.SlidesURL) != "" || strings.TrimSpace(talk.RecordingURL) != "" {
		breakdown.Readiness += 2
	}
	breakdown.Readiness = clamp(breakdown.Readiness, 0, 15)

	if cfp.ClosesAt != nil && cfp.ClosesAt.After(now) {
		days := cfp.ClosesAt.Sub(now).Hours() / 24
		switch {
		case days <= 3:
			breakdown.Deadline = 4
			reasons = append(reasons, "CFP closes within three days")
		case days <= 14:
			breakdown.Deadline = 10
			reasons = append(reasons, "CFP closes within two weeks")
		case days <= 45:
			breakdown.Deadline = 7
		case days <= 90:
			breakdown.Deadline = 4
		}
	}

	if cfp.FitScore != nil {
		breakdown.ExistingFit = clamp(int(math.Round(float64(*cfp.FitScore)*0.15)), 0, 15)
		if *cfp.FitScore >= 70 {
			reasons = append(reasons, "the CFP already has a strong project-level fit score")
		}
	}

	if existing == nil {
		breakdown.SubmissionGap = 5
	} else {
		switch existing.Status {
		case "accepted":
			breakdown.Penalty = 60
			reasons = append(reasons, "this talk is already accepted for the CFP")
		case "submitted", "ready", "needs_work", "draft":
			breakdown.Penalty = 35
			reasons = append(reasons, "a submission for this talk is already in progress")
		case "rejected", "withdrawn":
			breakdown.Penalty = 15
		}
	}

	score := breakdown.TopicFit + breakdown.Readiness + breakdown.Deadline + breakdown.ExistingFit + breakdown.SubmissionGap - breakdown.Penalty
	score = clamp(score, 0, 100)
	return CFPOpportunity{CFP: cfp, Event: event, Talk: talk, Score: score, Breakdown: breakdown, Reasons: reasons, ExistingSubmission: existing}
}
