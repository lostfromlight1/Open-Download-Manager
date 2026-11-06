package task

import "errors"

// Sentinel errors for the task package.
//
// Callers match with errors.Is(err, task.ErrInvalidTransition).
var (
	ErrInvalidTransition = errors.New("invalid task state transition")
	ErrNotFound          = errors.New("task not found")
	ErrConflict          = errors.New("task conflict")
	ErrDecisionRequired  = errors.New("user decision required")
)
