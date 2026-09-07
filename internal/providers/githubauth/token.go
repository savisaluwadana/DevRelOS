package githubauth

import (
	"os"
	"strings"
)

// Token resolves a GitHub credential from an in-memory hydrated connector secret first,
// then falls back to the legacy environment-variable reference. The direct token value
// must never be persisted in connector config; the worker injects it only for a run.
func Token(config map[string]any) string {
	if token, _ := config["token"].(string); strings.TrimSpace(token) != "" {
		return strings.TrimSpace(token)
	}
	if envName, _ := config["token_env"].(string); strings.TrimSpace(envName) != "" {
		return strings.TrimSpace(os.Getenv(strings.TrimSpace(envName)))
	}
	return ""
}

func HasToken(config map[string]any) bool { return Token(config) != "" }
