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
	return a.requestWorkspaceID(r)
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
	input.Provider = strings.TrimSpace(input.Provider)
	input.Name = strings.TrimSpace(input.Name)
	input.SecretID = strings.TrimSpace(input.SecretID)
	if input.Provider == "" || input.Name == "" {
		writeBadRequest(w, "provider and name are required")
		return
	}
	if hasPlaintextCredential(input.Config) {
		writeBadRequest(w, "connector config must not contain plaintext credentials; use secretId or token_env")
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
	if input.SecretID != "" {
		belongs, belongsErr := a.store.ConnectorSecretBelongs(r.Context(), workspaceID, input.SecretID, input.Provider)
		if belongsErr != nil { writeError(w, belongsErr); return }
		if !belongs { writeBadRequest(w, "secret does not belong to this workspace/provider"); return }
	}
	input.WorkspaceID = workspaceID
	created, err := a.store.CreateConnector(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func hasPlaintextCredential(config map[string]any) bool {
	for key, value := range config {
		normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "-", "_"))
		switch normalized {
		case "token", "access_token", "api_key", "apikey", "password", "client_secret", "secret", "bearer_token":
			if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
				return true
			}
		}
	}
	return false
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
	workspaceID, err := a.workspaceID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.ListConnectorRunsForWorkspace(r.Context(), workspaceID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) queueConnectorRun(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := a.workspaceID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	run, err := a.store.QueueConnectorRunForWorkspace(r.Context(), workspaceID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, run)
}
