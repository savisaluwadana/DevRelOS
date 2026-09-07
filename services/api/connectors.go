package main

import (
	"net/http"
	"strings"

	domain "github.com/savisaluwadana/DevRelOS/internal/domain/connectors"
)

func (a *api) workspaceID(r *http.Request) (string, error) {
	if id := strings.TrimSpace(r.URL.Query().Get("workspaceId")); id != "" {
		return id, nil
	}
	return a.store.DefaultWorkspaceID(r.Context())
}

func (a *api) listConnectors(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := a.workspaceID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.ListConnectors(r.Context(), workspaceID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createConnector(w http.ResponseWriter, r *http.Request) {
	var input domain.Connector
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if strings.TrimSpace(input.Provider) == "" || strings.TrimSpace(input.Name) == "" {
		writeBadRequest(w, "provider and name are required")
		return
	}
	if input.ScheduleMinutes != nil && (*input.ScheduleMinutes < 15 || *input.ScheduleMinutes > 10080) {
		writeBadRequest(w, "scheduleMinutes must be between 15 and 10080 minutes")
		return
	}
	if input.WorkspaceID == "" {
		var err error
		input.WorkspaceID, err = a.workspaceID(r)
		if err != nil {
			writeError(w, err)
			return
		}
	}
	created, err := a.store.CreateConnector(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) updateConnectorSchedule(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ScheduleMinutes *int `json:"scheduleMinutes"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	if input.ScheduleMinutes != nil && (*input.ScheduleMinutes < 15 || *input.ScheduleMinutes > 10080) {
		writeBadRequest(w, "scheduleMinutes must be between 15 and 10080 minutes")
		return
	}
	workspaceID, err := a.workspaceID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	nextRunAt, err := a.store.UpdateConnectorSchedule(r.Context(), workspaceID, r.PathValue("id"), input.ScheduleMinutes)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"scheduleMinutes": input.ScheduleMinutes,
		"nextRunAt":       nextRunAt,
	})
}

func (a *api) listConnectorRuns(w http.ResponseWriter, r *http.Request) {
	items, err := a.store.ListConnectorRuns(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) queueConnectorRun(w http.ResponseWriter, r *http.Request) {
	run, err := a.store.QueueConnectorRun(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, run)
}
