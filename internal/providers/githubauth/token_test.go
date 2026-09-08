package githubauth

import (
	"strings"
	"testing"
)

func TestValidateRepositoryAcceptsWellFormed(t *testing.T) {
	for _, raw := range []string{"savisaluwadana/DevRelOS", " owner/repo ", "/owner/repo/", "a.b-c_d/e.f-g_h"} {
		got, err := ValidateRepository(raw)
		if err != nil {
			t.Errorf("ValidateRepository(%q) = %v, want it accepted", raw, err)
			continue
		}
		if strings.HasPrefix(got, "/") || strings.HasSuffix(got, "/") || strings.TrimSpace(got) != got {
			t.Errorf("ValidateRepository(%q) = %q, want it trimmed", raw, got)
		}
	}
}

func TestValidateRepositoryRejectsPathInjection(t *testing.T) {
	// Each of these splits into two non-empty parts, so the previous
	// length-only check accepted them and they reached the request path.
	for _, raw := range []string{
		"../..",
		"./..",
		"owner/repo?state=all",
		"owner/repo#fragment",
		"owner/repo%2f..",
		"owner/re po",
		"owner/repo/extra",
		"owner",
		"",
		"/",
		"owner/",
		"/repo",
	} {
		if got, err := ValidateRepository(raw); err == nil {
			t.Errorf("ValidateRepository(%q) = %q with no error, want it rejected", raw, got)
		}
	}
}

func TestTokenPrefersHydratedSecretOverEnv(t *testing.T) {
	t.Setenv("DEVRELOS_TEST_GH_TOKEN", "from-env")
	if got := Token(map[string]any{"token": "  hydrated  ", "token_env": "DEVRELOS_TEST_GH_TOKEN"}); got != "hydrated" {
		t.Fatalf("Token() = %q, want the hydrated secret, trimmed", got)
	}
	if got := Token(map[string]any{"token_env": "DEVRELOS_TEST_GH_TOKEN"}); got != "from-env" {
		t.Fatalf("Token() = %q, want the env fallback", got)
	}
	if got := Token(map[string]any{"token": "   "}); got != "" {
		t.Fatalf("Token() = %q, want empty for a blank token", got)
	}
	if HasToken(map[string]any{}) {
		t.Fatal("HasToken() = true for empty config")
	}
}
