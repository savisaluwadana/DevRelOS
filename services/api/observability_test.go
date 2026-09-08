package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSanitizeRequestID(t *testing.T) {
	for _, value := range []string{"req_abc-123", "trace.123", "A_B"} {
		if got := sanitizeRequestID(value); got != value {
			t.Fatalf("expected %q, got %q", value, got)
		}
	}
	for _, value := range []string{"", "has space", "line\nbreak", strings.Repeat("a", 129)} {
		if got := sanitizeRequestID(value); got != "" {
			t.Fatalf("expected %q rejected, got %q", value, got)
		}
	}
}

func TestObservabilityEmitsRequestIDAndMetrics(t *testing.T) {
	a := &api{metrics: newAPIMetrics()}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) })
	// Mirror the production chain: withAuth sits between observability and the
	// mux and hands the mux a r.WithContext() clone, so the matched pattern must
	// travel back out through withRoutePattern rather than the outer request.
	fakeAuth := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		withRoutePattern(mux).ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorContextKey{}, actor{Kind: "operator"})))
	})
	handler := a.withObservability(fakeAuth)
	req := httptest.NewRequest(http.MethodGet, "/hello/123", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d", rec.Code)
	}
	if !strings.HasPrefix(rec.Header().Get("X-Request-ID"), "req_") {
		t.Fatalf("missing generated request id: %q", rec.Header().Get("X-Request-ID"))
	}

	metricsRec := httptest.NewRecorder()
	a.metricsHandler(metricsRec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := metricsRec.Body.String()
	if !strings.Contains(body, `/hello/{id}`) {
		t.Fatalf("expected templated route metric, got %s", body)
	}
	if strings.Contains(body, `pattern="/hello/123"`) {
		t.Fatalf("route pattern was lost through the request clone, got %s", body)
	}
	if !strings.Contains(body, `status="201"`) {
		t.Fatalf("expected status metric, got %s", body)
	}
}

func TestObservabilityRecoversPanicAndReleasesActiveGauge(t *testing.T) {
	a := &api{metrics: newAPIMetrics()}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /boom", func(http.ResponseWriter, *http.Request) { panic("handler exploded") })
	handler := a.withObservability(withRoutePattern(mux))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/boom", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 after panic, got %d", rec.Code)
	}
	if got := a.metrics.active.Load(); got != 0 {
		t.Fatalf("active-request gauge leaked after panic: %d", got)
	}

	metricsRec := httptest.NewRecorder()
	a.metricsHandler(metricsRec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if body := metricsRec.Body.String(); !strings.Contains(body, `status="500"`) {
		t.Fatalf("expected panic recorded as 500, got %s", body)
	}
}
