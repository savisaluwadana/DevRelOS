package main

import "testing"

func TestValidFeedbackTransition(t *testing.T) {
	cases := []struct {
		from string
		to   string
		want bool
	}{
		{"new", "triaged", true},
		{"triaged", "planned", true},
		{"planned", "in_progress", true},
		{"in_progress", "shipped", true},
		{"shipped", "closed", true},
		{"new", "shipped", false},
		{"triaged", "shipped", false},
		{"closed", "in_progress", false},
	}
	for _, tc := range cases {
		if got := validFeedbackTransition(tc.from, tc.to); got != tc.want {
			t.Errorf("validFeedbackTransition(%q,%q)=%v want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestFeedbackValidationHelpers(t *testing.T) {
	if !validFeedbackStatus("shipped") || validFeedbackStatus("published") {
		t.Fatal("unexpected feedback status validation")
	}
	if !validFeedbackSource("pain_point") || validFeedbackSource("outreach") {
		t.Fatal("unexpected feedback source validation")
	}
	if !scoreInRange(0) || !scoreInRange(100) || scoreInRange(-1) || scoreInRange(101) {
		t.Fatal("unexpected feedback score validation")
	}
}
