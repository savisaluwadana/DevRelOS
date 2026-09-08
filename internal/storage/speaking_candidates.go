package storage

import (
	"context"

	"github.com/savisaluwadana/DevRelOS/internal/domain/events"
)

// SpeakingCandidate is one (community, talk) pair worth scoring.
type SpeakingCandidate struct {
	Community events.Community
	Talk      events.Talk
}

// Recall budget for candidate selection. The exact score is computed in Go, so
// SQL cannot order by it; instead SQL orders by a cheap proxy that tracks the
// two largest score components (topic fit, then community quality) and returns
// the best slice for exact ranking.
const (
	minSpeakingCandidates = 500
	maxSpeakingCandidates = 5000
)

// SpeakingCandidateBudget converts a requested result count into a recall
// budget, wide enough that the exact top-N is stable in practice.
func SpeakingCandidateBudget(limit int) int {
	budget := limit * 20
	if budget < minSpeakingCandidates {
		budget = minSpeakingCandidates
	}
	if budget > maxSpeakingCandidates {
		budget = maxSpeakingCandidates
	}
	return budget
}

// ListSpeakingCandidates selects (community, talk) pairs worth ranking.
//
// Ranking used to read every community, talk, relationship and outreach row and
// build the whole community x talk cross product in memory on every request,
// only to return the top 50. This moves pair generation into SQL, keeps two
// filters that make a pair meaningful at all, and bounds the work:
//
//   - the community and talk are both eligible (not paused, do-not-contact or
//     retired), which the Go scorer used to check after materialising the pair;
//   - their topics overlap, or a relationship with that community already
//     exists. A pair with neither can only score on community activity, talk
//     readiness and timing - "active community, ready talk, unrelated subject,
//     nobody you know" - which is not an actionable speaking recommendation.
//
// Both are deliberate narrowings of the old result set, and both only remove
// rows that sat at the bottom of the ranking.
func (s *Store) ListSpeakingCandidates(ctx context.Context, projectID string, budget int) ([]SpeakingCandidate, error) {
	if budget <= 0 {
		budget = minSpeakingCandidates
	}
	if budget > maxSpeakingCandidates {
		budget = maxSpeakingCandidates
	}

	rows, err := s.pool.Query(ctx, `
		SELECT c.id::text, c.project_id::text, c.name, c.platform, COALESCE(c.external_id,''),
		       COALESCE(c.website_url,''), COALESCE(c.city,''), COALESCE(c.country,''),
		       COALESCE(c.timezone,''), c.topics, c.member_count, c.activity_score,
		       c.speaking_fit_score, c.last_event_at, c.next_event_at, c.status,
		       c.created_at, c.updated_at,
		       t.id::text, t.project_id::text, t.title, t.abstract, t.description, t.level,
		       t.duration_minutes, t.topics, COALESCE(t.demo_url,''), COALESCE(t.slides_url,''),
		       COALESCE(t.recording_url,''), t.status, t.created_at, t.updated_at
		FROM communities c
		JOIN talks t ON t.project_id = c.project_id
		LEFT JOIN LATERAL (
		    SELECT max(r.strength) AS strength
		    FROM relationships r
		    WHERE r.project_id = c.project_id AND r.community_id = c.id
		) rel ON TRUE
		WHERE c.project_id = $1
		  AND c.status NOT IN ('do_not_contact','paused')
		  AND t.status <> 'retired'
		  AND (c.topics && t.topics OR rel.strength IS NOT NULL)
		ORDER BY (c.topics && t.topics) DESC,
		         rel.strength DESC NULLS LAST,
		         c.speaking_fit_score DESC NULLS LAST,
		         c.activity_score DESC NULLS LAST,
		         (t.status = 'ready') DESC,
		         c.name, t.title
		LIMIT $2`, projectID, budget)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]SpeakingCandidate, 0, 64)
	for rows.Next() {
		var candidate SpeakingCandidate
		c := &candidate.Community
		t := &candidate.Talk
		if err := rows.Scan(
			&c.ID, &c.ProjectID, &c.Name, &c.Platform, &c.ExternalID, &c.WebsiteURL,
			&c.City, &c.Country, &c.Timezone, &c.Topics, &c.MemberCount, &c.ActivityScore,
			&c.SpeakingFitScore, &c.LastEventAt, &c.NextEventAt, &c.Status, &c.CreatedAt, &c.UpdatedAt,
			&t.ID, &t.ProjectID, &t.Title, &t.Abstract, &t.Description, &t.Level,
			&t.DurationMinutes, &t.Topics, &t.DemoURL, &t.SlidesURL, &t.RecordingURL,
			&t.Status, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, candidate)
	}
	return out, rows.Err()
}
