package storage

import "testing"

func TestPageNormalizeDefaults(t *testing.T) {
	cases := []struct {
		name string
		in   Page
		want Page
	}{
		{"zero value gets the default limit", Page{}, Page{Limit: DefaultPageLimit}},
		{"negative limit falls back", Page{Limit: -5}, Page{Limit: DefaultPageLimit}},
		{"over-max limit falls back", Page{Limit: MaxPageLimit + 1}, Page{Limit: DefaultPageLimit}},
		{"max limit is allowed", Page{Limit: MaxPageLimit}, Page{Limit: MaxPageLimit}},
		{"negative offset clamps to zero", Page{Limit: 10, Offset: -3}, Page{Limit: 10}},
		{"explicit page is preserved", Page{Limit: 25, Offset: 50}, Page{Limit: 25, Offset: 50}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.in.normalize(); got != tc.want {
				t.Fatalf("normalize(%+v) = %+v, want %+v", tc.in, got, tc.want)
			}
		})
	}
}

func TestPageClauseEmitsLimitAndOffset(t *testing.T) {
	clause, args := Page{Limit: 25, Offset: 50}.clause(3)
	if clause != " LIMIT $3 OFFSET $4" {
		t.Fatalf("clause = %q", clause)
	}
	if len(args) != 2 || args[0] != 25 || args[1] != 50 {
		t.Fatalf("args = %v, want [25 50]", args)
	}
}

func TestPageClauseNumbersPlaceholdersFromNextArg(t *testing.T) {
	// A query that already binds three values must continue at $4.
	clause, _ := Page{Limit: 10}.clause(4)
	if clause != " LIMIT $4 OFFSET $5" {
		t.Fatalf("clause = %q", clause)
	}
}

func TestAllRowsEmitsNoClause(t *testing.T) {
	clause, args := AllRows().clause(2)
	if clause != "" {
		t.Fatalf("AllRows must not emit a LIMIT, got %q", clause)
	}
	if len(args) != 0 {
		t.Fatalf("AllRows must not bind args, got %v", args)
	}
}

func TestAllRowsSurvivesNormalize(t *testing.T) {
	// AllRows must not be clamped into DefaultPageLimit by normalize, or the
	// analytical reads that depend on it would be silently truncated.
	if got := AllRows().normalize(); got.Limit != unboundedLimit {
		t.Fatalf("AllRows().normalize() = %+v, want the unbounded sentinel", got)
	}
}
