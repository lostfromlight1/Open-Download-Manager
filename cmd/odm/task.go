package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lostfromlight1/Open-Download-Manager/internal/manager"
	"github.com/lostfromlight1/Open-Download-Manager/internal/task"
)

// runTask dispatches: odm task <add|list|pause|resume|cancel|remove> ...
func runTask(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: odm task <add|list|pause|resume|cancel|remove>")
	}
	switch args[0] {
	case "add":
		return runTaskAdd(ctx, args[1:])
	case "list", "ls":
		return runTaskList(ctx, args[1:])
	case "pause":
		return runTaskPause(ctx, args[1:])
	case "resume":
		return runTaskResume(ctx, args[1:])
	case "cancel":
		return runTaskCancel(ctx, args[1:])
	case "remove", "rm":
		return runTaskRemove(ctx, args[1:])
	default:
		return fmt.Errorf("unknown task subcommand %q", args[0])
	}
}

func runTaskAdd(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("task add", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		output  string
		dir     string
		dataDir string
		start   bool
	)
	fs.StringVar(&output, "o", "", "final output file path")
	fs.StringVar(&output, "output", "", "final output file path")
	fs.StringVar(&dir, "dir", "", "directory when -o is omitted")
	fs.StringVar(&dataDir, "data-dir", "", "ODM data directory (database location)")
	fs.BoolVar(&start, "start", true, "run the queue until this add is processed")

	args = reorderDownloadArgs(args)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("task add requires exactly one URL")
	}
	rawURL := fs.Arg(0)

	dest, err := resolveCLIDestination(rawURL, output, dir)
	if err != nil {
		return err
	}
	// Absolute path so the DB stores a stable location.
	abs, err := filepath.Abs(dest)
	if err != nil {
		return err
	}

	m, _, dataPath, closeFn, err := openManagerFromFlags(dataDir, nil)
	if err != nil {
		return err
	}
	defer closeFn()

	tk, err := m.Add(ctx, manager.AddOptions{URL: rawURL, Destination: abs})
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "queued %s -> %s (db %s)\n", tk.ID, tk.DestinationPath, dataPath)

	if !start {
		return nil
	}
	// Run until idle so a single `task add` behaves like a managed download.
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	errCh := make(chan error, 1)
	go func() { errCh <- m.RunQueue(runCtx, false) }()

	deadline := time.Now().Add(24 * time.Hour)
	for time.Now().Before(deadline) {
		got, gerr := m.Get(ctx, tk.ID)
		if gerr != nil {
			return gerr
		}
		switch got.State {
		case task.StateCompleted:
			fmt.Fprintf(os.Stdout, "completed %s (%d bytes)\n", got.DestinationPath, got.DownloadedBytes)
			cancel()
			<-errCh
			return nil
		case task.StateFailed:
			cancel()
			<-errCh
			return fmt.Errorf("task failed: %s", got.ErrorMessage)
		case task.StateCancelled:
			cancel()
			<-errCh
			return fmt.Errorf("task cancelled")
		}
		select {
		case <-ctx.Done():
			cancel()
			<-errCh
			return ctx.Err()
		case err := <-errCh:
			// Queue became idle; re-check state once more.
			got, _ = m.Get(ctx, tk.ID)
			if got.State == task.StateCompleted {
				fmt.Fprintf(os.Stdout, "completed %s (%d bytes)\n", got.DestinationPath, got.DownloadedBytes)
				return err
			}
			if got.State == task.StateFailed {
				return fmt.Errorf("task failed: %s", got.ErrorMessage)
			}
			// Restart queue if still queued (race with scheduler).
			if got.State == task.StateQueued {
				go func() { errCh <- m.RunQueue(runCtx, false) }()
				continue
			}
			return err
		case <-time.After(200 * time.Millisecond):
		}
	}
	return fmt.Errorf("timed out waiting for task")
}

