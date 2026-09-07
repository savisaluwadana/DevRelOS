package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"os"
	"strings"

	identitydomain "github.com/savisaluwadana/DevRelOS/internal/domain/identity"
)

type actorContextKey struct{}

type actor struct {
	Kind        string
	UserID      string
	Email       string
	DisplayName string
	APIKeyID    string
	SessionID   string
	WorkspaceID string
}

func (a *api) withAuth(next http.Handler) http.Handler {
	operatorToken := strings.TrimSpace(os.Getenv("DEVRELOS_API_TOKEN"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" || r.URL.Path == "/metrics" ||
			(r.Method == http.MethodPost && r.URL.Path == "/api/v1/identity/invitations/accept") {
			next.ServeHTTP(w, r)
			return
		}

		if operatorToken == "" && !strings.EqualFold(strings.TrimSpace(os.Getenv("DEVRELOS_REQUIRE_AUTH")), "true") {
			ctx := context.WithValue(r.Context(), actorContextKey{}, actor{Kind: "operator"})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		const prefix = "Bearer "
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(header, prefix) {
			writeUnauthorized(w)
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
		if token == "" {
			writeUnauthorized(w)
			return
		}

		if operatorToken != "" && secureEqual(token, operatorToken) {
			ctx := context.WithValue(r.Context(), actorContextKey{}, actor{Kind: "operator"})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		if strings.HasPrefix(token, "ds_") {
			hash := sha256.Sum256([]byte(token))
			principal, err := a.store.ResolveSession(r.Context(), hex.EncodeToString(hash[:]))
			if err == nil && principal.Status == "active" {
				if !a.sessionRequestWithinWorkspace(w, r, principal.WorkspaceID) { return }
				role, roleErr := a.store.RoleForWorkspace(r.Context(), principal.UserID, principal.WorkspaceID)
				if roleErr != nil || roleRank(role) == 0 { writeForbidden(w); return }
				if requiresEditor(r) && roleRank(role) < roleRank("editor") {
					writeJSON(w, http.StatusForbidden, map[string]string{"error": "editor role required"})
					return
				}
				ctx := context.WithValue(r.Context(), actorContextKey{}, actor{
					Kind: "user", UserID: principal.UserID, Email: principal.Email, DisplayName: principal.DisplayName,
					SessionID: principal.SessionID, WorkspaceID: principal.WorkspaceID,
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		if strings.HasPrefix(token, "drk_") {
			hash := sha256.Sum256([]byte(token))
			principal, err := a.store.ResolveAPIKey(r.Context(), hex.EncodeToString(hash[:]))
			if err == nil && principal.Status == "active" {
				workspaceID, scopeErr := a.apiKeyWorkspace(r, principal.UserID)
				if scopeErr != nil { writeForbidden(w); return }
				role, roleErr := a.store.RoleForWorkspace(r.Context(), principal.UserID, workspaceID)
				if roleErr != nil || roleRank(role) == 0 { writeForbidden(w); return }
				if requiresEditor(r) && roleRank(role) < roleRank("editor") {
					writeJSON(w, http.StatusForbidden, map[string]string{"error": "editor role required"})
					return
				}
				ctx := context.WithValue(r.Context(), actorContextKey{}, actor{
					Kind: "user", UserID: principal.UserID, Email: principal.Email,
					DisplayName: principal.DisplayName, APIKeyID: principal.APIKeyID, WorkspaceID: workspaceID,
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		writeUnauthorized(w)
	})
}

func (a *api) apiKeyWorkspace(r *http.Request, userID string) (string, error) {
	if workspaceID := strings.TrimSpace(r.URL.Query().Get("workspaceId")); workspaceID != "" { return workspaceID, nil }
	if projectID := strings.TrimSpace(r.URL.Query().Get("projectId")); projectID != "" { return a.store.ProjectWorkspaceID(r.Context(), projectID) }
	items, err := a.store.ListUserWorkspaces(r.Context(), userID)
	if err != nil { return "", err }
	if len(items) == 0 { return "", os.ErrNotExist }
	return items[0].WorkspaceID, nil
}

func isMutation(method string) bool {
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}

func requiresEditor(r *http.Request) bool {
	if !isMutation(r.Method) { return false }
	// Identity endpoints have finer-grained self-service/admin/owner checks in their handlers.
	if strings.HasPrefix(r.URL.Path, "/api/v1/identity/") { return false }
	return true
}

func (a *api) sessionRequestWithinWorkspace(w http.ResponseWriter, r *http.Request, sessionWorkspaceID string) bool {
	if requested := strings.TrimSpace(r.URL.Query().Get("workspaceId")); requested != "" && requested != sessionWorkspaceID {
		writeForbidden(w)
		return false
	}
	if projectID := strings.TrimSpace(r.URL.Query().Get("projectId")); projectID != "" {
		workspaceID, err := a.store.ProjectWorkspaceID(r.Context(), projectID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
			return false
		}
		if workspaceID != sessionWorkspaceID {
			writeForbidden(w)
			return false
		}
	}
	return true
}

func (a *api) requestWorkspaceID(r *http.Request) (string, error) {
	act := currentActor(r)
	if act.SessionID != "" && act.WorkspaceID != "" { return act.WorkspaceID, nil }
	if workspaceID := strings.TrimSpace(r.URL.Query().Get("workspaceId")); workspaceID != "" { return workspaceID, nil }
	if projectID := strings.TrimSpace(r.URL.Query().Get("projectId")); projectID != "" { return a.store.ProjectWorkspaceID(r.Context(), projectID) }
	if act.WorkspaceID != "" { return act.WorkspaceID, nil }
	return a.store.DefaultWorkspaceID(r.Context())
}

func currentActor(r *http.Request) actor {
	value, _ := r.Context().Value(actorContextKey{}).(actor)
	return value
}

func principalToActor(p identitydomain.Principal) actor {
	return actor{Kind: "user", UserID: p.UserID, Email: p.Email, DisplayName: p.DisplayName, APIKeyID: p.APIKeyID}
}

func roleRank(role string) int {
	return map[string]int{"viewer": 1, "editor": 2, "admin": 3, "owner": 4}[role]
}

func secureEqual(a, b string) bool {
	if len(a) != len(b) { return false }
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="devrelos-api"`)
	writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
}
