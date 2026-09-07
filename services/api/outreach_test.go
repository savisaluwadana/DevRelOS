package main

import "testing"

func TestValidOutreachTransition(t *testing.T) {
	tests := []struct {
		current string
		next    string
		want    bool
	}{
		{"draft", "needs_approval", true},
		{"draft", "sent", false},
		{"needs_approval", "approved", true},
		{"needs_approval", "sent", false},
		{"approved", "queued", true},
		{"approved", "sent", false},
		{"queued", "sent", true},
		{"sent", "replied", true},
		{"replied", "sent", false},
		{"cancelled", "queued", false},
	}

	for _, test := range tests {
		if got := validOutreachTransition(test.current, test.next); got != test.want {
			t.Fatalf("transition %s -> %s: got %t want %t", test.current, test.next, got, test.want)
		}
	}
}
