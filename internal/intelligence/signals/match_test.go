package signals

import "testing"

func TestContainsTermRejectsSubstringsOfOtherWords(t *testing.T) {
	// Every one of these produced a false pain point with strings.Contains.
	cases := []struct{ text, term string }{
		{"hardware as fast as software", "hard"},
		{"terror management theory", "error"},
		{"costa rica cloud region", "cost"},
		{"a hardened image", "hard"},
		{"manually harden the cluster", "hard"},
		{"complexity theory", "complex"}, // "complexity" is a noun, not a complaint
	}
	for _, c := range cases {
		if containsTerm(c.text, c.term) {
			t.Errorf("containsTerm(%q, %q) = true, want false", c.text, c.term)
		}
	}
}

func TestContainsTermMatchesWholeWords(t *testing.T) {
	cases := []struct{ text, term string }{
		{"this is hard", "hard"},
		{"hard to configure", "hard"},
		{"it is hard.", "hard"},
		{"(hard)", "hard"},
		{"an error occurred", "error"},
		{"multiple errors occurred", "error"}, // plural
		{"the cost is high", "cost"},
		{"rising costs", "cost"},                     // plural
		{"we configured it", "configure"},            // -d
		{"configuring this is painful", "configure"}, // e-dropped -ing
		{"setups everywhere", "setup"},               // plural
		{"the build failed", "failed"},
		{"platform engineering is hard", "platform engineering"}, // phrase
		{"too many steps", "too many"},                           // phrase
	}
	for _, c := range cases {
		if !containsTerm(c.text, c.term) {
			t.Errorf("containsTerm(%q, %q) = false, want true", c.text, c.term)
		}
	}
}

func TestContainsTermEdgeCases(t *testing.T) {
	if containsTerm("anything", "") {
		t.Error("an empty term must not match")
	}
	if containsTerm("hi", "a much longer term") {
		t.Error("a term longer than the text must not match")
	}
	// A later occurrence must still be found when an earlier one is embedded.
	if !containsTerm("hardware is hard", "hard") {
		t.Error("an embedded first occurrence must not hide a later real match")
	}
	// Non-ASCII adjacency counts as a boundary.
	if !containsTerm("это hard", "hard") {
		t.Error("a term after a multi-byte rune should still match")
	}
}
