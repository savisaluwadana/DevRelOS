package signals

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// RulesEnvVar names the optional JSON file that overrides the built-in
// clustering vocabularies.
//
// The defaults are tuned for platform engineering, Kubernetes, CI/CD,
// observability and developer experience. That is correct for the project this
// was built for and wrong for everyone else, and the rules used to be compiled
// in with no way to change them short of a rebuild.
//
// A file was chosen over a database table deliberately: the rules are
// deployment-wide configuration rather than per-workspace data, and this needs
// no migration, no UI and no API surface. If per-workspace vocabularies are
// ever needed, this is the seam to move.
const RulesEnvVar = "DEVRELOS_CLUSTERING_RULES"

// DefaultMinEvidence is how many signals a cluster needs before it is reported
// as a recurring pain point.
const DefaultMinEvidence = 2

// Ruleset is the topic and friction vocabulary used to cluster signals.
type Ruleset struct {
	Topics    []topicRule
	Frictions []frictionRule
	// MinEvidence is the number of signals required to report a cluster. A
	// value below 1 falls back to DefaultMinEvidence.
	MinEvidence int
}

// rulesFile is the on-disk shape. Field names are lowercase so the file reads
// naturally; see docs/INTEGRATIONS.md for an example.
type rulesFile struct {
	Topics []struct {
		Key     string   `json:"key"`
		Label   string   `json:"label"`
		Persona string   `json:"persona"`
		Terms   []string `json:"terms"`
	} `json:"topics"`
	Frictions []struct {
		Key   string   `json:"key"`
		Label string   `json:"label"`
		Terms []string `json:"terms"`
	} `json:"frictions"`
	MinEvidence *int `json:"minEvidence"`
}

// DefaultRuleset returns a copy of the built-in vocabularies.
func DefaultRuleset() Ruleset {
	return Ruleset{
		Topics:      append([]topicRule(nil), defaultTopicRules...),
		Frictions:   append([]frictionRule(nil), defaultFrictionRules...),
		MinEvidence: DefaultMinEvidence,
	}
}

// ParseRuleset reads a ruleset from JSON. Both sections are optional; an absent
// or empty section keeps the built-in vocabulary for that section, so a file can
// override only topics without having to restate every friction term.
func ParseRuleset(raw []byte) (Ruleset, error) {
	var parsed rulesFile
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&parsed); err != nil {
		return Ruleset{}, fmt.Errorf("parse clustering rules: %w", err)
	}

	out := DefaultRuleset()
	if len(parsed.Topics) > 0 {
		topics := make([]topicRule, 0, len(parsed.Topics))
		seen := map[string]bool{}
		for i, item := range parsed.Topics {
			key := strings.TrimSpace(item.Key)
			if key == "" {
				return Ruleset{}, fmt.Errorf("clustering rules: topics[%d] has no key", i)
			}
			if seen[key] {
				return Ruleset{}, fmt.Errorf("clustering rules: duplicate topic key %q", key)
			}
			terms := normalizeTerms(item.Terms)
			if len(terms) == 0 {
				return Ruleset{}, fmt.Errorf("clustering rules: topic %q has no terms", key)
			}
			seen[key] = true
			label := strings.TrimSpace(item.Label)
			if label == "" {
				label = key
			}
			topics = append(topics, topicRule{Key: key, Label: label, Persona: strings.TrimSpace(item.Persona), Terms: terms})
		}
		out.Topics = topics
	}

	if len(parsed.Frictions) > 0 {
		frictions := make([]frictionRule, 0, len(parsed.Frictions))
		seen := map[string]bool{}
		for i, item := range parsed.Frictions {
			key := strings.TrimSpace(item.Key)
			if key == "" {
				return Ruleset{}, fmt.Errorf("clustering rules: frictions[%d] has no key", i)
			}
			if seen[key] {
				return Ruleset{}, fmt.Errorf("clustering rules: duplicate friction key %q", key)
			}
			terms := normalizeTerms(item.Terms)
			if len(terms) == 0 {
				return Ruleset{}, fmt.Errorf("clustering rules: friction %q has no terms", key)
			}
			seen[key] = true
			label := strings.TrimSpace(item.Label)
			if label == "" {
				label = key
			}
			frictions = append(frictions, frictionRule{Key: key, Label: label, Terms: terms})
		}
		out.Frictions = frictions
	}

	if parsed.MinEvidence != nil {
		if *parsed.MinEvidence < 1 {
			return Ruleset{}, fmt.Errorf("clustering rules: minEvidence must be at least 1, got %d", *parsed.MinEvidence)
		}
		out.MinEvidence = *parsed.MinEvidence
	}
	return out, nil
}

func normalizeTerms(terms []string) []string {
	out := make([]string, 0, len(terms))
	for _, term := range terms {
		// Matching is done against lowercased text, so normalise here rather
		// than trusting the file to be lowercase.
		term = strings.ToLower(strings.TrimSpace(term))
		if term != "" {
			out = append(out, term)
		}
	}
	return out
}

// LoadRulesetFromEnv reads the override file named by RulesEnvVar. It returns
// the built-in ruleset when the variable is unset.
//
// A malformed or unreadable file is an error, never a silent fallback: quietly
// reverting to the defaults would leave an operator believing their vocabulary
// was in effect while every cluster was computed with the wrong terms.
func LoadRulesetFromEnv() (Ruleset, error) {
	path := strings.TrimSpace(os.Getenv(RulesEnvVar))
	if path == "" {
		return DefaultRuleset(), nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Ruleset{}, fmt.Errorf("read %s=%q: %w", RulesEnvVar, path, err)
	}
	return ParseRuleset(raw)
}

var (
	activeRulesOnce sync.Once
	activeRules     Ruleset
	activeRulesErr  error
)

// ActiveRuleset resolves the ruleset once per process.
func ActiveRuleset() (Ruleset, error) {
	activeRulesOnce.Do(func() {
		activeRules, activeRulesErr = LoadRulesetFromEnv()
	})
	return activeRules, activeRulesErr
}
