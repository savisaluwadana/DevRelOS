package signals

import "strings"

// inflections are the endings a term may carry and still be the same word.
// Ordered longest-first so the longest valid ending is tried before a shorter
// prefix of it.
var inflections = []string{"ing", "es", "ed", "s", "d"}

// containsTerm reports whether text contains term as a word rather than as a
// substring of an unrelated word.
//
// Matching used to be strings.Contains, which produced false friction on
// ordinary text: "hard" matched "Hardware", "error" matched "terror", and
// "cost" matched "Costa Rica". Those became developer pain points.
//
// A match must begin at a word boundary and end either at a word boundary or on
// a common inflection, so "cost" still matches "costs" and "configure" still
// matches "configured" while "Costa" and "Hardware" no longer match. Multi-word
// terms such as "platform engineering" are matched as whole phrases.
//
// Both text and term are expected to be lowercase already; the ruleset
// normalises terms on load and the caller lowercases the text.
func containsTerm(text, term string) bool {
	if term == "" || len(term) > len(text) {
		return false
	}
	if matchesAt(text, term) {
		return true
	}
	// Try the e-dropped stem so "configure" also matches "configuring".
	if stem, ok := eDroppedStem(term); ok {
		return matchesAt(text, stem+"ing")
	}
	return false
}

func matchesAt(text, term string) bool {
	if term == "" || len(term) > len(text) {
		return false
	}
	for offset := 0; ; {
		index := strings.Index(text[offset:], term)
		if index < 0 {
			return false
		}
		start := offset + index
		end := start + len(term)
		if isBoundedWord(text, start, end) {
			return true
		}
		// Advance one byte past this start so overlapping matches are still found.
		offset = start + 1
		if offset >= len(text) {
			return false
		}
	}
}

func isBoundedWord(text string, start, end int) bool {
	if start > 0 && isWordByte(text[start-1]) {
		return false
	}
	if end >= len(text) || !isWordByte(text[end]) {
		return true
	}
	// The term is followed by more letters: accept only a known inflection that
	// itself ends the word.
	rest := text[end:]
	for _, suffix := range inflections {
		if strings.HasPrefix(rest, suffix) {
			after := end + len(suffix)
			if after >= len(text) || !isWordByte(text[after]) {
				return true
			}
		}
	}
	return false
}

// eDroppedStem returns the term with a trailing "e" removed, which is how
// English forms the present participle: "configure" becomes "configuring", not
// "configureing". Without this, the most common phrasing of a friction term
// ("configuring this is painful") would not match.
func eDroppedStem(term string) (string, bool) {
	if len(term) > 2 && strings.HasSuffix(term, "e") {
		return term[:len(term)-1], true
	}
	return "", false
}

// isWordByte treats ASCII letters and digits as word characters. Punctuation,
// whitespace and multi-byte runes act as boundaries, which is the behaviour we
// want: a term adjacent to a non-ASCII character is at a boundary.
func isWordByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z':
		return true
	case b >= 'A' && b <= 'Z':
		return true
	case b >= '0' && b <= '9':
		return true
	default:
		return false
	}
}
