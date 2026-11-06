package manager

import (
	"context"
	"fmt"
)

// FileAction is the keep-or-delete choice for cancel/remove.
type FileAction string

const (
	// FileKeep leaves .part / final files on disk.
	FileKeep FileAction = "keep"
	// FileDelete removes .part and the final destination file when present.
	FileDelete FileAction = "delete"
)

// ParseFileAction accepts CLI-friendly aliases.
func ParseFileAction(s string) (FileAction, error) {
	switch s {
	case "keep", "keep-files", "retain":
		return FileKeep, nil
	case "delete", "delete-files", "remove-files":
		return FileDelete, nil
	default:
		return "", fmt.Errorf("file action must be keep or delete, got %q", s)
	}
}

// MismatchAction is the user's choice when remote validators changed.
type MismatchAction string

const (
	// MismatchRestart deletes partial progress and downloads from byte 0.
	MismatchRestart MismatchAction = "restart"
	// MismatchAbort leaves the partial file and fails the attempt.
	MismatchAbort MismatchAction = "abort"
)

// ParseMismatchAction accepts UI/CLI-friendly aliases.
func ParseMismatchAction(s string) (MismatchAction, error) {
	switch s {
	case "restart", "restart-download":
		return MismatchRestart, nil
	case "abort", "keep", "keep-partial":
		return MismatchAbort, nil
	default:
		return "", fmt.Errorf("mismatch action must be restart or abort, got %q", s)
	}
}

// ValidatorMismatch describes a resume conflict for the decision callback.
type ValidatorMismatch struct {
	TaskID          string
	DestinationPath string
	SavedETag       string
	SavedModified   string
	RemoteETag      string
	RemoteModified  string
}

// DecisionHandler lets CLI/UI (or tests) answer user-facing questions.
//
// Returning an error aborts the operation.
// OnResumeMismatch receives ctx so desktop UIs can unblock on shutdown.
type DecisionHandler interface {
	OnCancelFiles(taskID string) (FileAction, error)
	OnRemoveFiles(taskID string) (FileAction, error)
	OnResumeMismatch(ctx context.Context, info ValidatorMismatch) (MismatchAction, error)
}

// StaticDecisions is a non-interactive handler used by flags and tests.
type StaticDecisions struct {
	Cancel   FileAction
	Remove   FileAction
	Mismatch MismatchAction
}

// OnCancelFiles implements DecisionHandler.
func (s StaticDecisions) OnCancelFiles(string) (FileAction, error) {
	if s.Cancel == "" {
		return FileKeep, nil
	}
	return s.Cancel, nil
}

// OnRemoveFiles implements DecisionHandler.
func (s StaticDecisions) OnRemoveFiles(string) (FileAction, error) {
	if s.Remove == "" {
		return FileKeep, nil
	}
	return s.Remove, nil
}

// OnResumeMismatch implements DecisionHandler.
func (s StaticDecisions) OnResumeMismatch(context.Context, ValidatorMismatch) (MismatchAction, error) {
	if s.Mismatch == "" {
		return MismatchAbort, nil
	}
	return s.Mismatch, nil
}
