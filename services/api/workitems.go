package main

import (
	"net/http"
	"strings"

	workdomain "github.com/savisaluwadana/DevRelOS/internal/domain/workitems"
	workintelligence "github.com/savisaluwadana/DevRelOS/internal/intelligence/workitems"
	"github.com/savisaluwadana/DevRelOS/internal/storage"
)

func (a *api) registerWorkItemRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/work-items", a.listWorkItems)
	mux.HandleFunc("POST /api/v1/work-items", a.createWorkItem)
	mux.HandleFunc("PATCH /api/v1/work-items/{id}/status", a.updateWorkItemStatus)
	mux.HandleFunc("POST /api/v1/pain-points/{id}/work-items", a.createWorkItemFromPainPoint)
	a.registerContentRoutes(mux)
}

func (a *api) listWorkItems(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.ListWorkItems(r.Context(), projectID, storage.WorkItemFilter{
		Status: strings.TrimSpace(r.URL.Query().Get("status")),
		Kind:   strings.TrimSpace(r.URL.Query().Get("kind")),
		Limit:  intQuery(r, "limit", 200),
		Offset: intQuery(r, "offset", 0),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createWorkItem(w http.ResponseWriter, r *http.Request) {
	var input workdomain.WorkItem
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if !workintelligence.ValidKind(input.Kind) {
		writeBadRequest(w, "invalid work item kind")
		return
	}
	if strings.TrimSpace(input.Title) == "" {
		writeBadRequest(w, "title is required")
		return
	}
	if input.Priority < 0 || input.Priority > 100 {
		writeBadRequest(w, "priority must be between 0 and 100")
		return
	}
	if input.Status != "" && !validWorkStatus(input.Status) {
		writeBadRequest(w, "invalid work item status")
		return
	}
	if input.SourceType == "" {
		input.SourceType = "manual"
	}
	if !validWorkSource(input.SourceType) {
		writeBadRequest(w, "invalid work item sourceType")
		return
	}
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	input.ProjectID = projectID
	created, err := a.store.CreateWorkItem(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) updateWorkItemStatus(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if !validWorkStatus(input.Status) {
		writeBadRequest(w, "invalid work item status")
		return
	}
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := a.store.UpdateWorkItemStatus(r.Context(), projectID, r.PathValue("id"), input.Status); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": input.Status})
}

func (a *api) createWorkItemFromPainPoint(w http.ResponseWriter, r *http.Request) {
	var input workdomain.PainPointConversion
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if !workintelligence.ValidKind(input.Kind) {
		writeBadRequest(w, "invalid work item kind")
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
	exists, err := a.store.WorkItemExistsForSource(r.Context(), projectID, "pain_point", painPoint.ID, input.Kind)
	if err != nil {
		writeError(w, err)
		return
	}
	if exists {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "this pain point already has a work item of that kind"})
		return
	}
	item, err := workintelligence.FromPainPoint(painPoint, input.Kind, strings.TrimSpace(input.Owner))
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	item.DueAt = input.DueAt
	created, err := a.store.CreateWorkItem(r.Context(), item)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func validWorkStatus(status string) bool {
	return map[string]bool{
		"backlog": true, "planned": true, "in_progress": true,
		"blocked": true, "done": true, "cancelled": true,
	}[status]
}

func validWorkSource(source string) bool {
	return map[string]bool{
		"manual": true, "pain_point": true, "signal": true, "community": true,
		"cfp": true, "outreach": true, "campaign": true,
	}[source]
}
