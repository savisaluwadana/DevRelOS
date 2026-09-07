package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithAuthDisabledWithoutToken(t *testing.T) {
	t.Setenv("DEVRELOS_API_TOKEN", "")
	handler := withAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
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

func TestWithAuthAcceptsBearerToken(t *testing.T) {
	t.Setenv("DEVRELOS_API_TOKEN", "secret-token")
	handler := withAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
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
	handler := withAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for _, path := range []string{"/healthz", "/readyz"} {
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
		if res.Code != http.StatusNoContent {
			t.Fatalf("%s: expected %d, got %d", path, http.StatusNoContent, res.Code)
		}
	}
}
