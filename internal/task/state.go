// Package task defines the download task model and its explicit state machine.
//
// Keeping lifecycle rules here (instead of scattered if-statements) makes
// illegal transitions fail loudly and keeps the manager easier to reason about.
package task

import "fmt"

// State is a typed string for task lifecycle values.
//
// Using a named type (not plain string) documents intent at API boundaries and
// helps catch accidental typos when comparing states.
type State string

// Explicit lifecycle states from docs/ARCHITECTURE.md.
//
// Exported constants start with an Uppercase letter so other packages can use
// task.StateQueued, task.StateDownloading, etc.
const (
	StateQueued      State = "queued"      // waiting for a free download slot
	StateInspecting  State = "inspecting"  // checking remote validators / headers
	StateDownloading State = "downloading" // bytes are being written
	StatePaused      State = "paused"      // user paused; partial file may remain
	StateFinalizing  State = "finalizing"  // renaming .part → final path
	StateCompleted   State = "completed"   // finished successfully
	StateFailed      State = "failed"      // stopped with an error
	StateCancelled   State = "cancelled"   // stopped by user cancel
)

// CanTransition reports whether moving from → to is allowed.
//
// Returning bool (instead of panicking) lets callers decide how to surface the
// error. The manager wraps false results as ErrInvalidTransition.
func CanTransition(from, to State) bool {
	if from == to {
		return true // no-op transition is always fine
	}

	// allowed[from] is the set of states reachable in one step from `from`.
	// map[State]map[State]bool is a nested map used as a set (value=true).
	allowed := map[State]map[State]bool{
		StateQueued: {
			StateInspecting: true,
			StatePaused:     true, // user paused before a worker picked it up
			StateCancelled:  true,
		},
		StateInspecting: {
			StateDownloading: true,
			StatePaused:      true,
			StateFailed:      true,
			StateCancelled:   true,
		},
		StateDownloading: {
			StateFinalizing: true,
			StatePaused:     true,
			StateFailed:     true,
			StateCancelled:  true,
		},
		StatePaused: {
			StateInspecting:  true, // resume re-checks validators
			StateDownloading: true, // resume may skip inspect when safe
			StateCancelled:   true,
		},
		StateFinalizing: {
			StateCompleted: true,
			StateFailed:    true,
		},
		StateFailed: {
			StateInspecting: true, // retry
			StateCancelled:  true,
		},
		StateCompleted: {
			// Terminal for normal flow; remove deletes the row instead.
		},
		StateCancelled: {
			StateInspecting: true, // user may re-queue a cancelled task
		},
	}

	next, ok := allowed[from]
	if !ok {
		return false
	}
	return next[to]
}

// Transition validates and returns the new state.
//
// Named result style is optional; here we return (State, error) explicitly.
func Transition(from, to State) (State, error) {
	if !CanTransition(from, to) {
		return from, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, from, to)
	}
	return to, nil
}

// IsTerminal reports states that the queue scheduler should not auto-start.
func IsTerminal(s State) bool {
	return s == StateCompleted || s == StateCancelled
}

// IsActive reports states that hold a live download worker slot.
func IsActive(s State) bool {
	return s == StateInspecting || s == StateDownloading || s == StateFinalizing
}
