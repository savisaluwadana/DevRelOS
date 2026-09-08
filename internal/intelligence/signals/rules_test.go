package signals

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	domain "github.com/savisaluwadana/DevRelOS/internal/domain/signals"
)

func TestDefaultRulesetIsACopy(t *testing.T) {
	first := DefaultRuleset()
	if len(first.Topics) == 0 || len(first.Frictions) == 0 {
		t.Fatal("built-in ruleset is empty")
	}
	// Mutating a returned ruleset must not corrupt the built-ins for the rest
	// of the process.
	first.Topics[0].Label = "clobbered"
	if DefaultRuleset().Topics[0].Label == "clobbered" {
		t.Fatal("DefaultRuleset handed out a reference to the built-in slice")
	}
}

func TestParseRulesetOverridesOnlyTheSectionsPresent(t *testing.T) {
	parsed, err := ParseRuleset([]byte(`{
	  "topics": [{"key": "fintech-payments", "label": "Payments", "persona": "Payments engineer",
	              "terms": ["Payment Rails", "  settlement  ", ""]}]
	}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(parsed.Topics) != 1 || parsed.Topics[0].Key != "fintech-payments" {
		t.Fatalf("topics = %+v", parsed.Topics)
	}
	// Terms are lowercased and trimmed, and blanks dropped, because matching
	// runs against lowercased text.
	if got := strings.Join(parsed.Topics[0].Terms, "|"); got != "payment rails|settlement" {
		t.Fatalf("terms = %q", got)
	}
	// Frictions were absent, so the built-ins must remain.
	if len(parsed.Frictions) != len(DefaultRuleset().Frictions) {
		t.Fatalf("absent section replaced the built-in frictions: %d", len(parsed.Frictions))
	}
}

func TestParseRulesetRejectsBadInput(t *testing.T) {
	cases := map[string]string{
		"malformed json":     `{"topics":`,
		"unknown field":      `{"topics": [{"key":"a","terms":["x"],"colour":"red"}]}`,
		"topic with no key":  `{"topics": [{"label":"No key","terms":["x"]}]}`,
		"topic with no term": `{"topics": [{"key":"a","terms":["  "]}]}`,
		"duplicate topic":    `{"topics": [{"key":"a","terms":["x"]},{"key":"a","terms":["y"]}]}`,
		"friction no key":    `{"frictions": [{"label":"x","terms":["y"]}]}`,
		"friction no term":   `{"frictions": [{"key":"a","terms":[]}]}`,
		"duplicate friction": `{"frictions": [{"key":"a","terms":["x"]},{"key":"a","terms":["y"]}]}`,
	}
	for name, raw := range cases {
		if _, err := ParseRuleset([]byte(raw)); err == nil {
			t.Errorf("%s: accepted, want an error", name)
		}
	}
}

func TestParseRulesetDefaultsLabelToKey(t *testing.T) {
	parsed, err := ParseRuleset([]byte(`{"topics": [{"key":"observability-lite","terms":["otel"]}]}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Topics[0].Label != "observability-lite" {
		t.Fatalf("Label = %q, want it to fall back to the key", parsed.Topics[0].Label)
	}
}

func TestLoadRulesetFromEnv(t *testing.T) {
	t.Setenv(RulesEnvVar, "")
	loaded, err := LoadRulesetFromEnv()
	if err != nil {
		t.Fatalf("unset env should give the built-ins: %v", err)
	}
	if len(loaded.Topics) != len(DefaultRuleset().Topics) {
		t.Fatal("unset env did not return the built-in ruleset")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(path, []byte(`{"topics":[{"key":"robotics","terms":["ros2"]}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(RulesEnvVar, path)
	loaded, err = LoadRulesetFromEnv()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded.Topics) != 1 || loaded.Topics[0].Key != "robotics" {
		t.Fatalf("topics = %+v", loaded.Topics)
	}

	// A missing or malformed file must be an error, never a silent fallback to
	// the built-ins: the operator would believe their vocabulary was in effect.
	t.Setenv(RulesEnvVar, filepath.Join(dir, "absent.json"))
	if _, err := LoadRulesetFromEnv(); err == nil {
		t.Fatal("a missing rules file was accepted")
	}
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte(`{"topics": [}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(RulesEnvVar, bad)
	if _, err := LoadRulesetFromEnv(); err == nil {
		t.Fatal("a malformed rules file was accepted")
	}
}

func TestClusterSignalsWithCustomRulesetChangesOutcome(t *testing.T) {
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	items := []domain.Signal{
		{ID: "1", Title: "ROS2 node discovery keeps failing", Body: "our ros2 fleet cannot see each other", CreatedAt: now},
		{ID: "2", Title: "ros2 launch files are broken", Body: "ros2 launch fails after upgrade", CreatedAt: now},
	}

	// The built-in vocabulary knows nothing about robotics, so these land in the
	// generic bucket.
	builtIn := ClusterSignalsWith(DefaultRuleset(), items, now)
	if len(builtIn) == 0 {
		t.Fatal("expected at least one cluster from the built-ins")
	}
	if !strings.HasPrefix(builtIn[0].Key, "general-developer-friction") {
		t.Fatalf("built-in ruleset unexpectedly recognised the domain: %q", builtIn[0].Key)
	}

	custom, err := ParseRuleset([]byte(`{"topics":[{"key":"robotics","label":"Robotics","persona":"Robotics engineer","terms":["ros2"]}]}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	clustered := ClusterSignalsWith(custom, items, now)
	if len(clustered) == 0 {
		t.Fatal("expected a cluster from the custom ruleset")
	}
	if !strings.HasPrefix(clustered[0].Key, "robotics") {
		t.Fatalf("custom ruleset was not applied, key = %q", clustered[0].Key)
	}
	if clustered[0].Persona != "Robotics engineer" {
		t.Fatalf("Persona = %q, want the configured persona", clustered[0].Persona)
	}
}
