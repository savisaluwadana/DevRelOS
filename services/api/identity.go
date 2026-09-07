package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	identitydomain "github.com/savisaluwadana/DevRelOS/internal/domain/identity"
	"github.com/savisaluwadana/DevRelOS/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

func (a *api) registerIdentityRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/login", a.login)
	mux.HandleFunc("POST /api/v1/auth/logout", a.logout)
	mux.HandleFunc("GET /api/v1/auth/me", a.me)
	mux.HandleFunc("GET /api/v1/members", a.listMembers)
	mux.HandleFunc("POST /api/v1/members", a.createMember)
	mux.HandleFunc("PATCH /api/v1/members/{userId}/role", a.updateMemberRole)
}

func (a *api) bootstrapIdentity(ctx context.Context) error {
	email := strings.TrimSpace(os.Getenv("DEVRELOS_BOOTSTRAP_EMAIL"))
	password := os.Getenv("DEVRELOS_BOOTSTRAP_PASSWORD")
	name := strings.TrimSpace(os.Getenv("DEVRELOS_BOOTSTRAP_NAME"))
	if email == "" && password == "" {
		return nil
	}
	if email == "" || password == "" {
		return errors.New("DEVRELOS_BOOTSTRAP_EMAIL and DEVRELOS_BOOTSTRAP_PASSWORD must be configured together")
	}
	if len(password) < 12 {
		return errors.New("DEVRELOS_BOOTSTRAP_PASSWORD must be at least 12 characters")
	}
	workspaceID, err := a.store.DefaultWorkspaceID(ctx)
	if err != nil {
		return fmt.Errorf("resolve default workspace: %w", err)
	}
	user, err := a.store.UserByEmail(ctx, email)
	if err == nil {
		return a.store.EnsureMembership(ctx, workspaceID, user.ID, "owner")
	}
	if !storage.IsNotFound(err) {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if name == "" {
		name = email
	}
	_, err = a.store.CreateUserWithMembership(ctx, identitydomain.User{
		Email: email, DisplayName: name, PasswordHash: string(hash),
	}, workspaceID, "owner")
	return err
}

func (a *api) login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	input.Email = strings.TrimSpace(input.Email)
	if input.Email == "" || input.Password == "" {
		writeBadRequest(w, "email and password are required")
		return
	}

	user, err := a.store.UserByEmail(r.Context(), input.Email)
	if err != nil || user.Status != "active" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}

	rawToken, err := newSessionToken()
	if err != nil {
		writeError(w, err)
		return
	}
	expiresAt := time.Now().UTC().Add(sessionTTL())
	if _, err := a.store.CreateSession(r.Context(), user.ID, hashSessionToken(rawToken), expiresAt); err != nil {
		writeError(w, err)
		return
	}
	setSessionCookie(w, rawToken, expiresAt)
	user.PasswordHash = ""
	memberships, err := a.store.ListMembershipsForUser(r.Context(), user.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, identitydomain.Principal{User: &user, Memberships: memberships})
}

func (a *api) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
		_ = a.store.RevokeSession(r.Context(), hashSessionToken(strings.TrimSpace(cookie.Value)))
	}
	clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *api) me(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, principal)
}

func (a *api) listMembers(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := a.workspaceID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := a.requireWorkspaceRole(r.Context(), workspaceID, "admin"); err != nil {
		writeError(w, err)
		return
	}
	items, err := a.store.ListWorkspaceMembers(r.Context(), workspaceID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *api) createMember(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email       string `json:"email"`
		DisplayName string `json:"displayName"`
		Password    string `json:"password"`
		Role        string `json:"role"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	input.Email = strings.TrimSpace(input.Email)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Role = strings.TrimSpace(input.Role)
	if input.Email == "" {
		writeBadRequest(w, "email is required")
		return
	}
	if input.Role == "" {
		input.Role = "editor"
	}
	if !validMembershipRole(input.Role) {
		writeBadRequest(w, "role must be owner, admin, editor or viewer")
		return
	}

	workspaceID, err := a.workspaceID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	required := "admin"
	if input.Role == "owner" || input.Role == "admin" {
		required = "owner"
	}
	if err := a.requireWorkspaceRole(r.Context(), workspaceID, required); err != nil {
		writeError(w, err)
		return
	}

	user, err := a.store.UserByEmail(r.Context(), input.Email)
	if err == nil {
		if _, roleErr := a.store.MembershipRole(r.Context(), user.ID, workspaceID); roleErr == nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "user is already a workspace member"})
			return
		} else if !storage.IsNotFound(roleErr) {
			writeError(w, roleErr)
			return
		}
		if err := a.store.EnsureMembership(r.Context(), workspaceID, user.ID, input.Role); err != nil {
			writeError(w, err)
			return
		}
	} else {
		if !storage.IsNotFound(err) {
			writeError(w, err)
			return
		}
		if len(input.Password) < 12 {
			writeBadRequest(w, "password must be at least 12 characters for a new user")
			return
		}
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if hashErr != nil {
			writeError(w, hashErr)
			return
		}
		if input.DisplayName == "" {
			input.DisplayName = input.Email
		}
		user, err = a.store.CreateUserWithMembership(r.Context(), identitydomain.User{
			Email: input.Email, DisplayName: input.DisplayName, PasswordHash: string(hash),
		}, workspaceID, input.Role)
		if err != nil {
			writeError(w, err)
			return
		}
	}

	user.PasswordHash = ""
	writeJSON(w, http.StatusCreated, map[string]any{"user": user, "role": input.Role, "workspaceId": workspaceID})
}

func (a *api) updateMemberRole(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Role string `json:"role"`
	}
	if err := decodeJSON(r, &input); err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	input.Role = strings.TrimSpace(input.Role)
	if !validMembershipRole(input.Role) {
		writeBadRequest(w, "role must be owner, admin, editor or viewer")
		return
	}
	workspaceID, err := a.workspaceID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := a.requireWorkspaceRole(r.Context(), workspaceID, "owner"); err != nil {
		writeError(w, err)
		return
	}

	userID := r.PathValue("userId")
	current, err := a.store.MembershipRole(r.Context(), userID, workspaceID)
	if err != nil {
		writeError(w, err)
		return
	}
	if current == "owner" && input.Role != "owner" {
		owners, countErr := a.store.CountWorkspaceOwners(r.Context(), workspaceID)
		if countErr != nil {
			writeError(w, countErr)
			return
		}
		if owners <= 1 {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "cannot demote the last workspace owner"})
			return
		}
	}
	if err := a.store.UpdateMembershipRole(r.Context(), workspaceID, userID, input.Role); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"userId": userID, "role": input.Role})
}

func validMembershipRole(role string) bool {
	return role == "owner" || role == "admin" || role == "editor" || role == "viewer"
}

func newSessionToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func sessionTTL() time.Duration {
	value := strings.TrimSpace(os.Getenv("DEVRELOS_SESSION_TTL"))
	if value == "" {
		return 12 * time.Hour
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed < 15*time.Minute || parsed > 30*24*time.Hour {
		return 12 * time.Hour
	}
	return parsed
}

func setSessionCookie(w http.ResponseWriter, value string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: value, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: secureCookies(), Expires: expiresAt,
		MaxAge: int(time.Until(expiresAt).Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: "", Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: secureCookies(), Expires: time.Unix(0, 0), MaxAge: -1,
	})
}

func secureCookies() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("DEVRELOS_SECURE_COOKIES")), "true")
}
