package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type rateWindow struct {
	StartedAt time.Time
	Count     int
}

type rateLimiter struct {
	mu         sync.Mutex
	requests   map[string]rateWindow
	limit      int
	window     time.Duration
	trustProxy bool
	enabled    bool
}

func newRateLimiterFromEnv() *rateLimiter {
	limit := 120
	if raw := strings.TrimSpace(os.Getenv("DEVRELOS_RATE_LIMIT_RPM")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 10 && parsed <= 10000 {
			limit = parsed
		}
	}
	enabled := true
	if raw := strings.TrimSpace(strings.ToLower(os.Getenv("DEVRELOS_RATE_LIMIT_ENABLED"))); raw != "" {
		enabled = raw == "true" || raw == "1" || raw == "yes" || raw == "on"
	}
	trustProxy := false
	if raw := strings.TrimSpace(strings.ToLower(os.Getenv("DEVRELOS_TRUST_PROXY"))); raw != "" {
		trustProxy = raw == "true" || raw == "1" || raw == "yes" || raw == "on"
	}
	return &rateLimiter{requests: map[string]rateWindow{}, limit: limit, window: time.Minute, trustProxy: trustProxy, enabled: enabled}
}

func (l *rateLimiter) Middleware(next http.Handler) http.Handler {
	if l == nil || !l.enabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}
		key := l.clientKey(r)
		now := time.Now().UTC()
		allowed, retryAfter := l.allow(key, now)
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(maxInt(1, int(retryAfter.Seconds()))))
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *rateLimiter) allow(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	window := l.requests[key]
	if window.StartedAt.IsZero() || now.Sub(window.StartedAt) >= l.window {
		l.requests[key] = rateWindow{StartedAt: now, Count: 1}
		l.cleanup(now)
		return true, 0
	}
	if window.Count >= l.limit {
		return false, l.window - now.Sub(window.StartedAt)
	}
	window.Count++
	l.requests[key] = window
	return true, 0
}

func (l *rateLimiter) cleanup(now time.Time) {
	if len(l.requests) < 2048 {
		return
	}
	for key, window := range l.requests {
		if now.Sub(window.StartedAt) >= 2*l.window {
			delete(l.requests, key)
		}
	}
}

func (l *rateLimiter) clientKey(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth != "" {
		digest := sha256.Sum256([]byte(auth))
		return "auth:" + hex.EncodeToString(digest[:8])
	}
	if l.trustProxy {
		if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
			return "ip:" + forwarded
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return "ip:" + host
	}
	return "ip:" + strings.TrimSpace(r.RemoteAddr)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
