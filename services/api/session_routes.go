package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	identitydomain "github.com/savisaluwadana/DevRelOS/internal/domain/identity"
)

func (a *api) registerSessionRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/identity/workspaces", a.listMyWorkspaces)
	mux.HandleFunc("POST /api/v1/identity/sessions", a.createBrowserSession)
	mux.HandleFunc("PUT /api/v1/identity/sessions/current/workspace", a.switchBrowserSessionWorkspace)
	mux.HandleFunc("DELETE /api/v1/identity/sessions/current", a.revokeBrowserSession)
	mux.HandleFunc("GET /api/v1/identity/invitations", a.listInvitations)
	mux.HandleFunc("POST /api/v1/identity/invitations", a.createInvitation)
	mux.HandleFunc("DELETE /api/v1/identity/invitations/{id}", a.revokeInvitation)
	mux.HandleFunc("POST /api/v1/identity/invitations/accept", a.acceptInvitation)
}

func sessionLifetime() time.Duration {
	seconds := 28800
	if raw := strings.TrimSpace(os.Getenv("DEVRELOS_SESSION_MAX_AGE_SECONDS")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			seconds = parsed
		}
	}
	if seconds < 900 {
		seconds = 900
	}
	if seconds > 86400 {
		seconds = 86400
	}
	return time.Duration(seconds) * time.Second
}

func generateOpaqueToken(prefix string) (plain, hash string, err error) {
	secret := make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		return "", "", err
	}
	plain = prefix + base64.RawURLEncoding.EncodeToString(secret)
	digest := sha256.Sum256([]byte(plain))
	return plain, hex.EncodeToString(digest[:]), nil
}