func runTaskList(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("task list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var dataDir string
	fs.StringVar(&dataDir, "data-dir", "", "ODM data directory")
	if err := fs.Parse(args); err != nil {
		return err
	}

	m, _, _, closeFn, err := openManagerFromFlags(dataDir, nil)
	if err != nil {
		return err
	}
	defer closeFn()

	tasks, err := m.List(ctx)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		fmt.Fprintln(os.Stdout, "(no tasks)")
		return nil
	}
	for _, tk := range tasks {
		total := "?"
		if tk.TotalBytes != nil {
			total = fmt.Sprintf("%d", *tk.TotalBytes)
		}
		fmt.Fprintf(os.Stdout, "%s  %-12s  %d/%s  %s\n", shortID(tk.ID), tk.State, tk.DownloadedBytes, total, tk.DestinationPath)
	}
	return nil
}

func runTaskPause(ctx context.Context, args []string) error {
	id, dataDir, err := parseIDArgs("task pause", args)
	if err != nil {
		return err
	}
	m, _, _, closeFn, err := openManagerFromFlags(dataDir, nil)
	if err != nil {
		return err
	}
	defer closeFn()
	return m.Pause(ctx, id)
}

func runTaskResume(ctx context.Context, args []string) error {
	id, dataDir, err := parseIDArgs("task resume", args)
	if err != nil {
		return err
	}
	m, _, _, closeFn, err := openManagerFromFlags(dataDir, nil)
	if err != nil {
		return err
	}
	defer closeFn()
	if err := m.Resume(ctx, id); err != nil {
		return err
	}
	// Kick the queue once so resume actually starts.
	return m.RunQueue(ctx, false)
}

func runTaskCancel(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("task cancel", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		dataDir string
		files   string
	)
	fs.StringVar(&dataDir, "data-dir", "", "ODM data directory")
	fs.StringVar(&files, "files", "keep", "keep or delete partial/final files")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("task cancel requires a task id")
	}
	action, err := manager.ParseFileAction(files)
	if err != nil {
		return err
	}
	m, _, _, closeFn, err := openManagerFromFlags(dataDir, manager.StaticDecisions{Cancel: action})
	if err != nil {
		return err
	}
	defer closeFn()
	return m.Cancel(ctx, expandID(ctx, m, fs.Arg(0)), action)
}

func runTaskRemove(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("task remove", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		dataDir string
		files   string
	)
	fs.StringVar(&dataDir, "data-dir", "", "ODM data directory")
	fs.StringVar(&files, "files", "keep", "keep or delete downloaded files")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("task remove requires a task id")
	}
	action, err := manager.ParseFileAction(files)
	if err != nil {
		return err
	}
	m, _, _, closeFn, err := openManagerFromFlags(dataDir, manager.StaticDecisions{Remove: action})
	if err != nil {
		return err
	}
	defer closeFn()
	return m.Remove(ctx, expandID(ctx, m, fs.Arg(0)), action)
}

func parseIDArgs(name string, args []string) (id, dataDir string, err error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&dataDir, "data-dir", "", "ODM data directory")
	if err := fs.Parse(args); err != nil {
		return "", "", err
	}
	if fs.NArg() != 1 {
		return "", "", fmt.Errorf("%s requires a task id", name)
	}
	return fs.Arg(0), dataDir, nil
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

// expandID allows using an 8-char prefix when it uniquely matches.
func expandID(ctx context.Context, m *manager.Manager, idOrPrefix string) string {
	idOrPrefix = strings.TrimSpace(idOrPrefix)
	if len(idOrPrefix) >= 32 {
		return idOrPrefix
	}
	tasks, err := m.List(ctx)
	if err != nil {
		return idOrPrefix
	}
	var match string
	for _, tk := range tasks {
		if tk.ID == idOrPrefix || strings.HasPrefix(tk.ID, idOrPrefix) {
			if match != "" && match != tk.ID {
				return idOrPrefix // ambiguous; let manager return not found/wrong id
			}
			match = tk.ID
		}
	}
	if match != "" {
		return match
	}
	return idOrPrefix
}
