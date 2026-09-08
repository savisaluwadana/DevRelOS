package main

import (
	"errors"
	"strings"
	"testing"
)

func TestGuardContainsPanic(t *testing.T) {
	// Returning from guard at all is the assertion: without recover the panic
	// would have taken the process down and this test binary with it.
	guard("test-stage", func() { panic("boom") })
}

func TestGuardRunsNormalWork(t *testing.T) {
	ran := false
	guard("test-stage", func() { ran = true })
	if !ran {
		t.Fatal("guard did not run the stage")
	}
}

func TestGuardJobConvertsPanicToError(t *testing.T) {
	processed, err := guardJob("test-job", func() (bool, error) { panic("kaboom") })
	if err == nil {
		t.Fatal("expected a panic to surface as an error")
	}
	if processed {
		t.Fatal("a panicking job must not report work as processed")
	}
	if !strings.Contains(err.Error(), "kaboom") {
		t.Fatalf("error should name the panic, got %v", err)
	}
}

func TestGuardJobPassesResultsThrough(t *testing.T) {
	sentinel := errors.New("normal failure")
	processed, err := guardJob("test-job", func() (bool, error) { return true, sentinel })
	if !processed || !errors.Is(err, sentinel) {
		t.Fatalf("guardJob altered a normal result: processed=%v err=%v", processed, err)
	}
}

func TestGuardJobSurvivesNilMapPanic(t *testing.T) {
	// The shape of failure a malformed upstream payload actually produces.
	processed, err := guardJob("test-job", func() (bool, error) {
		var payload map[string]any
		payload["key"] = "value" // assignment to entry in nil map
		return true, nil
	})
	if err == nil || processed {
		t.Fatalf("expected a contained failure, got processed=%v err=%v", processed, err)
	}
}
