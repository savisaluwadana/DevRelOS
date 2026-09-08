package opportunities

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/savisaluwadana/DevRelOS/internal/domain/events"
	outreachdomain "github.com/savisaluwadana/DevRelOS/internal/domain/outreach"
)

type ScoreBreakdown struct {
	TopicFit         int `json:"topicFit"`
	CommunityQuality int `json:"communityQuality"`
	Relationship     int `json:"relationship"`
	TalkReadiness    int `json:"talkReadiness"`
	Timing           int `json:"timing"`
	Penalty          int `json:"penalty"`
}

type SpeakingOpportunity struct {
	Community             events.Community             `json:"community"`
	Talk                  events.Talk                  `json:"talk"`
	Score                 int                          `json:"score"`
	Breakdown             ScoreBreakdown               `json:"breakdown"`
	Reasons               []string                     `json:"reasons"`
	Relationship          *outreachdomain.Relationship `json:"relationship,omitempty"`
	ExistingOutreachState string                       `json:"existingOutreachState,omitempty"`
}

// Pair is one (community, talk) candidate to score. Pair generation lives in
// the storage layer so the database can filter and bound the candidate set
// instead of the whole cross product being built in memory.
type Pair struct {
	Community events.Community
	Talk      events.Talk
}

// RankSpeaking builds the full community x talk cross product itself.
//
// It is retained for callers that already hold complete slices and for the
// scoring tests. Request handlers should use RankSpeakingPairs with candidates
// from storage.ListSpeakingCandidates: this function is quadratic in the number
// of communities and talks.
func RankSpeaking(communities []events.Community, talks []events.Talk, relationships []outreachdomain.Relationship, outreachItems []outreachdomain.Outreach, now time.Time) []SpeakingOpportunity {
	pairs := make([]Pair, 0, len(communities)*len(talks))
	for _, community := range communities {
		if community.Status == "do_not_contact" || community.Status == "paused" {
			continue
		}
		for _, talk := range talks {
			if talk.Status == "retired" {
				continue
			}
			pairs = append(pairs, Pair{Community: community, Talk: talk})
		}
	}
	return RankSpeakingPairs(pairs, relationships, outreachItems, now)
}

// RankSpeakingPairs scores an already-selected candidate set and orders it by
// exact score. Eligibility filtering is the caller's job, because the candidate
// query applies the same rules in SQL.
func RankSpeakingPairs(pairs []Pair, relationships []outreachdomain.Relationship, outreachItems []outreachdomain.Outreach, now time.Time) []SpeakingOpportunity {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	relByCommunity := bestRelationshipByCommunity(relationships)
	outreachByCommunity := latestOutreachByCommunity(outreachItems)
	results := make([]SpeakingOpportunity, 0, len(pairs))

	for _, pair := range pairs {
		results = append(results, scorePair(
			pair.Community, pair.Talk,
			relByCommunity[pair.Community.ID],
			outreachByCommunity[pair.Community.ID],
			now,
		))
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			if results[i].Community.Name == results[j].Community.Name {
				return results[i].Talk.Title < results[j].Talk.Title
			}
			return results[i].Community.Name < results[j].Community.Name
		}
		return results[i].Score > results[j].Score
	})
	return results
}

