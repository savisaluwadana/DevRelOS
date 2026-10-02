package main

import (
	"net/http"
	"strings"

	campaigndomain "github.com/savisaluwadana/DevRelOS/internal/domain/campaigns"
)

func (a *api) registerCampaignRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/campaigns", a.listCampaigns)
	mux.HandleFunc("POST /api/v1/campaigns", a.createCampaign)
	mux.HandleFunc("PATCH /api/v1/campaigns/{id}", a.updateCampaign)
	mux.HandleFunc("DELETE /api/v1/campaigns/{id}", a.deleteCampaign)
	mux.HandleFunc("GET /api/v1/campaigns/{id}/items", a.listCampaignItems)
	mux.HandleFunc("POST /api/v1/campaigns/{id}/items", a.linkCampaignItem)
	mux.HandleFunc("DELETE /api/v1/campaigns/{id}/items/{itemId}", a.deleteCampaignItem)
	mux.HandleFunc("POST /api/v1/campaigns/{id}/metrics", a.recordCampaignMetric)
	mux.HandleFunc("GET /api/v1/campaigns/{id}/report", a.campaignReport)
	mux.HandleFunc("GET /api/v1/relationships/radar", a.relationshipRadar)
}

func (a *api) listCampaigns(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.ListCampaigns(r.Context(), projectID, requestPage(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createCampaign(w http.ResponseWriter, r *http.Request) {
	var input campaigndomain.Campaign
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		writeBadRequest(w, "name is required")
		return
	}
	if input.BudgetUSD < 0 {
		writeBadRequest(w, "budgetUsd cannot be negative")
		return
	}
	if input.Status != "" && !validCampaignStatus(input.Status) {
		writeBadRequest(w, "invalid campaign status")
		return
	}
	if input.StartsAt != nil && input.EndsAt != nil && input.EndsAt.Before(*input.StartsAt) {
		writeBadRequest(w, "endsAt cannot be before startsAt")
		return
	}
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	input.ProjectID = projectID
	created, err := a.store.CreateCampaign(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) updateCampaign(w http.ResponseWriter, r *http.Request) {
	var input campaigndomain.CampaignUpdate
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if input.Status != nil && !validCampaignStatus(*input.Status) {
		writeBadRequest(w, "invalid campaign status")
		return
	}
	if input.BudgetUSD != nil && *input.BudgetUSD < 0 {
		writeBadRequest(w, "budgetUsd cannot be negative")
		return
	}
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	campaign, err := a.store.GetCampaign(r.Context(), projectID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if input.Status != nil && *input.Status != campaign.Status && !campaignStatusTransitionAllowed(campaign.Status, *input.Status) {
		writeBadRequest(w, "invalid campaign status transition")
		return
	}
	startsAt := campaign.StartsAt
	if input.StartsAt != nil {
		startsAt = input.StartsAt
	}
	endsAt := campaign.EndsAt
	if input.EndsAt != nil {
		endsAt = input.EndsAt
	}
	if startsAt != nil && endsAt != nil && endsAt.Before(*startsAt) {
		writeBadRequest(w, "endsAt cannot be before startsAt")
		return
	}
	updated, err := a.store.UpdateCampaign(r.Context(), projectID, campaign.ID, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (a *api) deleteCampaign(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := a.store.DeleteCampaign(r.Context(), projectID, r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) listCampaignItems(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.ListCampaignItems(r.Context(), projectID, r.PathValue("id"), requestPage(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) linkCampaignItem(w http.ResponseWriter, r *http.Request) {
	var input campaigndomain.Item
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	input.EntityType = strings.TrimSpace(input.EntityType)
	input.EntityID = strings.TrimSpace(input.EntityID)
	if input.EntityType == "" || input.EntityID == "" {
		writeBadRequest(w, "entityType and entityId are required")
		return
	}
	if input.CostUSD < 0 {
		writeBadRequest(w, "costUsd cannot be negative")
		return
	}
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if _, err := a.store.GetCampaign(r.Context(), projectID, r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	belongs, err := a.store.CampaignEntityBelongsToProject(r.Context(), projectID, input.EntityType, input.EntityID)
	if err != nil {
		// This is a database error, not a description of the client's JSON.
		// Passing err.Error() to writeBadRequest leaked the raw Postgres
		// message - type names and SQLSTATE codes - into the response. Let
		// writeError classify it (a malformed entityId becomes a clean 400).
		writeError(w, err)
		return
	}
	if !belongs {
		writeBadRequest(w, "entity does not belong to the active project")
		return
	}
	input.CampaignID = r.PathValue("id")
	created, err := a.store.LinkCampaignItem(r.Context(), projectID, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) deleteCampaignItem(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := a.store.DeleteCampaignItem(r.Context(), projectID, r.PathValue("id"), r.PathValue("itemId")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) recordCampaignMetric(w http.ResponseWriter, r *http.Request) {
	var input campaigndomain.Metric
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	input.MetricKey = strings.TrimSpace(input.MetricKey)
	if input.MetricKey == "" {
		writeBadRequest(w, "metricKey is required")
		return
	}
	if len(input.MetricKey) > 80 {
		writeBadRequest(w, "metricKey is too long")
		return
	}
	input.CampaignID = r.PathValue("id")
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	created, err := a.store.RecordCampaignMetric(r.Context(), projectID, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) campaignReport(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	report, err := a.store.CampaignReport(r.Context(), projectID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (a *api) relationshipRadar(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.RelationshipRadar(r.Context(), projectID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func validCampaignStatus(status string) bool {
	switch status {
	case "planning", "active", "paused", "completed", "archived":
		return true
	default:
		return false
	}
}

func campaignStatusTransitionAllowed(from, to string) bool {
	if from == to {
		return true
	}
	allowed := map[string]map[string]bool{
		"planning":  {"active": true, "archived": true},
		"active":    {"paused": true, "completed": true, "archived": true},
		"paused":    {"active": true, "completed": true, "archived": true},
		"completed": {"archived": true, "active": true},
		"archived":  {},
	}
	return allowed[from][to]
}
