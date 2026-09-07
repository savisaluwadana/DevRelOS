package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"strings"

	identitydomain "github.com/savisaluwadana/DevRelOS/internal/domain/identity"
)

type authContextKey string

const (
	principalContextKey authContextKey = "devrelos-principal"
	sessionCookieName                    = "devrelos_session"
)

var (
	errForbidden    = errors.New("forbidden")
	errUnauthorized = errors.New("unauthorized")
)

func (a *api) withAuth(next http.Handler) http.Handler {
	operatorToken := strings.TrimSpace(os.Getenv("DEVRELOS_API_TOKEN"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if publicAuthPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		if operatorToken == "" && !strings.EqualFold(strings.TrimSpace(os.Getenv("DEVRELOS_REQUIRE_AUTH")), "true") {
			next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), identitydomain.Principal{System: true})))
			return
		}

		rawToken := bearerToken(r)
		if rawToken != "" && operatorToken != "" && secureEqual(rawToken, operatorToken) {
			next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), identitydomain.Principal{System: true})))
			return
		}

		if rawToken == "" {
			if cookie, err := r.Cookie(sessionCookieName); err == nil {
				rawToken = strings.TrimSpace(cookie.Value)
			}
		}
		if rawToken != "" {
			principal, err := a.sessionPrincipal(r.Context(), rawToken)
			if err == nil {
				next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), principal)))
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Bearer realm="devrelos-api"`)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	})
}

func publicAuthPath(path string) bool {
	return path == "/healthz" || path == "/readyz" || path == "/api/v1/auth/login"
}

func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (a *api) sessionPrincipal(ctx context.Context, rawToken string) (identitydomain.Principal, error) {
	user, err := a.store.UserBySessionHash(ctx, hashSessionToken(rawToken))
	if err != nil {
		return identitydomain.Principal{}, err
	}
	memberships, err := a.store.ListMembershipsForUser(ctx, user.ID)
	if err != nil {
		return identitydomain.Principal{}, err
	}
	user.PasswordHash = ""
	return identitydomain.Principal{User: &user, Memberships: memberships}, nil
}

func withPrincipal(ctx context.Context, principal identitydomain.Principal) context.Context {
	return context.WithValue(ctx, principalContextKey, principal)
}

func principalFromContext(ctx context.Context) (identitydomain.Principal, bool) {
	principal, ok := ctx.Value(principalContextKey).(identitydomain.Principal)
	return principal, ok
}

func roleForMethod(method string) string {
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		return "viewer"
	}
	return "editor"
}

func roleRank(role string) int {
	switch role {
	case "owner":
		return 4
	case "admin":
		return 3
	case "editor":
		return 2
	case "viewer":
		return 1
	default:
		return 0
	}
}

func (a *api) requireWorkspaceRole(ctx context.Context, workspaceID, required string) error {
	principal, ok := principalFromContext(ctx)
	if !ok {
		return errUnauthorized
	}
	if principal.System {
		return nil
	}
	if principal.User == nil {
		return errUnauthorized
	}
	role, err := a.store.MembershipRole(ctx, principal.User.ID, workspaceID)
	if err != nil || roleRank(role) < roleRank(required) {
		return errForbidden
	}
	return nil
}

func (a *api) requireProjectRole(ctx context.Context, projectID, required string) error {
	workspaceID, err := a.store.ProjectWorkspaceID(ctx, projectID)
	if err != nil {
		return err
	}
	return a.requireWorkspaceRole(ctx, workspaceID, required)
}

func secureEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
