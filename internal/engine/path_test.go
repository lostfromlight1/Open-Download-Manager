package engine

// Internal tests (same package name "engine") can call unexported helpers
// like validateURL and filenameFromURL.
//
// File name convention: *_test.go files are only compiled during `go test`,
// never into the normal library/binary build.

import (
	"net/url"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidateURL(t *testing.T) {
	t.Parallel() // allow this test to run beside others

	if _, err := validateURL("ftp://example.com/a"); err == nil {
		t.Fatal("expected unsupported scheme error")
	}
	if _, err := validateURL("not a url"); err == nil {
		t.Fatal("expected parse error")
	}
	u, err := validateURL("https://example.com/file.bin")
	if err != nil || u.Host != "example.com" {
		// %#v prints a Go-syntax representation (useful for structs/pointers).
		t.Fatalf("got %#v err=%v", u, err)
	}
}

func TestFilenameFromURL(t *testing.T) {
	t.Parallel()

	u, _ := url.Parse("https://example.com/path/x")
	// Override Path after parsing so the test does not depend on how the
	// parser normalizes spaces in the original string.
	u.Path = "/path/my file.bin"
	if got := filenameFromURL(u); got != "my file.bin" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveDestinationRejectsDirectorySuffix(t *testing.T) {
	t.Parallel()
	// runtime.GOOS is "windows", "linux", "darwin", etc.
	if runtime.GOOS == "windows" {
		_, err := resolveDestination(`C:\tmp\`, &url.URL{Path: "/a.bin"})
		if err == nil {
			t.Fatal("expected error")
		}
		return
	}
	_, err := resolveDestination("/tmp/", &url.URL{Path: "/a.bin"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPartPath(t *testing.T) {
	t.Parallel()
	got := partPath(filepath.Join("a", "b.bin"))
	if got != filepath.Join("a", "b.bin")+".part" {
		t.Fatalf("got %q", got)
	}
}
