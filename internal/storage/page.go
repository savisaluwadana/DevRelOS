package storage

import "fmt"

// Page bounds a list query.
//
// Most list queries originally had no LIMIT at all, so a project that
// accumulated events, talks, outreach or contacts over time would load every
// row on every request. The four queries that did have a limit had no way to
// reach rows past it. Page supplies both halves.
//
// The zero value means "first page at the default size", so a caller that does
// not care about paging still gets a bounded query.
type Page struct {
	Limit  int
	Offset int
}

const (
	// DefaultPageLimit is generous enough that existing single-page callers and
	// the server-rendered views behave as before for realistic project sizes.
	DefaultPageLimit = 200
	// MaxPageLimit caps what a caller may request in one page.
	MaxPageLimit = 500

	unboundedLimit = -1
)

// AllRows requests every matching row, emitting no LIMIT clause.
//
// This is reserved for internal analytical reads that must see the complete set
// to produce a correct answer - opportunity scoring ranks a cross product of
// communities and talks, so a truncated input silently yields wrong rankings.
// Never use it to serve a request-facing list endpoint: make those pageable
// instead. Keeping it a named, greppable choice is the point; these queries used
// to be unbounded by accident, with no way to tell intent from oversight.
func AllRows() Page {
	return Page{Limit: unboundedLimit}
}

// clause returns the SQL suffix to append and the args that go with it.
// nextArg is the 1-based number of the next free placeholder in the query.
func (p Page) clause(nextArg int) (string, []any) {
	p = p.normalize()
	if p.Limit == unboundedLimit {
		return "", nil
	}
	return fmt.Sprintf(" LIMIT $%d OFFSET $%d", nextArg, nextArg+1), []any{p.Limit, p.Offset}
}

// normalize clamps the page into range. An out-of-range or absent limit falls
// back to DefaultPageLimit rather than erroring, matching how the existing
// limit-bearing queries already behaved.
func (p Page) normalize() Page {
	if p.Limit == unboundedLimit {
		return Page{Limit: unboundedLimit}
	}
	limit := p.Limit
	if limit <= 0 || limit > MaxPageLimit {
		limit = DefaultPageLimit
	}
	offset := p.Offset
	if offset < 0 {
		offset = 0
	}
	return Page{Limit: limit, Offset: offset}
}
