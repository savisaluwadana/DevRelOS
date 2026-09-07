package signals

import (
	"fmt"
	"sort"
	"strings"
	"time"

	domain "github.com/savisaluwadana/DevRelOS/internal/domain/signals"
)

type Cluster struct {
	Key        string
	Title      string
	Summary    string
	Persona    string
	Severity   int
	TrendScore int
	Topics     []string
	FirstSeen  *time.Time
	LastSeen   *time.Time
	SignalIDs  []string
}

type topicRule struct {
	Key     string
	Label   string
	Persona string
	Terms   []string
}

type frictionRule struct {
	Key   string
	Label string
	Terms []string
}

var topicRules = []topicRule{
	{Key: "platform-engineering", Label: "Platform engineering", Persona: "Platform engineer", Terms: []string{"platform engineering", "internal developer platform", "developer portal", "backstage", "idp"}},
	{Key: "kubernetes", Label: "Kubernetes", Persona: "Platform engineer", Terms: []string{"kubernetes", "k8s", "helm", "pod", "cluster", "operator"}},
	{Key: "observability", Label: "Observability", Persona: "SRE / platform engineer", Terms: []string{"observability", "opentelemetry", "otel", "tracing", "telemetry", "metrics", "logs"}},
	{Key: "developer-experience", Label: "Developer experience", Persona: "Developer", Terms: []string{"developer experience", "devex", "developer onboarding", "inner loop"}},
	{Key: "ci-cd", Label: "CI/CD", Persona: "DevOps engineer", Terms: []string{"ci/cd", "continuous integration", "continuous delivery", "pipeline", "github actions", "jenkins", "build"}},
	{Key: "deployment", Label: "Deployment", Persona: "Platform engineer", Terms: []string{"deployment", "deploy", "rollout", "release", "promotion"}},
	{Key: "configuration", Label: "Configuration", Persona: "Platform engineer", Terms: []string{"configuration", "config", "environment variable", "secret management", "secrets"}},
	{Key: "documentation", Label: "Documentation", Persona: "Developer", Terms: []string{"documentation", "docs", "tutorial", "guide", "example"}},
	{Key: "security", Label: "Security and access", Persona: "Platform / security engineer", Terms: []string{"security", "rbac", "permission", "permissions", "authentication", "authorization", "auth"}},
	{Key: "cost", Label: "Cloud and tooling cost", Persona: "Engineering leader", Terms: []string{"cost", "expensive", "billing", "price", "pricing"}},
}

var frictionRules = []frictionRule{
	{Key: "missing-capability", Label: "missing capability", Terms: []string{"doesn't support", "does not support", "can't", "cannot", "missing", "wish", "need a way", "no way to"}},
	{Key: "reliability", Label: "reliability failures", Terms: []string{"broken", "breaks", "failing", "fails", "failure", "error", "crash", "unreliable"}},
	{Key: "performance", Label: "performance friction", Terms: []string{"slow", "latency", "timeout", "takes forever", "performance"}},
	{Key: "documentation", Label: "documentation gaps", Terms: []string{"docs are", "documentation is", "undocumented", "no docs", "missing docs", "unclear docs", "example missing"}},
	{Key: "complexity", Label: "complexity and setup friction", Terms: []string{"hard", "difficult", "complex", "confusing", "painful", "friction", "manual", "too many", "setup", "configure"}},
	{Key: "cost", Label: "cost pressure", Terms: []string{"too expensive", "expensive", "costly", "pricing", "cost"}},
}

type accumulator struct {
	topic          topicRule
	friction       frictionRule
	signalIDs      []string
	topics         map[string]struct{}
	relevanceTotal int
	engagement     int
	recent         int
	previous       int
	firstSeen      *time.Time
	lastSeen       *time.Time
}

