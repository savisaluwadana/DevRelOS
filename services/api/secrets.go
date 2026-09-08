package main

import (
	"errors"
	"net/http"
	"os"
	"strings"

	connectordomain "github.com/savisaluwadana/DevRelOS/internal/domain/connectors"
	identitydomain "github.com/savisaluwadana/DevRelOS/internal/domain/identity"
	"github.com/savisaluwadana/DevRelOS/internal/security"
	"github.com/savisaluwadana/DevRelOS/internal/storage"
)

func (a *api) registerSecretRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/secrets", a.listSecrets)
	mux.HandleFunc("POST /api/v1/secrets", a.createSecret)
	mux.HandleFunc("PUT /api/v1/secrets/{id}", a.rotateSecret)
	mux.HandleFunc("DELETE /api/v1/secrets/{id}", a.deleteSecret)
	mux.HandleFunc("PUT /api/v1/connectors/{id}/secret", a.updateConnectorSecret)
}

func configuredSecretBox() (*security.SecretBox, error) {
	return security.NewSecretBox(os.Getenv("DEVRELOS_SECRET_KEY"))
}

func (a *api) listSecrets(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := a.requestWorkspaceID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if !a.requireWorkspaceRole(w, r, workspaceID, "admin") {
		return
	}
	items, err := a.store.ListConnectorSecrets(r.Context(), workspaceID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createSecret(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := a.requestWorkspaceID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if !a.requireWorkspaceRole(w, r, workspaceID, "admin") {
		return
	}
	var input struct {
		Provider string `json:"provider"`
		Name     string `json:"name"`
		Value    string `json:"value"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	input.Provider = strings.ToLower(strings.TrimSpace(input.Provider))
	input.Name = strings.TrimSpace(input.Name)
	input.Value = strings.TrimSpace(input.Value)
	if input.Provider == "" || input.Name == "" || input.Value == "" {
		writeBadRequest(w, "provider, name and value are required")
		return
	}
	box, err := configuredSecretBox()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "encrypted secret storage is not configured"})
		return
	}
	const version = 1
	ciphertext, nonce, err := box.Encrypt([]byte(input.Value), security.AssociatedData(workspaceID, input.Provider, input.Name, version))
	if err != nil {
		writeError(w, err)
		return
	}
	act := currentActor(r)
	item, err := a.store.CreateConnectorSecret(r.Context(), connectordomain.EncryptedSecret{
		Secret: connectordomain.Secret{
			WorkspaceID: workspaceID, Provider: input.Provider, Name: input.Name,
			KeyVersion: version, CreatedByUserID: act.UserID,
		},
		Ciphertext: ciphertext, Nonce: nonce,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	_ = a.store.AppendAuditEvent(r.Context(), identitydomain.AuditEvent{
		WorkspaceID: workspaceID, ActorUserID: act.UserID, ActorKind: auditActorKind(act),
		Action: "secret.created", ResourceType: "connector_secret", ResourceID: item.ID,
		Metadata: map[string]any{"provider": item.Provider, "name": item.Name, "keyVersion": item.KeyVersion},
	})
	writeJSON(w, http.StatusCreated, item)
}

func (a *api) rotateSecret(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := a.requestWorkspaceID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if !a.requireWorkspaceRole(w, r, workspaceID, "admin") {
		return
	}
	var input struct {
		Value string `json:"value"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	input.Value = strings.TrimSpace(input.Value)
	if input.Value == "" {
		writeBadRequest(w, "value is required")
		return
	}
	current, err := a.store.GetEncryptedConnectorSecret(r.Context(), workspaceID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	box, err := configuredSecretBox()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "encrypted secret storage is not configured"})
		return
	}
	version := current.KeyVersion + 1
	ciphertext, nonce, err := box.Encrypt([]byte(input.Value), security.AssociatedData(workspaceID, current.Provider, current.Name, version))
	if err != nil {
		writeError(w, err)
		return
	}
	item, err := a.store.RotateConnectorSecret(r.Context(), workspaceID, current.ID, ciphertext, nonce, version)
	if err != nil {
		writeError(w, err)
		return
	}
	act := currentActor(r)
	_ = a.store.AppendAuditEvent(r.Context(), identitydomain.AuditEvent{
		WorkspaceID: workspaceID, ActorUserID: act.UserID, ActorKind: auditActorKind(act),
		Action: "secret.rotated", ResourceType: "connector_secret", ResourceID: item.ID,
		Metadata: map[string]any{"provider": item.Provider, "name": item.Name, "keyVersion": item.KeyVersion},
	})
	writeJSON(w, http.StatusOK, item)
}

func (a *api) deleteSecret(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := a.requestWorkspaceID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if !a.requireWorkspaceRole(w, r, workspaceID, "admin") {
		return
	}
	secretID := r.PathValue("id")
	if err := a.store.DeleteConnectorSecret(r.Context(), workspaceID, secretID); err != nil {
		if errors.Is(err, storage.ErrSecretInUse) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "secret is attached to a connector"})
			return
		}
		writeError(w, err)
		return
	}
	act := currentActor(r)
	_ = a.store.AppendAuditEvent(r.Context(), identitydomain.AuditEvent{
		WorkspaceID: workspaceID, ActorUserID: act.UserID, ActorKind: auditActorKind(act),
		Action: "secret.deleted", ResourceType: "connector_secret", ResourceID: secretID,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) updateConnectorSecret(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := a.requestWorkspaceID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if !a.requireWorkspaceRole(w, r, workspaceID, "admin") {
		return
	}
	var input struct {
		SecretID string `json:"secretId"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	input.SecretID = strings.TrimSpace(input.SecretID)
	if input.SecretID != "" {
		connectors, listErr := a.store.ListConnectors(r.Context(), workspaceID)
		if listErr != nil {
			writeError(w, listErr)
			return
		}
		provider := ""
		for _, connector := range connectors {
			if connector.ID == r.PathValue("id") {
				provider = connector.Provider
				break
			}
		}
		if provider == "" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "connector not found"})
			return
		}
		belongs, belongsErr := a.store.ConnectorSecretBelongs(r.Context(), workspaceID, input.SecretID, provider)
		if belongsErr != nil {
			writeError(w, belongsErr)
			return
		}
		if !belongs {
			writeBadRequest(w, "secret does not belong to this workspace/provider")
			return
		}
	}
	if err := a.store.UpdateConnectorSecret(r.Context(), workspaceID, r.PathValue("id"), input.SecretID); err != nil {
		writeError(w, err)
		return
	}
	act := currentActor(r)
	_ = a.store.AppendAuditEvent(r.Context(), identitydomain.AuditEvent{
		WorkspaceID: workspaceID, ActorUserID: act.UserID, ActorKind: auditActorKind(act),
		Action: "connector.secret_updated", ResourceType: "connector", ResourceID: r.PathValue("id"),
		Metadata: map[string]any{"secretId": input.SecretID},
	})
	writeJSON(w, http.StatusOK, map[string]string{"secretId": input.SecretID})
}
