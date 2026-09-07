package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithAuthDisabledWithoutToken(t *testing.T) {
	t.Setenv("DEVRELOS_API_TOKEN", "")
	t.Setenv("DEVRELOS_REQUIRE_AUTH", "false")
	handler := withAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromContext(r.Context())
		if !ok || !principal.System {
			t.Fatal("expected local-dev system principal")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, res.Code)
	}
}

func TestWithAuthRejectsMissingOrWrongBearerToken(t *testing.T) {
	t.Setenv("DEVRELOS_API_TOKEN", "secret-token")
	handler := withAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))

	for _, header := range []string{"", "Bearer wrong-token", "Basic abc"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
		if header != "" { req.Header.Set("Authorization", header) }
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("header %q: expected %d, got %d", header, http.StatusUnauthorized, res.Code)
		}
	}
}

func TestWithAuthAcceptsBearerTokenAsSystemPrincipal(t *testing.T) {
	t.Setenv("DEVRELOS_API_TOKEN", "secret-token")
	handler := withAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromContext(r.Context())
		if !ok || !principal.System {
			t.Fatal("expected system principal")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, res.Code)
	}
}

func TestWithAuthLeavesHealthAndLoginPublic(t *testing.T) {
	t.Setenv("DEVRELOS_API_TOKEN", "secret-token")
	handler := withAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for _, path := range []string{"/healthz", "/readyz", "/api/v1/auth/login"} {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
		if res.Code != http.StatusNoContent {
			t.Fatalf("%s: expected %d, got %d", path, http.StatusNoContent, res.Code)
		}
	}
}

func TestRoleHierarchy(t *testing.T) {
	if !(roleRank("owner") > roleRank("admin") && roleRank("admin") > roleRank("editor") && roleRank("editor") > roleRank("viewer")) {
		t.Fatal("unexpected role hierarchy")
	}
	if roleRank("unknown") != 0 {
		t.Fatal("unknown roles must have no privileges")
	}
	if roleForMethod(http.MethodGet) != "viewer" || roleForMethod(http.MethodPost) != "editor" || roleForMethod(http.MethodPatch) != "editor" {
		t.Fatal("method role mapping is incorrect")
	}
}

func TestSessionTokenHashIsDeterministicAndNotRaw(t *testing.T) {
	raw := "session-value"
	first := hashSessionToken(raw)
	second := hashSessionToken(raw)
	if first != second {
		t.Fatal("session token hash must be deterministic")
	}
	if first == raw || len(first) != 64 {
		t.Fatal("session token must be stored as a SHA-256 hex hash")
	}
}
