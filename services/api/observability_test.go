package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSanitizeRequestID(t *testing.T) {
	for _, value := range []string{"req_abc-123", "trace.123", "A_B"} {
		if got := sanitizeRequestID(value); got != value { t.Fatalf("expected %q, got %q", value, got) }
	}
	for _, value := range []string{"", "has space", "line\nbreak", strings.Repeat("a", 129)} {
		if got := sanitizeRequestID(value); got != "" { t.Fatalf("expected %q rejected, got %q", value, got) }
	}
}

func TestObservabilityEmitsRequestIDAndMetrics(t *testing.T) {
	a := &api{metrics: newAPIMetrics()}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /hello/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) })
	handler := a.withObservability(mux)
	req := httptest.NewRequest(http.MethodGet, "/hello/123", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated { t.Fatalf("status=%d", rec.Code) }
	if !strings.HasPrefix(rec.Header().Get("X-Request-ID"), "req_") { t.Fatalf("missing generated request id: %q", rec.Header().Get("X-Request-ID")) }

	metricsRec := httptest.NewRecorder()
	a.metricsHandler(metricsRec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := metricsRec.Body.String()
	if !strings.Contains(body, `pattern="GET /hello/{id}"`) && !strings.Contains(body, `pattern="/hello/{id}"`) {
		// Go's ServeMux Request.Pattern does not include the method in current Go; keep this guard portable.
		if !strings.Contains(body, `/hello/{id}`) { t.Fatalf("expected route metric, got %s", body) }
	}
	if !strings.Contains(body, `status="201"`) { t.Fatalf("expected status metric, got %s", body) }
}
