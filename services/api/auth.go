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
}

func (a *api) withAuth(next http.Handler) http.Handler {
	operatorToken := strings.TrimSpace(os.Getenv("DEVRELOS_API_TOKEN"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
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

		if strings.HasPrefix(token, "drk_") {
			hash := sha256.Sum256([]byte(token))
			principal, err := a.store.ResolveAPIKey(r.Context(), hex.EncodeToString(hash[:]))
			if err == nil && principal.Status == "active" {
				workspaceID, scopeErr := a.requestWorkspaceID(r)
				if scopeErr != nil {
					writeUnauthorized(w)
					return
				}
				role, roleErr := a.store.RoleForWorkspace(r.Context(), principal.UserID, workspaceID)
				if roleErr != nil || roleRank(role) == 0 {
					writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
					return
				}
				if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions && roleRank(role) < roleRank("editor") {
					writeJSON(w, http.StatusForbidden, map[string]string{"error": "editor role required"})
					return
				}
				ctx := context.WithValue(r.Context(), actorContextKey{}, actor{
					Kind: "user", UserID: principal.UserID, Email: principal.Email,
					DisplayName: principal.DisplayName, APIKeyID: principal.APIKeyID,
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		writeUnauthorized(w)
	})
}

func (a *api) requestWorkspaceID(r *http.Request) (string, error) {
	if workspaceID := strings.TrimSpace(r.URL.Query().Get("workspaceId")); workspaceID != "" {
		return workspaceID, nil
	}
	if projectID := strings.TrimSpace(r.URL.Query().Get("projectId")); projectID != "" {
		return a.store.ProjectWorkspaceID(r.Context(), projectID)
	}
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
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="devrelos-api"`)
	writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
}
