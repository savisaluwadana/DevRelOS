package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWithAuthDisabledWithoutToken(t *testing.T) {
	t.Setenv("DEVRELOS_API_TOKEN", "")
	t.Setenv("DEVRELOS_REQUIRE_AUTH", "false")
	a := &api{}
	handler := a.withAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if currentActor(r).Kind != "operator" {
			t.Fatalf("expected local operator actor")
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
	a := &api{}
	handler := a.withAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))

	for _, header := range []string{"", "Bearer wrong-token", "Basic abc"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("header %q: expected %d, got %d", header, http.StatusUnauthorized, res.Code)
		}
	}
}

func TestWithAuthAcceptsOperatorBearerToken(t *testing.T) {
	t.Setenv("DEVRELOS_API_TOKEN", "secret-token")
	a := &api{}
	handler := a.withAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if currentActor(r).Kind != "operator" {
			t.Fatalf("expected operator actor")
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

func TestWithAuthLeavesHealthEndpointsPublic(t *testing.T) {
	t.Setenv("DEVRELOS_API_TOKEN", "secret-token")
	a := &api{}
	handler := a.withAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for _, path := range []string{"/healthz", "/readyz"} {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
		if res.Code != http.StatusNoContent {
			t.Fatalf("%s: expected %d, got %d", path, http.StatusNoContent, res.Code)
		}
	}
}

func TestGenerateAPIKey(t *testing.T) {
	plain, prefix, hash, err := generateAPIKey()
	if err != nil {
		t.Fatalf("generate api key: %v", err)
	}
	if !strings.HasPrefix(plain, "drk_") {
		t.Fatalf("expected drk_ prefix, got %q", plain)
	}
	if prefix == "" || len(prefix) > 12 {
		t.Fatalf("unexpected display prefix %q", prefix)
	}
	if len(hash) != 64 {
		t.Fatalf("expected sha256 hex hash, got length %d", len(hash))
	}
}

func TestRoleRank(t *testing.T) {
	if !(roleRank("viewer") < roleRank("editor") && roleRank("editor") < roleRank("admin") && roleRank("admin") < roleRank("owner")) {
		t.Fatalf("role hierarchy is invalid")
	}
}
