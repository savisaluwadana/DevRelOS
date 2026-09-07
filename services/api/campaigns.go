package main

import (
	"net/http"
	"strings"

	campaigndomain "github.com/savisaluwadana/DevRelOS/internal/domain/campaigns"
)

func (a *api) registerCampaignRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/campaigns", a.listCampaigns)
	mux.HandleFunc("POST /api/v1/campaigns", a.createCampaign)
	mux.HandleFunc("PATCH /api/v1/campaigns/{id}/status", a.updateCampaignStatus)
	mux.HandleFunc("POST /api/v1/campaigns/{id}/items", a.linkCampaignItem)
	mux.HandleFunc("POST /api/v1/campaigns/{id}/metrics", a.recordCampaignMetric)
	mux.HandleFunc("GET /api/v1/campaigns/{id}/report", a.campaignReport)
	mux.HandleFunc("GET /api/v1/relationships/radar", a.relationshipRadar)
}

func (a *api) listCampaigns(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil { writeError(w, err); return }
	items, err := a.store.ListCampaigns(r.Context(), projectID)
	if err != nil { writeError(w, err); return }
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createCampaign(w http.ResponseWriter, r *http.Request) {
	var input campaigndomain.Campaign
	if err := decodeJSON(r, &input); err != nil { writeBadRequest(w, err.Error()); return }
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" { writeBadRequest(w, "name is required"); return }
	if input.BudgetUSD < 0 { writeBadRequest(w, "budgetUsd cannot be negative"); return }
	if input.Status != "" && !validCampaignStatus(input.Status) { writeBadRequest(w, "invalid campaign status"); return }
	if input.StartsAt != nil && input.EndsAt != nil && input.EndsAt.Before(*input.StartsAt) { writeBadRequest(w, "endsAt cannot be before startsAt"); return }
	projectID, err := a.projectID(r)
	if err != nil { writeError(w, err); return }
	input.ProjectID = projectID
	created, err := a.store.CreateCampaign(r.Context(), input)
	if err != nil { writeError(w, err); return }
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) updateCampaignStatus(w http.ResponseWriter, r *http.Request) {
	var input struct { Status string `json:"status"` }
	if err := decodeJSON(r, &input); err != nil { writeBadRequest(w, err.Error()); return }
	if !validCampaignStatus(input.Status) { writeBadRequest(w, "invalid campaign status"); return }
	projectID, err := a.projectID(r)
	if err != nil { writeError(w, err); return }
	campaign, err := a.store.GetCampaign(r.Context(), projectID, r.PathValue("id"))
	if err != nil { writeError(w, err); return }
	if !campaignStatusTransitionAllowed(campaign.Status, input.Status) { writeBadRequest(w, "invalid campaign status transition"); return }
	if err := a.store.UpdateCampaignStatus(r.Context(), projectID, campaign.ID, input.Status); err != nil { writeError(w, err); return }
	writeJSON(w, http.StatusOK, map[string]string{"status": input.Status})
}

func (a *api) linkCampaignItem(w http.ResponseWriter, r *http.Request) {
	var input campaigndomain.Item
	if err := decodeJSON(r, &input); err != nil { writeBadRequest(w, err.Error()); return }
	input.EntityType = strings.TrimSpace(input.EntityType)
	input.EntityID = strings.TrimSpace(input.EntityID)
	if input.EntityType == "" || input.EntityID == "" { writeBadRequest(w, "entityType and entityId are required"); return }
	if input.CostUSD < 0 { writeBadRequest(w, "costUsd cannot be negative"); return }
	projectID, err := a.projectID(r)
	if err != nil { writeError(w, err); return }
	if _, err := a.store.GetCampaign(r.Context(), projectID, r.PathValue("id")); err != nil { writeError(w, err); return }
	belongs, err := a.store.CampaignEntityBelongsToProject(r.Context(), projectID, input.EntityType, input.EntityID)
	if err != nil { writeBadRequest(w, err.Error()); return }
	if !belongs { writeBadRequest(w, "entity does not belong to the active project"); return }
	input.CampaignID = r.PathValue("id")
	created, err := a.store.LinkCampaignItem(r.Context(), projectID, input)
	if err != nil { writeError(w, err); return }
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) recordCampaignMetric(w http.ResponseWriter, r *http.Request) {
	var input campaigndomain.Metric
	if err := decodeJSON(r, &input); err != nil { writeBadRequest(w, err.Error()); return }
	input.MetricKey = strings.TrimSpace(input.MetricKey)
	if input.MetricKey == "" { writeBadRequest(w, "metricKey is required"); return }
	if len(input.MetricKey) > 80 { writeBadRequest(w, "metricKey is too long"); return }
	input.CampaignID = r.PathValue("id")
	projectID, err := a.projectID(r)
	if err != nil { writeError(w, err); return }
	created, err := a.store.RecordCampaignMetric(r.Context(), projectID, input)
	if err != nil { writeError(w, err); return }
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) campaignReport(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil { writeError(w, err); return }
	report, err := a.store.CampaignReport(r.Context(), projectID, r.PathValue("id"))
	if err != nil { writeError(w, err); return }
	writeJSON(w, http.StatusOK, report)
}

func (a *api) relationshipRadar(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil { writeError(w, err); return }
	items, err := a.store.RelationshipRadar(r.Context(), projectID)
	if err != nil { writeError(w, err); return }
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
	if from == to { return true }
	allowed := map[string]map[string]bool{
		"planning": {"active": true, "archived": true},
		"active": {"paused": true, "completed": true, "archived": true},
		"paused": {"active": true, "completed": true, "archived": true},
		"completed": {"archived": true, "active": true},
		"archived": {},
	}
	return allowed[from][to]
}
