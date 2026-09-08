package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/savisaluwadana/DevRelOS/internal/intelligence/opportunities"
	"github.com/savisaluwadana/DevRelOS/internal/storage"
)

func (a *api) listSpeakingOpportunities(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	minScore := boundedIntQuery(r, "minScore", 0, 0, 100)
	limit := boundedIntQuery(r, "limit", 50, 1, 200)

	// Candidate selection happens in SQL and is bounded, so the request no
	// longer materialises the community x talk cross product to return 50 rows.
	candidates, err := a.store.ListSpeakingCandidates(r.Context(), projectID, storage.SpeakingCandidateBudget(limit))
	if err != nil {
		writeError(w, err)
		return
	}
	// Relationships and outreach are read per community rather than per pair,
	// so these stay linear in project size.
	relationships, err := a.store.ListRelationships(r.Context(), projectID, storage.AllRows())
	if err != nil {
		writeError(w, err)
		return
	}
	outreachItems, err := a.store.ListOutreach(r.Context(), projectID, storage.AllRows())
	if err != nil {
		writeError(w, err)
		return
	}

	pairs := make([]opportunities.Pair, 0, len(candidates))
	for _, candidate := range candidates {
		pairs = append(pairs, opportunities.Pair{Community: candidate.Community, Talk: candidate.Talk})
	}
	items := opportunities.RankSpeakingPairs(pairs, relationships, outreachItems, time.Now().UTC())

	filtered := make([]opportunities.SpeakingOpportunity, 0, min(limit, len(items)))
	for _, item := range items {
		if item.Score < minScore {
			continue
		}
		filtered = append(filtered, item)
		if len(filtered) >= limit {
			break
		}
	}
	writeJSON(w, http.StatusOK, filtered)
}

func (a *api) listCFPOpportunities(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	cfps, err := a.store.ListCFPs(r.Context(), projectID, storage.AllRows())
	if err != nil {
		writeError(w, err)
		return
	}
	eventsList, err := a.store.ListEvents(r.Context(), projectID, storage.AllRows())
	if err != nil {
		writeError(w, err)
		return
	}
	talks, err := a.store.ListTalks(r.Context(), projectID, storage.AllRows())
	if err != nil {
		writeError(w, err)
		return
	}
	submissions, err := a.store.ListSubmissions(r.Context(), projectID, storage.AllRows())
	if err != nil {
		writeError(w, err)
		return
	}

	items := opportunities.RankCFPs(cfps, eventsList, talks, submissions, time.Now().UTC())
	minScore := boundedIntQuery(r, "minScore", 0, 0, 100)
	limit := boundedIntQuery(r, "limit", 50, 1, 200)
	filtered := make([]opportunities.CFPOpportunity, 0, min(limit, len(items)))
	for _, item := range items {
		if item.Score < minScore {
			continue
		}
		filtered = append(filtered, item)
		if len(filtered) >= limit {
			break
		}
	}
	writeJSON(w, http.StatusOK, filtered)
}

func boundedIntQuery(r *http.Request, key string, fallback, minValue, maxValue int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	if parsed < minValue {
		return minValue
	}
	if parsed > maxValue {
		return maxValue
	}
	return parsed
}