func scorePair(community events.Community, talk events.Talk, relationship *outreachdomain.Relationship, existing *outreachdomain.Outreach, now time.Time) SpeakingOpportunity {
	breakdown := ScoreBreakdown{}
	reasons := make([]string, 0, 6)

	overlap, talkCoverage, communityCoverage := topicOverlap(talk.Topics, community.Topics)
	if overlap > 0 {
		breakdown.TopicFit = clamp(int(math.Round((0.65*talkCoverage+0.35*communityCoverage)*45)), 0, 45)
		reasons = append(reasons, fmt.Sprintf("%d shared topic(s) between the talk and community", overlap))
	} else if len(talk.Topics) > 0 && len(community.Topics) > 0 {
		reasons = append(reasons, "no explicit topic overlap yet")
	}

	quality := 0
	if community.ActivityScore != nil {
		quality += int(math.Round(float64(*community.ActivityScore) * 0.12))
		if *community.ActivityScore >= 70 {
			reasons = append(reasons, "community activity is strong")
		}
	}
	if community.MemberCount != nil && *community.MemberCount > 0 {
		memberBonus := int(math.Min(6, math.Log10(float64(*community.MemberCount)+1)*2))
		quality += memberBonus
	}
	if community.SpeakingFitScore != nil {
		quality += int(math.Round(float64(*community.SpeakingFitScore) * 0.07))
	}
	breakdown.CommunityQuality = clamp(quality, 0, 25)

	if relationship != nil {
		relScore := int(math.Round(float64(relationship.Strength) * 0.12))
		switch relationship.Stage {
		case "partner":
			relScore += 8
		case "engaged":
			relScore += 6
		case "warm":
			relScore += 3
		}
		breakdown.Relationship = clamp(relScore, 0, 20)
		if relationship.Stage != "cold" {
			reasons = append(reasons, fmt.Sprintf("relationship is %s", relationship.Stage))
		}
	}

	readiness := 0
	if talk.Status == "ready" {
		readiness += 5
		reasons = append(reasons, "talk is marked ready")
	}
	if strings.TrimSpace(talk.DemoURL) != "" {
		readiness += 2
	}
	if strings.TrimSpace(talk.SlidesURL) != "" {
		readiness += 2
	}
	if strings.TrimSpace(talk.RecordingURL) != "" {
		readiness += 1
	}
	breakdown.TalkReadiness = clamp(readiness, 0, 10)

	if community.NextEventAt != nil && community.NextEventAt.After(now) {
		days := community.NextEventAt.Sub(now).Hours() / 24
		switch {
		case days <= 14:
			breakdown.Timing = 6
			reasons = append(reasons, "an upcoming event is within two weeks")
		case days <= 45:
			breakdown.Timing = 4
			reasons = append(reasons, "an upcoming event is within 45 days")
		case days <= 90:
			breakdown.Timing = 2
		}
	}

	if existing != nil {
		switch existing.Status {
		case "needs_approval", "approved", "queued":
			breakdown.Penalty = 25
			reasons = append(reasons, "outreach is already in progress")
		case "sent":
			breakdown.Penalty = 35
			reasons = append(reasons, "outreach was already sent")
		case "replied":
			breakdown.Penalty = 15
			reasons = append(reasons, "the community has already replied")
		}
	}

	score := breakdown.TopicFit + breakdown.CommunityQuality + breakdown.Relationship + breakdown.TalkReadiness + breakdown.Timing - breakdown.Penalty
	score = clamp(score, 0, 100)

	state := ""
	if existing != nil {
		state = existing.Status
	}
	return SpeakingOpportunity{
		Community:             community,
		Talk:                  talk,
		Score:                 score,
		Breakdown:             breakdown,
		Reasons:               reasons,
		Relationship:          relationship,
		ExistingOutreachState: state,
	}
}

func topicOverlap(talkTopics, communityTopics []string) (int, float64, float64) {
	talkSet := topicSet(talkTopics)
	communitySet := topicSet(communityTopics)
	if len(talkSet) == 0 || len(communitySet) == 0 {
		return 0, 0, 0
	}
	matches := 0
	for topic := range talkSet {
		if _, ok := communitySet[topic]; ok {
			matches++
		}
	}
	return matches, float64(matches) / float64(len(talkSet)), float64(matches) / float64(len(communitySet))
}

func topicSet(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		normalized := strings.ToLower(strings.TrimSpace(item))
		normalized = strings.ReplaceAll(normalized, "_", "-")
		normalized = strings.ReplaceAll(normalized, " ", "-")
		if normalized != "" {
			out[normalized] = struct{}{}
		}
	}
	return out
}

func bestRelationshipByCommunity(items []outreachdomain.Relationship) map[string]*outreachdomain.Relationship {
	out := make(map[string]*outreachdomain.Relationship)
	for i := range items {
		item := &items[i]
		if item.CommunityID == "" {
			continue
		}
		current, ok := out[item.CommunityID]
		if !ok || item.Strength > current.Strength {
			out[item.CommunityID] = item
		}
	}
	return out
}

func latestOutreachByCommunity(items []outreachdomain.Outreach) map[string]*outreachdomain.Outreach {
	out := make(map[string]*outreachdomain.Outreach)
	for i := range items {
		item := &items[i]
		if item.CommunityID == "" || item.Status == "cancelled" || item.Status == "failed" {
			continue
		}
		current, ok := out[item.CommunityID]
		if !ok || item.UpdatedAt.After(current.UpdatedAt) {
			out[item.CommunityID] = item
		}
	}
	return out
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
