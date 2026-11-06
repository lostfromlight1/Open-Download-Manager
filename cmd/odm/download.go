package main

import (
	// context is passed through so Ctrl+C cancels engine.Download.
	"context"

	// flag parses command-line options (-o, --proxy, ...).
	"flag"

	// fmt builds errors and prints the success line.
	"fmt"

	// net/url parses the download URL when deriving a default filename.
	"net/url"

	// os provides Stderr/Stdout for progress vs final messages.
	"os"

	// path works on slash-separated URL paths (not OS paths).
	"path"

	// path/filepath joins OS-specific destination paths.
	"path/filepath"

	// strings.TrimSpace cleans user flag values.
	"strings"

	// time.Duration is used for --stall-timeout / --connect-timeout flags.
	"time"

	"github.com/lostfromlight1/Open-Download-Manager/internal/engine"
	"github.com/lostfromlight1/Open-Download-Manager/internal/netx"
)

// runDownload parses CLI flags and calls the reusable download engine.
//
// Keeping CLI parsing here (and download logic in internal/engine) means the
// desktop app can later call the same engine without going through flags.
func runDownload(ctx context.Context, args []string) error {
	// flag.NewFlagSet creates a private flag parser for this subcommand.
	// ContinueOnError means Parse returns an error instead of calling os.Exit.
	fs := flag.NewFlagSet("download", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	// var (...) declares a group of variables.
	// These hold the parsed flag values.
	var (
		output         string
		dir            string
		proxy          string
		forceRestart   bool
		maxRetries     int
		stallTimeout   time.Duration
		connectTimeout time.Duration
	)

	// StringVar binds a flag name to a *string variable.
	// The third argument is the default; the fourth is help text.
	// Registering both -o and --output updates the same variable.
	fs.StringVar(&output, "o", "", "final output file path")
	fs.StringVar(&output, "output", "", "final output file path")
	fs.StringVar(&dir, "dir", "", "output directory when -o is omitted")
	fs.StringVar(&proxy, "proxy", "system", "system, off, or a proxy URL")
	fs.BoolVar(&forceRestart, "force-restart", false, "ignore partial files and restart")
	fs.IntVar(&maxRetries, "max-retries", 5, "retries after the first attempt")

	// Duration flags accept values like "30s", "2m", "500ms".
	fs.DurationVar(&stallTimeout, "stall-timeout", 30*time.Second, "max idle time between received bytes")
	fs.DurationVar(&connectTimeout, "connect-timeout", 30*time.Second, "TCP connect timeout")

	// Go's flag.Parse stops at the first non-flag argument. Users often write:
	//   odm download <url> -o file.exe
	// which would otherwise leave "-o" and "file.exe" as extra positionals.
	// Reorder so flags come first, then the URL.
	args = reorderDownloadArgs(args)

	// Parse reads args and fills the variables above.
	if err := fs.Parse(args); err != nil {
		return err
	}

	// NArg is the count of leftover non-flag arguments (the URL).
	if fs.NArg() != 1 {
		// fmt.Errorf creates an error value. %w is not needed here because we
		// are not wrapping another error.
		return fmt.Errorf("download requires exactly one URL")
	}
	rawURL := fs.Arg(0) // first leftover argument

	// ParseProxyMode converts user text into a typed proxy mode + optional URL.
	mode, proxyURL, err := netx.ParseProxyMode(proxy)
	if err != nil {
		return err
	}

	dest, err := resolveCLIDestination(rawURL, output, dir)
	if err != nil {
		return err
	}

	// engine.Options is a struct literal. Field names are written explicitly
	// so the call stays readable as the Options type grows.
	result, err := engine.Download(ctx, engine.Options{
		URL:                   rawURL,
		Destination:           dest,
		ForceRestart:          forceRestart,
		Proxy:                 netx.ProxyConfig{Mode: mode, URL: proxyURL},
		MaxRetries:            maxRetries,
		StallTimeout:          stallTimeout,
		ConnectTimeout:        connectTimeout,
		ProgressWriter:        os.Stderr, // progress on stderr; final message on stdout
		ResponseHeaderTimeout: 60 * time.Second,
	})
	if err != nil {
		return err
	}

	// Success message goes to stdout so it can be piped separately from progress.
	fmt.Fprintf(os.Stdout, "saved %s (%d bytes, %d attempt(s))\n", result.Destination, result.BytesWritten, result.Attempts)
	return nil
}

// reorderDownloadArgs moves positional args (the URL) after flags so flag.Parse
// still sees -o / --proxy when the user puts the URL first.
//
// Flags that take a value are listed below. Bool flags must NOT pull the next
// token as a value (otherwise a URL after --force-restart would be swallowed).
func reorderDownloadArgs(args []string) []string {
	valueFlags := map[string]bool{
		"o": true, "output": true, "dir": true, "proxy": true,
		"max-retries": true, "stall-timeout": true, "connect-timeout": true,
	}

	var flags, positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		// Non-flags (and lone "-") are positionals — usually the download URL.
		if a == "-" || !strings.HasPrefix(a, "-") {
			positionals = append(positionals, a)
			continue
		}

		flags = append(flags, a)

		// --name=value already includes the value in this token.
		name := strings.TrimLeft(a, "-")
		if eq := strings.IndexByte(name, '='); eq >= 0 {
			name = name[:eq]
			continue
		}

		if !valueFlags[name] {
			continue // bool flag such as --force-restart
		}
		// Take the next token as this flag's value when present.
		if i+1 < len(args) && (args[i+1] == "-" || !strings.HasPrefix(args[i+1], "-")) {
			i++
			flags = append(flags, args[i])
		}
	}

	return append(flags, positionals...)
}

// resolveCLIDestination decides the final file path from -o / --dir / URL.
//
// Multiple return values are idiomatic Go: (value, error).
// By convention, if error != nil, the value should be ignored.
func resolveCLIDestination(rawURL, output, dir string) (string, error) {
	output = strings.TrimSpace(output)
	dir = strings.TrimSpace(dir)

	if output != "" && dir != "" {
		return "", fmt.Errorf("use either -o/--output or --dir, not both")
	}
	if output != "" {
		return output, nil // nil error means success
	}

	// url.Parse understands scheme/host/path/query fragments.
	u, err := url.Parse(rawURL)
	if err != nil {
		// %w wraps err so callers can use errors.Is / errors.As on the cause.
		return "", fmt.Errorf("parse URL: %w", err)
	}

	// path.Base works on URL paths (always slash-separated), not OS paths.
	// filepath.Base would use backslashes on Windows, which is wrong for URLs.
	name := path.Base(u.Path)
	if name == "" || name == "." || name == "/" {
		name = "download.bin"
	}

	if dir == "" {
		// Relative path in the current working directory.
		return name, nil
	}

	// filepath.Join builds OS-correct paths (backslash on Windows).
	return filepath.Join(dir, name), nil
}
