package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	domain "github.com/savisaluwadana/DevRelOS/internal/domain/identity"
)

func (a *api) registerIdentityRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/identity/me", a.identityMe)
	mux.HandleFunc("GET /api/v1/identity/users", a.listIdentityUsers)
	mux.HandleFunc("POST /api/v1/identity/users", a.createIdentityUser)
	mux.HandleFunc("GET /api/v1/identity/memberships", a.listIdentityMemberships)
	mux.HandleFunc("PUT /api/v1/identity/memberships", a.upsertIdentityMembership)
	mux.HandleFunc("GET /api/v1/identity/users/{id}/api-keys", a.listIdentityAPIKeys)
	mux.HandleFunc("POST /api/v1/identity/users/{id}/api-keys", a.createIdentityAPIKey)
	mux.HandleFunc("POST /api/v1/identity/users/{id}/api-keys/{keyId}/revoke", a.revokeIdentityAPIKey)
	mux.HandleFunc("GET /api/v1/audit-events", a.listAuditEvents)
}

func (a *api) identityMe(w http.ResponseWriter, r *http.Request) {
	act := currentActor(r)
	workspaceID, err := a.requestWorkspaceID(r)
	if err != nil { writeError(w, err); return }
	role := "owner"
	if act.Kind == "user" {
		role, err = a.store.RoleForWorkspace(r.Context(), act.UserID, workspaceID)
		if err != nil { writeForbidden(w); return }
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"kind": act.Kind, "userId": act.UserID, "email": act.Email,
		"displayName": act.DisplayName, "workspaceId": workspaceID, "role": role,
	})
}

func (a *api) listIdentityUsers(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := a.requestWorkspaceID(r)
	if err != nil { writeError(w, err); return }
	if !a.requireWorkspaceRole(w, r, workspaceID, "admin") { return }
	items, err := a.store.ListWorkspaceUsers(r.Context(), workspaceID)
	if err != nil { writeError(w, err); return }
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createIdentityUser(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := a.requestWorkspaceID(r)
	if err != nil { writeError(w, err); return }
	if !a.requireWorkspaceRole(w, r, workspaceID, "admin") { return }

	var input struct {
		Email       string `json:"email"`
		DisplayName string `json:"displayName"`
		Status      string `json:"status"`
		Role        string `json:"role"`
	}
	if err := decodeJSON(r, &input); err != nil { writeBadRequest(w, err.Error()); return }
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.Email == "" || !strings.Contains(input.Email, "@") { writeBadRequest(w, "valid email is required"); return }
	if input.Status == "" { input.Status = "active" }
	if input.Status != "active" && input.Status != "disabled" { writeBadRequest(w, "status must be active or disabled"); return }
	if input.Role == "" { input.Role = "viewer" }
	if roleRank(input.Role) == 0 { writeBadRequest(w, "role must be owner, admin, editor or viewer"); return }
	if currentActor(r).Kind == "user" && input.Role == "owner" {
		currentRole, roleErr := a.store.RoleForWorkspace(r.Context(), currentActor(r).UserID, workspaceID)
		if roleErr != nil || currentRole != "owner" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only an owner can grant owner role"})
			return
		}
	}

	created, err := a.store.CreateWorkspaceUser(r.Context(), workspaceID, domain.User{
		Email: input.Email, DisplayName: input.DisplayName, Status: input.Status,
	}, input.Role)
	if err != nil { writeError(w, err); return }
	act := currentActor(r)
	_ = a.store.AppendAuditEvent(r.Context(), domain.AuditEvent{
		WorkspaceID: workspaceID, ActorUserID: act.UserID, ActorKind: auditActorKind(act),
		Action: "identity.user_provisioned", ResourceType: "user", ResourceID: created.ID,
		Metadata: map[string]any{"email": created.Email, "role": input.Role},
	})
	writeJSON(w, http.StatusCreated, created)
}

func (a *api) listIdentityMemberships(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := a.requestWorkspaceID(r)
	if err != nil { writeError(w, err); return }
	if !a.requireWorkspaceRole(w, r, workspaceID, "viewer") { return }
	items, err := a.store.ListMemberships(r.Context(), workspaceID)
	if err != nil { writeError(w, err); return }
	writeJSON(w, http.StatusOK, items)
}

func (a *api) upsertIdentityMembership(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := a.requestWorkspaceID(r)
	if err != nil { writeError(w, err); return }
	if !a.requireWorkspaceRole(w, r, workspaceID, "admin") { return }
	var input domain.Membership
	if err := decodeJSON(r, &input); err != nil { writeBadRequest(w, err.Error()); return }
	input.WorkspaceID = workspaceID
	if strings.TrimSpace(input.UserID) == "" { writeBadRequest(w, "userId is required"); return }
	if roleRank(input.Role) == 0 { writeBadRequest(w, "role must be owner, admin, editor or viewer"); return }
	if currentActor(r).Kind == "user" && input.Role == "owner" {
		currentRole, roleErr := a.store.RoleForWorkspace(r.Context(), currentActor(r).UserID, workspaceID)
		if roleErr != nil || currentRole != "owner" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only an owner can grant owner role"})
			return
		}
	}
	created, err := a.store.UpsertMembership(r.Context(), input)
	if err != nil { writeError(w, err); return }
	act := currentActor(r)
	_ = a.store.AppendAuditEvent(r.Context(), domain.AuditEvent{
		WorkspaceID: workspaceID, ActorUserID: act.UserID, ActorKind: auditActorKind(act),
		Action: "identity.membership_upserted", ResourceType: "workspace_membership", ResourceID: created.UserID,
		Metadata: map[string]any{"role": created.Role},
	})
	writeJSON(w, http.StatusOK, created)
}

