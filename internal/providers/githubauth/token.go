package githubauth

import (
	"errors"
	"os"
	"regexp"
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

// repositorySegment matches a GitHub owner or repository name: alphanumerics
// plus hyphen, underscore and period.
var repositorySegment = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// ValidateRepository checks an operator-supplied "owner/repo" value before it is
// interpolated into an API path.
//
// A length-2 split on "/" is not enough: "../..", "owner/repo?x=1" and
// "owner/repo#f" all split into two non-empty parts, then land in the request
// path as dot segments, a query or a fragment. None of them can change the
// host, but all of them produce a silently malformed request instead of a clear
// configuration error.
func ValidateRepository(repository string) (string, error) {
	repository = strings.Trim(strings.TrimSpace(repository), "/")
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return "", errors.New("repository must use owner/repo format")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || !repositorySegment.MatchString(part) {
			return "", errors.New("repository must use owner/repo format with no path, query or fragment characters")
		}
	}
	return repository, nil
}