func (a *api) listMyWorkspaces(w http.ResponseWriter, r *http.Request) {
	act := currentActor(r)
	if act.Kind != "user" {
		writeForbidden(w)
		return
	}
	items, err := a.store.ListUserWorkspaces(r.Context(), act.UserID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createBrowserSession(w http.ResponseWriter, r *http.Request) {
	act := currentActor(r)
	if act.Kind != "user" {
		writeForbidden(w)
		return
	}
	var input struct {
		WorkspaceID string `json:"workspaceId"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	input.WorkspaceID = strings.TrimSpace(input.WorkspaceID)
	if input.WorkspaceID == "" {
		if act.WorkspaceID != "" {
			input.WorkspaceID = act.WorkspaceID
		} else {
			items, err := a.store.ListUserWorkspaces(r.Context(), act.UserID)
			if err != nil {
				writeError(w, err)
				return
			}
			if len(items) == 0 {
				writeForbidden(w)
				return
			}
			input.WorkspaceID = items[0].WorkspaceID
		}
	}
	if _, err := a.store.RoleForWorkspace(r.Context(), act.UserID, input.WorkspaceID); err != nil {
		writeForbidden(w)
		return
	}
	plain, tokenHash, err := generateOpaqueToken("ds_")
	if err != nil {
		writeError(w, err)
		return
	}
	expiresAt := time.Now().UTC().Add(sessionLifetime())
	session, err := a.store.CreateSession(r.Context(), act.UserID, input.WorkspaceID, tokenHash, r.UserAgent(), expiresAt)
	if err != nil {
		writeError(w, err)
		return
	}
	role, _ := a.store.RoleForWorkspace(r.Context(), act.UserID, input.WorkspaceID)
	_ = a.store.AppendAuditEvent(r.Context(), identitydomain.AuditEvent{
		WorkspaceID: input.WorkspaceID, ActorUserID: act.UserID, ActorKind: "user",
		Action: "session.created", ResourceType: "user_session", ResourceID: session.ID,
		Metadata: map[string]any{"role": role, "expiresAt": expiresAt},
	})
	writeJSON(w, http.StatusCreated, map[string]any{
		"sessionToken": plain,
		"expiresAt":    expiresAt,
		"principal": map[string]any{
			"kind": "user", "userId": act.UserID, "email": act.Email, "displayName": act.DisplayName,
			"workspaceId": input.WorkspaceID, "role": role,
		},
	})
}

func (a *api) switchBrowserSessionWorkspace(w http.ResponseWriter, r *http.Request) {
	act := currentActor(r)
	if act.Kind != "user" || act.SessionID == "" {
		writeBadRequest(w, "dedicated browser session required")
		return
	}
	var input struct {
		WorkspaceID string `json:"workspaceId"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	input.WorkspaceID = strings.TrimSpace(input.WorkspaceID)
	if input.WorkspaceID == "" {
		writeBadRequest(w, "workspaceId is required")
		return
	}
	if err := a.store.SwitchSessionWorkspace(r.Context(), act.SessionID, act.UserID, input.WorkspaceID); err != nil {
		writeForbidden(w)
		return
	}
	_ = a.store.AppendAuditEvent(r.Context(), identitydomain.AuditEvent{
		WorkspaceID: input.WorkspaceID, ActorUserID: act.UserID, ActorKind: "user",
		Action: "session.workspace_switched", ResourceType: "user_session", ResourceID: act.SessionID,
		Metadata: map[string]any{"fromWorkspaceId": act.WorkspaceID, "toWorkspaceId": input.WorkspaceID},
	})
	writeJSON(w, http.StatusOK, map[string]string{"workspaceId": input.WorkspaceID})
}

func (a *api) revokeBrowserSession(w http.ResponseWriter, r *http.Request) {
	act := currentActor(r)
	if act.Kind != "user" || act.SessionID == "" {
		writeBadRequest(w, "dedicated browser session required")
		return
	}
	if err := a.store.RevokeSession(r.Context(), act.SessionID, act.UserID); err != nil {
		writeError(w, err)
		return
	}
	_ = a.store.AppendAuditEvent(r.Context(), identitydomain.AuditEvent{
		WorkspaceID: act.WorkspaceID, ActorUserID: act.UserID, ActorKind: "user",
		Action: "session.revoked", ResourceType: "user_session", ResourceID: act.SessionID,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) listInvitations(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := a.requestWorkspaceID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if !a.requireWorkspaceRole(w, r, workspaceID, "admin") {
		return
	}
	items, err := a.store.ListInvitations(r.Context(), workspaceID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createInvitation(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := a.requestWorkspaceID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if !a.requireWorkspaceRole(w, r, workspaceID, "admin") {
		return
	}
	var input struct {
		Email        string `json:"email"`
		Role         string `json:"role"`
		ExpiresHours int    `json:"expiresHours"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Role = strings.TrimSpace(input.Role)
	if input.Email == "" || !strings.Contains(input.Email, "@") {
		writeBadRequest(w, "valid email is required")
		return
	}
	if roleRank(input.Role) == 0 {
		writeBadRequest(w, "role must be owner, admin, editor or viewer")
		return
	}
	act := currentActor(r)
	if act.Kind == "user" && input.Role == "owner" {
		role, roleErr := a.store.RoleForWorkspace(r.Context(), act.UserID, workspaceID)
		if roleErr != nil || role != "owner" {
			writeForbidden(w)
			return
		}
	}
	hours := input.ExpiresHours
	if hours <= 0 {
		hours = 168
	}
	if hours > 720 {
		hours = 720
	}
	plain, tokenHash, err := generateOpaqueToken("di_")
	if err != nil {
		writeError(w, err)
		return
	}
	item, err := a.store.CreateInvitation(r.Context(), identitydomain.Invitation{
		WorkspaceID: workspaceID, Email: input.Email, Role: input.Role,
		InvitedByUserID: act.UserID, ExpiresAt: time.Now().UTC().Add(time.Duration(hours) * time.Hour),
	}, tokenHash)
	if err != nil {
		writeError(w, err)
		return
	}
	_ = a.store.AppendAuditEvent(r.Context(), identitydomain.AuditEvent{
		WorkspaceID: workspaceID, ActorUserID: act.UserID, ActorKind: auditActorKind(act),
		Action: "invitation.created", ResourceType: "workspace_invitation", ResourceID: item.ID,
		Metadata: map[string]any{"email": item.Email, "role": item.Role, "expiresAt": item.ExpiresAt},
	})
	writeJSON(w, http.StatusCreated, map[string]any{"invitation": item, "inviteToken": plain})
}

func (a *api) revokeInvitation(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := a.requestWorkspaceID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if !a.requireWorkspaceRole(w, r, workspaceID, "admin") {
		return
	}
	if err := a.store.RevokeInvitation(r.Context(), workspaceID, r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	act := currentActor(r)
	_ = a.store.AppendAuditEvent(r.Context(), identitydomain.AuditEvent{
		WorkspaceID: workspaceID, ActorUserID: act.UserID, ActorKind: auditActorKind(act),
		Action: "invitation.revoked", ResourceType: "workspace_invitation", ResourceID: r.PathValue("id"),
	})
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) acceptInvitation(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token       string `json:"token"`
		DisplayName string `json:"displayName"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	input.Token = strings.TrimSpace(input.Token)
	if !strings.HasPrefix(input.Token, "di_") {
		writeBadRequest(w, "valid invitation token is required")
		return
	}
	digest := sha256.Sum256([]byte(input.Token))
	user, invitation, err := a.store.AcceptInvitation(r.Context(), hex.EncodeToString(digest[:]), strings.TrimSpace(input.DisplayName))
	if err != nil {
		writeUnauthorized(w)
		return
	}
	plain, sessionHash, err := generateOpaqueToken("ds_")
	if err != nil {
		writeError(w, err)
		return
	}
	expiresAt := time.Now().UTC().Add(sessionLifetime())
	session, err := a.store.CreateSession(r.Context(), user.ID, invitation.WorkspaceID, sessionHash, r.UserAgent(), expiresAt)
	if err != nil {
		writeError(w, err)
		return
	}
	_ = a.store.AppendAuditEvent(r.Context(), identitydomain.AuditEvent{
		WorkspaceID: invitation.WorkspaceID, ActorUserID: user.ID, ActorKind: "user",
		Action: "invitation.accepted", ResourceType: "workspace_invitation", ResourceID: invitation.ID,
		Metadata: map[string]any{"email": user.Email, "role": invitation.Role, "sessionId": session.ID},
	})
	writeJSON(w, http.StatusCreated, map[string]any{
		"sessionToken": plain, "expiresAt": expiresAt,
		"principal": map[string]any{"kind": "user", "userId": user.ID, "email": user.Email, "displayName": user.DisplayName, "workspaceId": invitation.WorkspaceID, "role": invitation.Role},
	})
}
