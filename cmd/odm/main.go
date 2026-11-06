// Package main is the entry point for the `odm` command-line program.
//
// In Go, an executable program must live in package "main" and define func main().
// When you run `go run ./cmd/odm` or build a binary, the Go toolchain looks for
// this package and starts execution at main().
package main

// import lists other packages this file needs.
//
// - Standard library packages use short paths like "fmt" or "os".
// - There are no unused imports allowed: if you import it, you must use it.
import (
	// context carries deadlines, cancellation signals, and request-scoped values.
	// We use it so Ctrl+C can stop an in-progress download cleanly.
	"context"

	// errors helps compare and inspect error values (errors.Is).
	"errors"

	// fmt formats text for printing (Fprintf, Sprint, ...).
	"fmt"

	// os talks to the operating system: args, stdin/stdout/stderr, exit codes.
	"os"

	// signal lets us listen for OS signals such as Interrupt (Ctrl+C).
	"os/signal"

	// strings has helpers for searching and transforming text.
	"strings"
)

// main is where the program starts.
//
// Go does not use a return code from main(). To set a process exit code,
// call os.Exit(code). Code 0 means success; non-zero means failure.
func main() {
	// os.Args is a []string of command-line arguments.
	// os.Args[0] is the program name/path. Real user args start at index 1.
	// Example: `odm download https://x` → len=3, Args[1]="download".
	if len(os.Args) < 2 {
		// Not enough args: show help on stderr and exit with code 2 (usage error).
		printUsage(os.Stderr)
		os.Exit(2)
	}

	// signal.NotifyContext returns:
	//  1) a context that is cancelled when an Interrupt signal arrives
	//  2) a stop function that releases the signal listener
	//
	// defer stop() runs when main returns (or when the surrounding function ends).
	// defer schedules cleanup to run in LIFO order at function exit.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// switch compares os.Args[1] (the subcommand) against known commands.
	switch os.Args[1] {
	case "version", "--version", "-V":
		// Multiple case values share the same branch (like OR).
		printVersion()

	case "help", "--help", "-h":
		printUsage(os.Stdout)

	case "download", "dl", "get":
		// Pass remaining args after the subcommand name: os.Args[2:].
		// The [2:] syntax is a slice from index 2 to the end.
		if err := runDownload(ctx, os.Args[2:]); err != nil {
			// %v prints the error's message via its Error() method.
			fmt.Fprintf(os.Stderr, "odm: %v\n", err)
			os.Exit(exitCode(err))
		}

	case "task":
		if err := runTask(ctx, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "odm: %v\n", err)
			os.Exit(exitCode(err))
		}

	case "queue":
		if err := runQueue(ctx, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "odm: %v\n", err)
			os.Exit(exitCode(err))
		}

	case "settings":
		if err := runSettings(ctx, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "odm: %v\n", err)
			os.Exit(exitCode(err))
		}

	default:
		// %q prints the string in quoted form, useful for debugging user input.
		fmt.Fprintf(os.Stderr, "odm: unknown command %q\n\n", os.Args[1])
		printUsage(os.Stderr)
		os.Exit(2)
	}
}

// printUsage writes help text to the given file (stdout or stderr).
//
// *os.File is a pointer to an open file handle. os.Stdout and os.Stderr are
// already-open *os.File values provided by the standard library.
func printUsage(w *os.File) {
	// fmt.Fprint writes to any io.Writer. We ignore the byte count and error
	// with `_, _ =` because failing to print help is rare and not actionable here.
	//
	// The string uses raw string literals with backticks `...` so we can include
	// newlines and quotes without escaping.
	_, _ = fmt.Fprint(w, `ODM — Open Download Manager

Usage:
  odm version
  odm download <url> [flags]          One-shot download (no DB)
  odm task add <url> [flags]          Enqueue + run via SQLite queue
  odm task list
  odm task pause|resume|cancel|remove <id>
  odm queue run [--once]              Process queue (Ctrl+C to stop; --once exits when idle)
  odm settings show|set <key> <value>

Shared flags:
      --data-dir <dir>    Database location (default: OS config dir/odm)

Download / task add flags:
  -o, --output <path>     Final file path
      --dir <dir>         Directory when --output is omitted
      --proxy <mode|url>  (one-shot download only) system|off|URL
      --files keep|delete (cancel/remove) whether to delete files

Examples:
  odm download https://example.com/file.bin -o file.bin --proxy off
  odm task add https://example.com/file.bin -o D:\\Downloads\\file.bin
  odm settings set queue.max_active 2
  odm queue run --on-mismatch abort
`)
}

// exitCode maps an error to a process exit status.
//
// Returning different codes helps scripts distinguish "bad usage" from
// "download failed" from "user cancelled".
func exitCode(err error) int {
	if err == nil {
		return 0
	}

	// strings.ToLower makes comparisons case-insensitive.
	msg := strings.ToLower(err.Error())

	switch {
	// errors.Is checks whether err wraps (or equals) a target sentinel error.
	// context.Canceled is set when the context is cancelled (e.g. Ctrl+C).
	case errors.Is(err, context.Canceled):
		return 130 // conventional "terminated by Ctrl+C" style code
	case strings.Contains(msg, "invalid download url"):
		return 2
	case strings.Contains(msg, "invalid destination"):
		return 2
	default:
		return 1
	}
}