func (a *api) listIdentityAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if !a.canManageUserKeys(w, r, userID) { return }
	items, err := a.store.ListAPIKeys(r.Context(), userID)
	if err != nil { writeError(w, err); return }
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createIdentityAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if !a.canManageUserKeys(w, r, userID) { return }
	var input struct {
		Name string `json:"name"`
		ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	}
	if err := decodeJSON(r, &input); err != nil { writeBadRequest(w, err.Error()); return }
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" { writeBadRequest(w, "name is required"); return }
	if input.ExpiresAt != nil && input.ExpiresAt.Before(time.Now().UTC()) { writeBadRequest(w, "expiresAt must be in the future"); return }
	plain, prefix, secretHash, err := generateAPIKey()
	if err != nil { writeError(w, err); return }
	item, err := a.store.CreateAPIKey(r.Context(), userID, input.Name, prefix, secretHash, input.ExpiresAt)
	if err != nil { writeError(w, err); return }
	act := currentActor(r)
	workspaceID, _ := a.requestWorkspaceID(r)
	_ = a.store.AppendAuditEvent(r.Context(), domain.AuditEvent{
		WorkspaceID: workspaceID, ActorUserID: act.UserID, ActorKind: auditActorKind(act),
		Action: "identity.api_key_created", ResourceType: "api_key", ResourceID: item.ID,
		Metadata: map[string]any{"userId": userID, "name": item.Name, "prefix": item.KeyPrefix},
	})
	writeJSON(w, http.StatusCreated, map[string]any{"apiKey": item, "token": plain})
}

func (a *api) revokeIdentityAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if !a.canManageUserKeys(w, r, userID) { return }
	if err := a.store.RevokeAPIKey(r.Context(), userID, r.PathValue("keyId")); err != nil { writeError(w, err); return }
	act := currentActor(r)
	workspaceID, _ := a.requestWorkspaceID(r)
	_ = a.store.AppendAuditEvent(r.Context(), domain.AuditEvent{
		WorkspaceID: workspaceID, ActorUserID: act.UserID, ActorKind: auditActorKind(act),
		Action: "identity.api_key_revoked", ResourceType: "api_key", ResourceID: r.PathValue("keyId"),
		Metadata: map[string]any{"userId": userID},
	})
	writeJSON(w, http.StatusOK, map[string]bool{"revoked": true})
}

func (a *api) listAuditEvents(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := a.requestWorkspaceID(r)
	if err != nil { writeError(w, err); return }
	if !a.requireWorkspaceRole(w, r, workspaceID, "admin") { return }
	items, err := a.store.ListAuditEvents(r.Context(), workspaceID, intQuery(r, "limit", 100))
	if err != nil { writeError(w, err); return }
	writeJSON(w, http.StatusOK, items)
}

func (a *api) requireWorkspaceRole(w http.ResponseWriter, r *http.Request, workspaceID, minimum string) bool {
	act := currentActor(r)
	if act.Kind == "operator" { return true }
	if act.Kind != "user" { writeForbidden(w); return false }
	role, err := a.store.RoleForWorkspace(r.Context(), act.UserID, workspaceID)
	if err != nil || roleRank(role) < roleRank(minimum) { writeForbidden(w); return false }
	return true
}

func (a *api) canManageUserKeys(w http.ResponseWriter, r *http.Request, userID string) bool {
	act := currentActor(r)
	if act.Kind == "operator" || (act.Kind == "user" && act.UserID == userID) { return true }
	workspaceID, err := a.requestWorkspaceID(r)
	if err != nil || !a.requireWorkspaceRole(w, r, workspaceID, "admin") { return false }
	return true
}

func generateAPIKey() (plain, prefix, secretHash string, err error) {
	secret := make([]byte, 32)
	if _, err = rand.Read(secret); err != nil { return "", "", "", err }
	plain = "drk_" + base64.RawURLEncoding.EncodeToString(secret)
	prefix = plain
	if len(prefix) > 12 { prefix = prefix[:12] }
	hash := sha256.Sum256([]byte(plain))
	secretHash = hex.EncodeToString(hash[:])
	return plain, prefix, secretHash, nil
}

func auditActorKind(act actor) string {
	if act.Kind == "user" { return "user" }
	return "operator"
}

func writeForbidden(w http.ResponseWriter) {
	writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
}
