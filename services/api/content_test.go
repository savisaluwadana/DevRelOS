package main

import "testing"

func TestValidContentStatus(t *testing.T) {
	for _, status := range []string{"brief", "drafting", "review", "approved", "published", "archived"} {
		if !validContentStatus(status) {
			t.Fatalf("expected %q to be a valid content status", status)
		}
	}
	if validContentStatus("sent") {
		t.Fatal("unexpected content status accepted")
	}
}

func TestValidContentTransition(t *testing.T) {
	cases := []struct {
		from string
		to   string
		want bool
	}{
		{"brief", "drafting", true},
		{"drafting", "review", true},
		{"review", "approved", true},
		{"approved", "published", true},
		{"published", "archived", true},
		{"drafting", "published", false},
		{"brief", "approved", false},
		{"review", "published", false},
	}
	for _, tc := range cases {
		if got := validContentTransition(tc.from, tc.to); got != tc.want {
			t.Errorf("validContentTransition(%q,%q)=%v want %v", tc.from, tc.to, got, tc.want)
		}
	}
}
