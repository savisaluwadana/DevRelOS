package main

import (
	"net/http"
	"strings"

	feedbackdomain "github.com/savisaluwadana/DevRelOS/internal/domain/feedback"
	feedbackintelligence "github.com/savisaluwadana/DevRelOS/internal/intelligence/feedback"
	"github.com/savisaluwadana/DevRelOS/internal/storage"
)

func (a *api) registerFeedbackRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/feedback", a.listFeedback)
	mux.HandleFunc("POST /api/v1/feedback", a.createFeedback)
	mux.HandleFunc("PATCH /api/v1/feedback/{id}", a.updateFeedback)
	mux.HandleFunc("GET /api/v1/feedback/{id}/github/prefill", a.feedbackGitHubPrefill)
	mux.HandleFunc("POST /api/v1/feedback/{id}/github/sync", a.syncFeedbackGitHubIssue)
	mux.HandleFunc("POST /api/v1/pain-points/{id}/feedback", a.createFeedbackFromPainPoint)
}

func (a *api) listFeedback(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.ListFeedback(r.Context(), projectID, storage.FeedbackFilter{
		Status:    strings.TrimSpace(r.URL.Query().Get("status")),
		Component: strings.TrimSpace(r.URL.Query().Get("component")),
		Limit:     intQuery(r, "limit", 200),
		Offset:    intQuery(r, "offset", 0),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createFeedback(w http.ResponseWriter, r *http.Request) {
	var input feedbackdomain.Item
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if strings.TrimSpace(input.Title) == "" {
		writeBadRequest(w, "title is required")
		return
	}
	if input.SourceType == "" {
		input.SourceType = "manual"
	}
	if !validFeedbackSource(input.SourceType) {
		writeBadRequest(w, "invalid feedback sourceType")
		return
	}
	if input.Status != "" && !validFeedbackStatus(input.Status) {
		writeBadRequest(w, "invalid feedback status")
		return
	}
	if !scoreInRange(input.ImpactScore) || !scoreInRange(input.FrequencyScore) {
		writeBadRequest(w, "impactScore and frequencyScore must be between 0 and 100")
		return
	}
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	input.ProjectID = projectID
	created, err := a.store.CreateFeedback(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) updateFeedback(w http.ResponseWriter, r *http.Request) {
	var input feedbackdomain.Update
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if input.Status != nil && !validFeedbackStatus(*input.Status) {
		writeBadRequest(w, "invalid feedback status")
		return
	}
	if input.ImpactScore != nil && !scoreInRange(*input.ImpactScore) {
		writeBadRequest(w, "impactScore must be between 0 and 100")
		return
	}
	if input.FrequencyScore != nil && !scoreInRange(*input.FrequencyScore) {
		writeBadRequest(w, "frequencyScore must be between 0 and 100")
		return
	}
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	current, err := a.store.GetFeedback(r.Context(), projectID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if input.Status != nil && *input.Status != current.Status && !validFeedbackTransition(current.Status, *input.Status) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid feedback status transition"})
		return
	}
	if input.Status != nil && *input.Status == "shipped" {
		followUp := strings.TrimSpace(current.FollowUpNote)
		if input.FollowUpNote != nil {
			followUp = strings.TrimSpace(*input.FollowUpNote)
		}
		if followUp == "" {
			writeBadRequest(w, "followUpNote is required before marking feedback shipped")
			return
		}
	}
	updated, err := a.store.UpdateFeedback(r.Context(), projectID, current.ID, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (a *api) createFeedbackFromPainPoint(w http.ResponseWriter, r *http.Request) {
	var input feedbackdomain.PainPointConversion
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	painPoint, err := a.store.GetPainPoint(r.Context(), projectID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	exists, err := a.store.FeedbackExistsForPainPoint(r.Context(), projectID, painPoint.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	if exists {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "this pain point already has a feedback item"})
		return
	}
	item := feedbackintelligence.FromPainPoint(
		painPoint,
		strings.TrimSpace(input.Component),
		strings.TrimSpace(input.Owner),
		strings.TrimSpace(input.GitHubRepository),
	)
	created, err := a.store.CreateFeedback(r.Context(), item)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func validFeedbackStatus(status string) bool {
	return map[string]bool{
		"new": true, "triaged": true, "planned": true, "in_progress": true,
		"shipped": true, "closed": true, "wont_fix": true,
	}[status]
}

func validFeedbackSource(source string) bool {
	return map[string]bool{"manual": true, "pain_point": true, "signal": true, "work_item": true}[source]
}

func validFeedbackTransition(from, to string) bool {
	allowed := map[string]map[string]bool{
		"new":         {"triaged": true, "closed": true, "wont_fix": true},
		"triaged":     {"planned": true, "closed": true, "wont_fix": true},
		"planned":     {"in_progress": true, "triaged": true, "wont_fix": true},
		"in_progress": {"shipped": true, "planned": true, "wont_fix": true},
		"shipped":     {"closed": true},
		"closed":      {"triaged": true},
		"wont_fix":    {"triaged": true},
	}
	return allowed[from][to]
}

func scoreInRange(value int) bool { return value >= 0 && value <= 100 }
