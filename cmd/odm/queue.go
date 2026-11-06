package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/lostfromlight1/Open-Download-Manager/internal/manager"
)

// runQueue handles: odm queue run
func runQueue(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: odm queue run")
	}
	switch args[0] {
	case "run":
		return runQueueRun(ctx, args[1:])
	default:
		return fmt.Errorf("unknown queue subcommand %q", args[0])
	}
}

func runQueueRun(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("queue run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		dataDir  string
		mismatch string
		once     bool
	)
	fs.StringVar(&dataDir, "data-dir", "", "ODM data directory")
	fs.StringVar(&mismatch, "on-mismatch", "abort", "resume validator mismatch: abort or restart")
	// --once exits when the queue is empty instead of waiting forever.
	fs.BoolVar(&once, "once", false, "exit when no queued/active tasks remain")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var mismatchAction manager.MismatchAction
	switch mismatch {
	case "abort":
		mismatchAction = manager.MismatchAbort
	case "restart":
		mismatchAction = manager.MismatchRestart
	default:
		return fmt.Errorf("on-mismatch must be abort or restart")
	}

	m, _, dataPath, closeFn, err := openManagerFromFlags(dataDir, manager.StaticDecisions{
		Cancel:   manager.FileKeep,
		Remove:   manager.FileKeep,
		Mismatch: mismatchAction,
	})
	if err != nil {
		return err
	}
	defer closeFn()

	if once {
		fmt.Fprintf(os.Stderr, "odm: draining queue once (db %s)\n", dataPath)
		err = m.RunQueue(ctx, false)
	} else {
		// wait=true is intentional: this is a long-running worker process.
		fmt.Fprintf(os.Stderr, "odm: queue idle/running (db %s)\n", dataPath)
		fmt.Fprintf(os.Stderr, "odm: waiting for queued tasks; press Ctrl+C to stop (or use --once to exit when idle)\n")
		err = m.RunQueue(ctx, true)
	}
	if err == context.Canceled {
		return nil
	}
	return err
}
