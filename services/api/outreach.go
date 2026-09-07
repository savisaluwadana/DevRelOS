package main

import (
	"net/http"
	"strings"
	"time"

	domain "github.com/savisaluwadana/DevRelOS/internal/domain/outreach"
)

func (a *api) registerOutreachRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/contacts", a.listContacts)
	mux.HandleFunc("POST /api/v1/contacts", a.createContact)
	mux.HandleFunc("GET /api/v1/relationships", a.listRelationships)
	mux.HandleFunc("POST /api/v1/relationships", a.createRelationship)
	mux.HandleFunc("GET /api/v1/touchpoints", a.listTouchpoints)
	mux.HandleFunc("POST /api/v1/touchpoints", a.createTouchpoint)
	mux.HandleFunc("GET /api/v1/outreach", a.listOutreach)
	mux.HandleFunc("POST /api/v1/outreach", a.createOutreach)
	mux.HandleFunc("PATCH /api/v1/outreach/{id}/status", a.updateOutreachStatus)
}

func (a *api) listContacts(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := a.workspaceID(r)
	if err != nil { writeError(w, err); return }
	items, err := a.store.ListContacts(r.Context(), workspaceID)
	if err != nil { writeError(w, err); return }
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createContact(w http.ResponseWriter, r *http.Request) {
	var input domain.Contact
	if err := decodeJSON(r, &input); err != nil { writeBadRequest(w, err.Error()); return }
	if strings.TrimSpace(input.Name) == "" { writeBadRequest(w, "name is required"); return }
	workspaceID, err := a.workspaceID(r)
	if err != nil { writeError(w, err); return }
	input.WorkspaceID = workspaceID
	created, err := a.store.CreateContact(r.Context(), input)
	if err != nil { writeError(w, err); return }
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) listRelationships(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil { writeError(w, err); return }
	items, err := a.store.ListRelationships(r.Context(), projectID)
	if err != nil { writeError(w, err); return }
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createRelationship(w http.ResponseWriter, r *http.Request) {
	var input domain.Relationship
	if err := decodeJSON(r, &input); err != nil { writeBadRequest(w, err.Error()); return }
	allowedStages := map[string]bool{"": true, "cold": true, "warm": true, "engaged": true, "partner": true, "dormant": true}
	if !allowedStages[input.Stage] { writeBadRequest(w, "invalid relationship stage"); return }
	if input.Strength < 0 || input.Strength > 100 { writeBadRequest(w, "strength must be between 0 and 100"); return }
	projectID, err := a.projectID(r)
	if err != nil { writeError(w, err); return }
	input.ProjectID = projectID
	created, err := a.store.CreateRelationship(r.Context(), input)
	if err != nil { writeError(w, err); return }
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) listTouchpoints(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil { writeError(w, err); return }
	items, err := a.store.ListTouchpoints(r.Context(), projectID, strings.TrimSpace(r.URL.Query().Get("relationshipId")), intQuery(r, "limit", 50))
	if err != nil { writeError(w, err); return }
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createTouchpoint(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RelationshipID string     `json:"relationshipId"`
		Channel        string     `json:"channel"`
		Direction      string     `json:"direction"`
		Summary        string     `json:"summary"`
		OccurredAt     *time.Time `json:"occurredAt,omitempty"`
		NextFollowUpAt *time.Time `json:"nextFollowUpAt,omitempty"`
	}
	if err := decodeJSON(r, &input); err != nil { writeBadRequest(w, err.Error()); return }
	if input.RelationshipID == "" || strings.TrimSpace(input.Channel) == "" || strings.TrimSpace(input.Summary) == "" {
		writeBadRequest(w, "relationshipId, channel and summary are required"); return
	}
	if input.Direction != "inbound" && input.Direction != "outbound" && input.Direction != "internal" {
		writeBadRequest(w, "direction must be inbound, outbound or internal"); return
	}
	projectID, err := a.projectID(r)
	if err != nil { writeError(w, err); return }
	occurredAt := time.Now().UTC()
	if input.OccurredAt != nil { occurredAt = input.OccurredAt.UTC() }
	created, err := a.store.AddTouchpoint(r.Context(), projectID, domain.Touchpoint{
		RelationshipID: input.RelationshipID,
		Channel: strings.TrimSpace(input.Channel),
		Direction: input.Direction,
		Summary: strings.TrimSpace(input.Summary),
		OccurredAt: occurredAt,
	}, input.NextFollowUpAt)
	if err != nil { writeError(w, err); return }
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) listOutreach(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.projectID(r)
	if err != nil { writeError(w, err); return }
	items, err := a.store.ListOutreach(r.Context(), projectID)
	if err != nil { writeError(w, err); return }
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createOutreach(w http.ResponseWriter, r *http.Request) {
	var input domain.Outreach
	if err := decodeJSON(r, &input); err != nil { writeBadRequest(w, err.Error()); return }
	input.Channel = strings.ToLower(strings.TrimSpace(input.Channel))
	if input.Channel == "" || strings.TrimSpace(input.Body) == "" { writeBadRequest(w, "channel and body are required"); return }
	if input.CommunityID == "" && input.ContactID == "" { writeBadRequest(w, "communityId or contactId is required"); return }
	if input.Status != "" && input.Status != "draft" && input.Status != "needs_approval" {
		writeBadRequest(w, "new outreach can only be draft or needs_approval"); return
	}
	projectID, err := a.projectID(r)
	if err != nil { writeError(w, err); return }
	input.ProjectID = projectID
	created, err := a.store.CreateOutreach(r.Context(), input)
	if err != nil { writeError(w, err); return }
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) updateOutreachStatus(w http.ResponseWriter, r *http.Request) {
	var input struct { Status string `json:"status"` }
	if err := decodeJSON(r, &input); err != nil { writeBadRequest(w, err.Error()); return }
	projectID, err := a.projectID(r)
	if err != nil { writeError(w, err); return }
	state, err := a.store.OutreachState(r.Context(), projectID, r.PathValue("id"))
	if err != nil { writeError(w, err); return }
	if !validOutreachTransition(state.Status, input.Status) { writeBadRequest(w, "invalid outreach status transition"); return }

	if state.Channel == "email" && input.Status == "sent" {
		writeBadRequest(w, "email outreach is marked sent only after SMTP delivery succeeds")
		return
	}
	if state.Channel == "email" && input.Status == "queued" {
		delivery, queueErr := a.store.QueueOutreachDelivery(r.Context(), projectID, r.PathValue("id"))
		if queueErr != nil { writeBadRequest(w, queueErr.Error()); return }
		writeJSON(w, http.StatusAccepted, map[string]any{"status": "queued", "deliveryId": delivery.ID, "transport": delivery.Transport})
		return
	}

	if err := a.store.UpdateOutreachStatus(r.Context(), projectID, r.PathValue("id"), input.Status); err != nil {
		writeError(w, err); return
	}
	if input.Status == "cancelled" {
		_ = a.store.CancelOutreachDelivery(r.Context(), projectID, r.PathValue("id"))
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": input.Status})
}

func validOutreachTransition(current, next string) bool {
	allowed := map[string]map[string]bool{
		"draft": {"needs_approval": true, "cancelled": true},
		"needs_approval": {"draft": true, "approved": true, "cancelled": true},
		"approved": {"queued": true, "cancelled": true},
		"queued": {"sent": true, "failed": true, "cancelled": true},
		"sent": {"replied": true},
		"failed": {"queued": true, "cancelled": true},
	}
	return allowed[current][next]
}