// ClusterSignals groups signals with a deterministic, explainable heuristic. It intentionally
// avoids embeddings/LLMs so the baseline product works offline and cluster rebuilds are reproducible.
func ClusterSignals(items []domain.Signal, now time.Time) []Cluster {
	clusters := map[string]*accumulator{}

	for _, item := range items {
		if item.Status == "ignored" || strings.TrimSpace(item.Title+item.Body) == "" {
			continue
		}

		text := strings.ToLower(item.Title + " " + item.Body)
		topic := detectTopic(item, text)
		friction := detectFriction(text)
		key := topic.Key + ":" + friction.Key

		acc, ok := clusters[key]
		if !ok {
			acc = &accumulator{topic: topic, friction: friction, topics: map[string]struct{}{}}
			clusters[key] = acc
		}

		acc.signalIDs = append(acc.signalIDs, item.ID)
		acc.topics[topic.Key] = struct{}{}
		for _, candidate := range item.Topics {
			candidate = canonicalTopic(candidate)
			if candidate != "" {
				acc.topics[candidate] = struct{}{}
			}
		}

		relevance := 50
		if item.RelevanceScore != nil {
			relevance = *item.RelevanceScore
		}
		acc.relevanceTotal += relevance
		acc.engagement += min(item.EngagementScore, 100)

		seenAt := item.CreatedAt
		if item.OccurredAt != nil {
			seenAt = *item.OccurredAt
		}
		seenAt = seenAt.UTC()
		if acc.firstSeen == nil || seenAt.Before(*acc.firstSeen) {
			copy := seenAt
			acc.firstSeen = &copy
		}
		if acc.lastSeen == nil || seenAt.After(*acc.lastSeen) {
			copy := seenAt
			acc.lastSeen = &copy
		}

		age := now.UTC().Sub(seenAt)
		if age >= 0 && age <= 7*24*time.Hour {
			acc.recent++
		} else if age > 7*24*time.Hour && age <= 30*24*time.Hour {
			acc.previous++
		}
	}

	out := make([]Cluster, 0, len(clusters))
	for key, acc := range clusters {
		count := len(acc.signalIDs)
		if count == 0 {
			continue
		}
		avgRelevance := acc.relevanceTotal / count
		avgEngagement := acc.engagement / count
		severity := clamp(30+count*7+avgRelevance/4+avgEngagement/10, 0, 100)
		trend := clamp((acc.recent-acc.previous)*20, -100, 100)

		topics := make([]string, 0, len(acc.topics))
		for topic := range acc.topics {
			topics = append(topics, topic)
		}
		sort.Strings(topics)
		sort.Strings(acc.signalIDs)

		out = append(out, Cluster{
			Key:        key,
			Title:      fmt.Sprintf("%s: %s", acc.topic.Label, acc.friction.Label),
			Summary:    summary(acc.topic.Label, acc.friction.Label, count),
			Persona:    acc.topic.Persona,
			Severity:   severity,
			TrendScore: trend,
			Topics:     topics,
			FirstSeen:  acc.firstSeen,
			LastSeen:   acc.lastSeen,
			SignalIDs:  acc.signalIDs,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Severity == out[j].Severity {
			if out[i].TrendScore == out[j].TrendScore {
				return out[i].Key < out[j].Key
			}
			return out[i].TrendScore > out[j].TrendScore
		}
		return out[i].Severity > out[j].Severity
	})
	return out
}

func detectTopic(item domain.Signal, text string) topicRule {
	best := topicRule{Key: "general-developer-friction", Label: "Developer workflow", Persona: "Developer"}
	bestScore := 0

	for _, rule := range topicRules {
		score := 0
		for _, topic := range item.Topics {
			if canonicalTopic(topic) == rule.Key {
				score += 5
			}
		}
		for _, term := range rule.Terms {
			if strings.Contains(text, term) {
				score++
			}
		}
		if score > bestScore {
			best = rule
			bestScore = score
		}
	}
	return best
}

func detectFriction(text string) frictionRule {
	best := frictionRule{Key: "friction", Label: "recurring developer friction"}
	bestScore := 0
	for _, rule := range frictionRules {
		score := 0
		for _, term := range rule.Terms {
			if strings.Contains(text, term) {
				score++
			}
		}
		if score > bestScore {
			best = rule
			bestScore = score
		}
	}
	return best
}

func canonicalTopic(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	for _, rule := range topicRules {
		if value == rule.Key {
			return rule.Key
		}
	}
	return value
}

func summary(topic, friction string, evidence int) string {
	if evidence == 1 {
		return fmt.Sprintf("1 developer signal points to %s around %s.", friction, strings.ToLower(topic))
	}
	return fmt.Sprintf("%d developer signals point to %s around %s.", evidence, friction, strings.ToLower(topic))
}

func clamp(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
