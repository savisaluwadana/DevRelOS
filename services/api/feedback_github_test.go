package main

import "testing"

func TestParseGitHubRepository(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		owner     string
		repo      string
		wantError bool
	}{
		{name: "slug", value: "openai/openai-go", owner: "openai", repo: "openai-go"},
		{name: "url", value: "https://github.com/cncf/openchoreo", owner: "cncf", repo: "openchoreo"},
		{name: "git suffix", value: "owner/repo.git", owner: "owner", repo: "repo"},
		{name: "wrong host", value: "https://example.com/owner/repo", wantError: true},
		{name: "extra path", value: "owner/repo/issues", wantError: true},
		{name: "empty", value: "", wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			owner, repo, err := parseGitHubRepository(test.value)
			if test.wantError {
				if err == nil { t.Fatalf("expected error") }
				return
			}
			if err != nil { t.Fatalf("unexpected error: %v", err) }
			if owner != test.owner || repo != test.repo {
				t.Fatalf("got %s/%s, want %s/%s", owner, repo, test.owner, test.repo)
			}
		})
	}
}
