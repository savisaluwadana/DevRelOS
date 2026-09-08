package connectors

import (
	"context"
	"sort"
	"testing"
)

type stubProvider struct{ id string }

func (s stubProvider) ID() string                          { return s.id }
func (s stubProvider) Capabilities() []Capability          { return []Capability{CapabilitySearch} }
func (s stubProvider) ValidateConfig(map[string]any) error { return nil }
func (s stubProvider) Policy(map[string]any) Policy        { return Policy{} }
func (s stubProvider) Fetch(context.Context, map[string]any, FetchRequest) (FetchResult, error) {
	return FetchResult{}, nil
}

func TestRegistryGet(t *testing.T) {
	registry := NewRegistry(stubProvider{id: "github.issues"}, stubProvider{id: "rss"})

	provider, ok := registry.Get("rss")
	if !ok {
		t.Fatal("Get(\"rss\") reported missing")
	}
	if provider.ID() != "rss" {
		t.Fatalf("Get returned %q", provider.ID())
	}

	if _, ok := registry.Get("does-not-exist"); ok {
		t.Fatal("Get reported an unregistered provider as present")
	}
	// An unknown id must report absence rather than a nil provider callers
	// would dereference.
	if provider, ok := registry.Get(""); ok || provider != nil {
		t.Fatalf("Get(\"\") = %v, %v; want nil, false", provider, ok)
	}
}

func TestRegistryProvidersReturnsAll(t *testing.T) {
	registry := NewRegistry(stubProvider{id: "b"}, stubProvider{id: "a"}, stubProvider{id: "c"})
	ids := make([]string, 0, 3)
	for _, provider := range registry.Providers() {
		ids = append(ids, provider.ID())
	}
	sort.Strings(ids)
	if len(ids) != 3 || ids[0] != "a" || ids[1] != "b" || ids[2] != "c" {
		t.Fatalf("Providers() = %v, want all three registered ids", ids)
	}
}

func TestRegistryLastRegistrationWinsOnDuplicateID(t *testing.T) {
	// Registration is keyed by ID, so a duplicate silently replaces rather than
	// producing two entries. Pin the behaviour so it is a decision, not a
	// surprise.
	first := stubProvider{id: "rss"}
	second := stubProvider{id: "rss"}
	registry := NewRegistry(first, second)
	if got := len(registry.Providers()); got != 1 {
		t.Fatalf("Providers() has %d entries for a duplicate id, want 1", got)
	}
}

func TestEmptyRegistry(t *testing.T) {
	registry := NewRegistry()
	if got := len(registry.Providers()); got != 0 {
		t.Fatalf("empty registry reported %d providers", got)
	}
	if _, ok := registry.Get("rss"); ok {
		t.Fatal("empty registry reported a provider")
	}
}
