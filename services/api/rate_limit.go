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
	mu       sync.Mutex
	requests map[string]rateWindow
	limit    int
	// operatorLimit applies to the trusted first-party server tier. The limiter
	// runs before withAuth, so it can only key on the raw Authorization header;
	// the web app's serverFetch sends the same operator token for every
	// server-rendered page load regardless of who is browsing, which put all
	// SSR traffic (3+ API calls per page) into one bucket and produced 429s
	// during ordinary navigation.
	operatorLimit      int
	operatorAuthDigest string
	window             time.Duration
	trustProxy         bool
	enabled            bool
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

	// The trusted server tier gets its own allowance, ten times the client limit
	// by default, so page rendering is not throttled alongside untrusted callers.
	operatorLimit := limit * 10
	if raw := strings.TrimSpace(os.Getenv("DEVRELOS_RATE_LIMIT_OPERATOR_RPM")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= limit && parsed <= 100000 {
			operatorLimit = parsed
		}
	}
	operatorAuthDigest := ""
	if token := strings.TrimSpace(os.Getenv("DEVRELOS_API_TOKEN")); token != "" {
		operatorAuthDigest = authDigest("Bearer " + token)
	}

	return &rateLimiter{
		requests:           map[string]rateWindow{},
		limit:              limit,
		operatorLimit:      operatorLimit,
		operatorAuthDigest: operatorAuthDigest,
		window:             time.Minute,
		trustProxy:         trustProxy,
		enabled:            enabled,
	}
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
		allowed, retryAfter := l.allow(key, l.limitFor(key), now)
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(maxInt(1, int(retryAfter.Seconds()))))
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// limitFor returns the allowance for a client key: the trusted server tier is
// identified by the operator token's Authorization digest.
func (l *rateLimiter) limitFor(key string) int {
	if l.operatorAuthDigest != "" && key == "auth:"+l.operatorAuthDigest {
		return l.operatorLimit
	}
	return l.limit
}

func (l *rateLimiter) allow(key string, limit int, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	window := l.requests[key]
	if window.StartedAt.IsZero() || now.Sub(window.StartedAt) >= l.window {
		l.requests[key] = rateWindow{StartedAt: now, Count: 1}
		l.cleanup(now)
		return true, 0
	}
	if window.Count >= limit {
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
		return "auth:" + authDigest(auth)
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

func authDigest(authorization string) string {
	digest := sha256.Sum256([]byte(authorization))
	return hex.EncodeToString(digest[:8])
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
