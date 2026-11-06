package task

import "testing"

func TestCanTransitionHappyPath(t *testing.T) {
	t.Parallel()

	steps := []struct {
		from, to State
	}{
		{StateQueued, StateInspecting},
		{StateInspecting, StateDownloading},
		{StateDownloading, StateFinalizing},
		{StateFinalizing, StateCompleted},
	}
	for _, step := range steps {
		if !CanTransition(step.from, step.to) {
			t.Fatalf("expected allow %s -> %s", step.from, step.to)
		}
	}
}

func TestCanTransitionRejectsIllegal(t *testing.T) {
	t.Parallel()

	if CanTransition(StateCompleted, StateDownloading) {
		t.Fatal("completed should not go back to downloading")
	}
	if CanTransition(StateQueued, StateCompleted) {
		t.Fatal("queued should not jump to completed")
	}
}

func TestTransitionReturnsError(t *testing.T) {
	t.Parallel()

	_, err := Transition(StateCompleted, StatePaused)
	if err == nil {
		t.Fatal("expected error")
	}
}
