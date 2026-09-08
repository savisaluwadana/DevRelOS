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
	if ok, _ := limiter.allow("client", limiter.limit, now); !ok {
		t.Fatal("first request should pass")
	}
	if ok, _ := limiter.allow("client", limiter.limit, now.Add(time.Second)); !ok {
		t.Fatal("second request should pass")
	}
	if ok, retry := limiter.allow("client", limiter.limit, now.Add(2*time.Second)); ok || retry <= 0 {
		t.Fatal("third request should be limited with retry")
	}
	if ok, _ := limiter.allow("client", limiter.limit, now.Add(time.Minute)); !ok {
		t.Fatal("new window should pass")
	}
}

func TestRateLimiterHealthExemption(t *testing.T) {
	limiter := &rateLimiter{requests: map[string]rateWindow{}, limit: 1, window: time.Minute, enabled: true}
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("health request %d got %d", i, rec.Code)
		}
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
	if keyA == keyB {
		t.Fatal("distinct bearer credentials must have distinct rate keys")
	}
	if keyA == "auth:alpha" || keyB == "auth:beta" {
		t.Fatal("raw bearer credentials must not be retained in keys")
	}
}

func TestRateLimiterGivesOperatorTokenItsOwnAllowance(t *testing.T) {
	t.Setenv("DEVRELOS_RATE_LIMIT_RPM", "10")
	t.Setenv("DEVRELOS_API_TOKEN", "operator-secret")
	limiter := newRateLimiterFromEnv()

	if limiter.limit != 10 {
		t.Fatalf("client limit = %d, want 10", limiter.limit)
	}
	if limiter.operatorLimit != 100 {
		t.Fatalf("operator limit = %d, want 10x the client limit", limiter.operatorLimit)
	}

	operatorReq := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	operatorReq.Header.Set("Authorization", "Bearer operator-secret")
	sessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	sessionReq.Header.Set("Authorization", "Bearer ds_somebrowsersession")

	operatorKey := limiter.clientKey(operatorReq)
	sessionKey := limiter.clientKey(sessionReq)
	if got := limiter.limitFor(operatorKey); got != 100 {
		t.Fatalf("operator request got limit %d, want 100", got)
	}
	if got := limiter.limitFor(sessionKey); got != 10 {
		t.Fatalf("session request got limit %d, want the client limit 10", got)
	}

	// The server tier must keep serving past the client limit: this is the case
	// that used to 429 during ordinary page navigation.
	now := time.Unix(2000, 0).UTC()
	for i := 0; i < 40; i++ {
		if ok, _ := limiter.allow(operatorKey, limiter.limitFor(operatorKey), now.Add(time.Duration(i)*time.Millisecond)); !ok {
			t.Fatalf("operator request %d was throttled below the operator limit", i)
		}
	}
	// An untrusted caller is still capped at the client limit.
	for i := 0; i < 10; i++ {
		if ok, _ := limiter.allow(sessionKey, limiter.limitFor(sessionKey), now.Add(time.Duration(i)*time.Millisecond)); !ok {
			t.Fatalf("session request %d throttled early", i)
		}
	}
	if ok, retry := limiter.allow(sessionKey, limiter.limitFor(sessionKey), now.Add(20*time.Millisecond)); ok || retry <= 0 {
		t.Fatal("session caller should be throttled once past the client limit")
	}
}

func TestRateLimiterOperatorTierInertWithoutToken(t *testing.T) {
	t.Setenv("DEVRELOS_RATE_LIMIT_RPM", "10")
	t.Setenv("DEVRELOS_API_TOKEN", "")
	limiter := newRateLimiterFromEnv()
	if limiter.operatorAuthDigest != "" {
		t.Fatal("no operator token configured, so no request should qualify for the operator tier")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	req.Header.Set("Authorization", "Bearer anything")
	if got := limiter.limitFor(limiter.clientKey(req)); got != 10 {
		t.Fatalf("limit = %d, want the client limit 10", got)
	}
}
