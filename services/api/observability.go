package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type metricKey struct {
	Method  string
	Pattern string
	Status  int
}

type metricValue struct {
	Count       uint64
	DurationSec float64
}

type apiMetrics struct {
	startedAt time.Time
	active    atomic.Int64
	mu        sync.Mutex
	requests  map[metricKey]metricValue
}

func newAPIMetrics() *apiMetrics {
	return &apiMetrics{startedAt: time.Now().UTC(), requests: map[metricKey]metricValue{}}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusRecorder) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusRecorder) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(data)
	w.bytes += n
	return n, err
}

func (a *api) withObservability(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := sanitizeRequestID(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = newRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		r.Header.Set("X-Request-ID", requestID)

		started := time.Now()
		a.metrics.active.Add(1)
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		a.metrics.active.Add(-1)
		if recorder.status == 0 {
			recorder.status = http.StatusOK
		}
		duration := time.Since(started)
		pattern := r.Pattern
		if pattern == "" {
			pattern = normalizedFallbackPath(r.URL.Path)
		}
		a.metrics.observe(metricKey{Method: r.Method, Pattern: pattern, Status: recorder.status}, duration)
		log.Printf(`{"request_id":%q,"method":%q,"path":%q,"pattern":%q,"status":%d,"bytes":%d,"duration_ms":%d}`,
			requestID, r.Method, r.URL.Path, pattern, recorder.status, recorder.bytes, duration.Milliseconds())
	})
}

func (m *apiMetrics) observe(key metricKey, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value := m.requests[key]
	value.Count++
	value.DurationSec += duration.Seconds()
	m.requests[key] = value
}

func (a *api) metricsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprintf(w, "# HELP devrelos_api_uptime_seconds API process uptime.\n")
	fmt.Fprintf(w, "# TYPE devrelos_api_uptime_seconds gauge\n")
	fmt.Fprintf(w, "devrelos_api_uptime_seconds %.0f\n", time.Since(a.metrics.startedAt).Seconds())
	fmt.Fprintf(w, "# HELP devrelos_api_active_requests Requests currently being handled.\n")
	fmt.Fprintf(w, "# TYPE devrelos_api_active_requests gauge\n")
	fmt.Fprintf(w, "devrelos_api_active_requests %d\n", a.metrics.active.Load())

	a.metrics.mu.Lock()
	keys := make([]metricKey, 0, len(a.metrics.requests))
	values := make(map[metricKey]metricValue, len(a.metrics.requests))
	for key, value := range a.metrics.requests {
		keys = append(keys, key)
		values[key] = value
	}
	a.metrics.mu.Unlock()
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Pattern != keys[j].Pattern {
			return keys[i].Pattern < keys[j].Pattern
		}
		if keys[i].Method != keys[j].Method {
			return keys[i].Method < keys[j].Method
		}
		return keys[i].Status < keys[j].Status
	})
	fmt.Fprintln(w, "# HELP devrelos_api_requests_total Total API requests.")
	fmt.Fprintln(w, "# TYPE devrelos_api_requests_total counter")
	fmt.Fprintln(w, "# HELP devrelos_api_request_duration_seconds_sum Total request duration in seconds.")
	fmt.Fprintln(w, "# TYPE devrelos_api_request_duration_seconds_sum counter")
	for _, key := range keys {
		value := values[key]
		labels := `method="` + metricLabel(key.Method) + `",pattern="` + metricLabel(key.Pattern) + `",status="` + strconv.Itoa(key.Status) + `"`
		fmt.Fprintf(w, "devrelos_api_requests_total{%s} %d\n", labels, value.Count)
		fmt.Fprintf(w, "devrelos_api_request_duration_seconds_sum{%s} %.6f\n", labels, value.DurationSec)
	}
}

func sanitizeRequestID(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 0 || len(value) > 128 {
		return ""
	}
	for _, r := range value {
		if !(r == '-' || r == '_' || r == '.' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z') {
			return ""
		}
	}
	return value
}

func newRequestID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err == nil {
		return "req_" + hex.EncodeToString(buf)
	}
	return "req_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func normalizedFallbackPath(path string) string {
	if strings.HasPrefix(path, "/api/") {
		return "/api/*"
	}
	return path
}

func metricLabel(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return value
}
