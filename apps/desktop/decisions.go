package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/lostfromlight1/Open-Download-Manager/internal/manager"
	"github.com/lostfromlight1/Open-Download-Manager/internal/task"
)

// InteractiveDecisions bridges resume-mismatch waits to the React UI via events.
//
// Cancel/Remove from the UI pass explicit FileAction and do not use this path.
type InteractiveDecisions struct {
	sink   EventSink
	needUI func()

	mu      sync.Mutex
	pending map[string]chan manager.MismatchAction // decisionID → response
}

// NewInteractiveDecisions constructs a decision bridge.
func NewInteractiveDecisions(sink EventSink) *InteractiveDecisions {
	return &InteractiveDecisions{
		sink:    sink,
		pending: make(map[string]chan manager.MismatchAction),
	}
}

// OnCancelFiles is unused by the desktop UI (explicit FileAction is passed).
func (d *InteractiveDecisions) OnCancelFiles(string) (manager.FileAction, error) {
	return manager.FileKeep, nil
}

// OnRemoveFiles is unused by the desktop UI (explicit FileAction is passed).
func (d *InteractiveDecisions) OnRemoveFiles(string) (manager.FileAction, error) {
	return manager.FileKeep, nil
}

// SetNeedUI registers a callback invoked when a decision requires the UI
// (e.g. show the window if it was hidden to the tray).
func (d *InteractiveDecisions) SetNeedUI(fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.needUI = fn
}

// OnResumeMismatch emits a decision-required event and waits for ResolveDecision.
func (d *InteractiveDecisions) OnResumeMismatch(ctx context.Context, info manager.ValidatorMismatch) (manager.MismatchAction, error) {
	id := task.NewID()
	ch := make(chan manager.MismatchAction, 1)

	d.mu.Lock()
	d.pending[id] = ch
	needUI := d.needUI
	d.mu.Unlock()

	defer func() {
		d.mu.Lock()
		delete(d.pending, id)
		d.mu.Unlock()
	}()

	if needUI != nil {
		needUI()
	}

	if d.sink != nil {
		d.sink.Emit(EventDecisionRequired, DecisionRequiredDTO{
			DecisionID:      id,
			TaskID:          info.TaskID,
			DestinationPath: info.DestinationPath,
			SavedETag:       info.SavedETag,
			SavedModified:   info.SavedModified,
			RemoteETag:      info.RemoteETag,
			RemoteModified:  info.RemoteModified,
			Kind:            "resume_mismatch",
		})
	}

	select {
	case <-ctx.Done():
		return "", fmt.Errorf("%w: decision cancelled: %v", context.Canceled, ctx.Err())
	case action := <-ch:
		return action, nil
	}
}

// Resolve delivers a user choice for a pending decision ID.
func (d *InteractiveDecisions) Resolve(decisionID, choice string) error {
	action, err := manager.ParseMismatchAction(choice)
	if err != nil {
		return err
	}

	d.mu.Lock()
	ch, ok := d.pending[decisionID]
	d.mu.Unlock()
	if !ok {
		return fmt.Errorf("stale or unknown decision id")
	}

	select {
	case ch <- action:
		return nil
	default:
		return fmt.Errorf("decision already resolved")
	}
}

// PendingCount is for tests.
func (d *InteractiveDecisions) PendingCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.pending)
}
