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
	)
	fs.StringVar(&dataDir, "data-dir", "", "ODM data directory")
	fs.StringVar(&mismatch, "on-mismatch", "abort", "resume validator mismatch: abort or restart")
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

	fmt.Fprintf(os.Stderr, "odm: queue running (db %s); Ctrl+C to stop\n", dataPath)
	// wait=true keeps the process alive for newly added tasks until cancelled.
	err = m.RunQueue(ctx, true)
	if err == context.Canceled {
		return nil
	}
	return err
}
