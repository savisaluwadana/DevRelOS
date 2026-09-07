package main

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
)

func withAuth(next http.Handler) http.Handler {
	token := strings.TrimSpace(os.Getenv("DEVRELOS_API_TOKEN"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token == "" || r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}

		const prefix = "Bearer "
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(header, prefix) || !secureEqual(strings.TrimSpace(strings.TrimPrefix(header, prefix)), token) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="devrelos-api"`)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func secureEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
