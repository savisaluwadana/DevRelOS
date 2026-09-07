package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterWindow(t *testing.T) {
	limiter := &rateLimiter{requests: map[string]rateWindow{}, limit: 2, window: time.Minute, enabled: true}
	now := time.Unix(1000, 0).UTC()
	if ok, _ := limiter.allow("client", now); !ok { t.Fatal("first request should pass") }
	if ok, _ := limiter.allow("client", now.Add(time.Second)); !ok { t.Fatal("second request should pass") }
	if ok, retry := limiter.allow("client", now.Add(2*time.Second)); ok || retry <= 0 { t.Fatal("third request should be limited with retry") }
	if ok, _ := limiter.allow("client", now.Add(time.Minute)); !ok { t.Fatal("new window should pass") }
}

func TestRateLimiterHealthExemption(t *testing.T) {
	limiter := &rateLimiter{requests: map[string]rateWindow{}, limit: 1, window: time.Minute, enabled: true}
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent { t.Fatalf("health request %d got %d", i, rec.Code) }
	}
}

func TestRateLimiterHashesBearerIdentity(t *testing.T) {
	limiter := &rateLimiter{trustProxy: true}
	reqA := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	reqA.Header.Set("Authorization", "Bearer alpha")
	reqA.Header.Set("X-Forwarded-For", "203.0.113.10")
	reqB := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	reqB.Header.Set("Authorization", "Bearer beta")
	reqB.Header.Set("X-Forwarded-For", "203.0.113.10")
	keyA := limiter.clientKey(reqA)
	keyB := limiter.clientKey(reqB)
	if keyA == keyB { t.Fatal("distinct bearer credentials must have distinct rate keys") }
	if keyA == "auth:alpha" || keyB == "auth:beta" { t.Fatal("raw bearer credentials must not be retained in keys") }
}
