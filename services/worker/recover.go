package main

import (
	"fmt"
	"log"
	"runtime/debug"
)

// guard runs one tick of a worker stage and converts a panic into a logged
// error instead of a process exit.
//
// The worker had no recover anywhere: a panic in any provider parser, media
// step or delivery attempt terminated the whole process, which stopped
// connector ingestion, media rendering and outreach delivery together until
// someone restarted the container. A single malformed upstream payload should
// cost one job, not the worker.
func guard(stage string, fn func()) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf(`{"level":"error","stage":%q,"msg":"worker stage panicked","panic":%q,"stack":%q}`,
				stage, fmt.Sprint(recovered), truncateStack(debug.Stack()))
		}
	}()
	fn()
}

// guardJob is the guard for a stage that reports whether it did work, so a
// panicking job stops that drain loop for this tick rather than spinning.
func guardJob(stage string, fn func() (bool, error)) (processed bool, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf(`{"level":"error","stage":%q,"msg":"worker job panicked","panic":%q,"stack":%q}`,
				stage, fmt.Sprint(recovered), truncateStack(debug.Stack()))
			processed = false
			err = fmt.Errorf("%s panicked: %v", stage, recovered)
		}
	}()
	return fn()
}

func truncateStack(stack []byte) string {
	const max = 4000
	if len(stack) > max {
		return string(stack[:max])
	}
	return string(stack)
}
