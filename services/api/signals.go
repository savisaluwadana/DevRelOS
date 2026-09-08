package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	domain "github.com/savisaluwadana/DevRelOS/internal/domain/signals"
	intelligence "github.com/savisaluwadana/DevRelOS/internal/intelligence/signals"
	"github.com/savisaluwadana/DevRelOS/internal/storage"
)

func (a *api) registerSignalRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/signals", a.listSignals)
	mux.HandleFunc("POST /api/v1/signals", a.createSignal)
	mux.HandleFunc("PATCH /api/v1/signals/{id}/status", a.updateSignalStatus)
	mux.HandleFunc("GET /api/v1/pain-points", a.listPainPoints)
	mux.HandleFunc("GET /api/v1/pain-points/{id}/evidence", a.listPainPointEvidence)
	mux.HandleFunc("POST /api/v1/pain-points/rebuild", a.rebuildPainPoints)
	a.registerFeedbackRoutes(mux)
}

func (a *api) listSignals(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	filter := storage.SignalFilter{
		Status:   strings.TrimSpace(r.URL.Query().Get("status")),
		Provider: strings.TrimSpace(r.URL.Query().Get("provider")),
		Topic:    strings.TrimSpace(r.URL.Query().Get("topic")),
		Query:    strings.TrimSpace(r.URL.Query().Get("q")),
		Limit:    intQuery(r, "limit", 100),
	}
	items, err := a.store.ListSignals(r.Context(), projectID, filter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createSignal(w http.ResponseWriter, r *http.Request) {
	var input domain.Signal
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	input.Provider = strings.TrimSpace(input.Provider)
	if input.Provider == "" {
		writeBadRequest(w, "provider is required")
		return
	}
	if strings.TrimSpace(input.Title) == "" && strings.TrimSpace(input.Body) == "" {
		writeBadRequest(w, "title or body is required")
		return
	}
	if input.RelevanceScore != nil && (*input.RelevanceScore < 0 || *input.RelevanceScore > 100) {
		writeBadRequest(w, "relevanceScore must be between 0 and 100")
		return
	}
	if input.EngagementScore < 0 {
		writeBadRequest(w, "engagementScore cannot be negative")
		return
	}
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	input.ProjectID = projectID
	created, err := a.store.CreateSignal(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) updateSignalStatus(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	allowed := map[string]bool{"new": true, "reviewed": true, "ignored": true, "converted": true}
	if !allowed[input.Status] {
		writeBadRequest(w, "invalid signal status")
		return
	}
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := a.store.UpdateSignalStatus(r.Context(), r.PathValue("id"), projectID, input.Status); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": input.Status})
}

func (a *api) listPainPoints(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.ListPainPoints(
		r.Context(), projectID, strings.TrimSpace(r.URL.Query().Get("status")), intQuery(r, "limit", 100),
	)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) listPainPointEvidence(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.ListPainPointEvidence(r.Context(), projectID, r.PathValue("id"), intQuery(r, "limit", 50))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) rebuildPainPoints(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.ListSignals(r.Context(), projectID, storage.SignalFilter{Limit: 500})
	if err != nil {
		writeError(w, err)
		return
	}
	rules, err := intelligence.ActiveRuleset()
	if err != nil {
		// A bad rules file is a configuration error, not a reason to silently
		// cluster with the wrong vocabulary.
		writeError(w, err)
		return
	}
	clusters := intelligence.ClusterSignalsWith(rules, items, time.Now().UTC())
	persistable := make([]storage.PainPointCluster, 0, len(clusters))
	for _, cluster := range clusters {
		persistable = append(persistable, storage.PainPointCluster{
			Key: cluster.Key, Title: cluster.Title, Summary: cluster.Summary, Persona: cluster.Persona,
			Severity: cluster.Severity, TrendScore: cluster.TrendScore, Topics: cluster.Topics,
			FirstSeen: cluster.FirstSeen, LastSeen: cluster.LastSeen, SignalIDs: cluster.SignalIDs,
		})
	}
	if err := a.store.ReplacePainPointClusters(r.Context(), projectID, persistable); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"signalsConsidered": len(items),
		"clustersCreated":   len(clusters),
	})
}

func intQuery(r *http.Request, key string, fallback int) int {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
