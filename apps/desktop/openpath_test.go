package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareOpenPathRejectsMissingAndEmpty(t *testing.T) {
	t.Parallel()

	if _, err := prepareOpenPath(""); err == nil {
		t.Fatal("expected error for empty path")
	}
	if _, err := prepareOpenPath(filepath.Join(t.TempDir(), "missing.bin")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestPrepareOpenPathCleansExisting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "a.bin")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := prepareOpenPath(filepath.Join(dir, ".", "a.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Clean(file) {
		t.Fatalf("got %q want %q", got, filepath.Clean(file))
	}
}

func TestPlanOpenCommand(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "a.bin")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	win, err := planOpenCommand("windows", file)
	if err != nil {
		t.Fatal(err)
	}
	if win.Name != "cmd" || len(win.Args) < 4 || win.Args[3] != file {
		t.Fatalf("windows open: %+v", win)
	}

	mac, err := planOpenCommand("darwin", file)
	if err != nil {
		t.Fatal(err)
	}
	if mac.Name != "open" || len(mac.Args) != 1 || mac.Args[0] != file {
		t.Fatalf("darwin open: %+v", mac)
	}

	linux, err := planOpenCommand("linux", file)
	if err != nil {
		t.Fatal(err)
	}
	if linux.Name != "xdg-open" || len(linux.Args) != 1 || linux.Args[0] != file {
		t.Fatalf("linux open: %+v", linux)
	}
}

func TestPlanRevealCommandMissingFallsBackToParent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	missing := filepath.Join(dir, "gone.bin")

	win := planRevealCommand("windows", missing)
	if win.Name != "explorer" || len(win.Args) != 1 || win.Args[0] != dir {
		t.Fatalf("windows missing: %+v", win)
	}

	mac := planRevealCommand("darwin", missing)
	if mac.Name != "open" || len(mac.Args) != 1 || mac.Args[0] != dir {
		t.Fatalf("darwin missing: %+v", mac)
	}

	linux := planRevealCommand("linux", missing)
	if linux.Name != "xdg-open" || len(linux.Args) != 1 || linux.Args[0] != dir {
		t.Fatalf("linux missing: %+v", linux)
	}
}

func TestPlanRevealCommandSelectsExistingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "a.bin")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	win := planRevealCommand("windows", file)
	if win.Name != "explorer" || len(win.Args) != 2 || win.Args[0] != "/select," || win.Args[1] != file {
		t.Fatalf("windows select: %+v", win)
	}

	mac := planRevealCommand("darwin", file)
	if mac.Name != "open" || len(mac.Args) != 2 || mac.Args[0] != "-R" || mac.Args[1] != file {
		t.Fatalf("darwin reveal: %+v", mac)
	}

	linux := planRevealCommand("linux", file)
	if linux.Name != "xdg-open" || len(linux.Args) != 1 || linux.Args[0] != dir {
		t.Fatalf("linux reveal: %+v", linux)
	}
}

func TestPlanRevealCommandOpensDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	for _, goos := range []string{"windows", "darwin", "linux"} {
		cmd := planRevealCommand(goos, dir)
		if len(cmd.Args) != 1 || cmd.Args[0] != dir {
			t.Fatalf("%s dir reveal: %+v", goos, cmd)
		}
		switch goos {
		case "windows":
			if cmd.Name != "explorer" {
				t.Fatalf("windows name: %q", cmd.Name)
			}
		case "darwin":
			if cmd.Name != "open" {
				t.Fatalf("darwin name: %q", cmd.Name)
			}
		default:
			if cmd.Name != "xdg-open" {
				t.Fatalf("linux name: %q", cmd.Name)
			}
		}
	}
}

func TestPlanOpenCommandRejectsTraversalStyleMissing(t *testing.T) {
	t.Parallel()

	// Cleaned path that does not exist must still fail before exec.
	bad := filepath.Join(t.TempDir(), "..", "nope-does-not-exist-"+strings.ReplaceAll(t.Name(), "/", "_"))
	if _, err := planOpenCommand("linux", bad); err == nil {
		t.Fatal("expected missing cleaned path to fail")
	}
}
