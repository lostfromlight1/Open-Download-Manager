package main

import (
	"reflect"
	"testing"
)

func TestReorderDownloadArgsURLBeforeFlags(t *testing.T) {
	t.Parallel()

	in := []string{
		"https://desktop-release.notion-static.com/notion-windows-installer.exe",
		"-o", "notion-windows-installer.exe",
		"--proxy", "off",
	}
	want := []string{
		"-o", "notion-windows-installer.exe",
		"--proxy", "off",
		"https://desktop-release.notion-static.com/notion-windows-installer.exe",
	}
	got := reorderDownloadArgs(in)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestReorderDownloadArgsFlagsFirstUnchanged(t *testing.T) {
	t.Parallel()

	in := []string{"-o", "out.bin", "--proxy", "off", "https://example.com/a.bin"}
	got := reorderDownloadArgs(in)
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("got %#v", got)
	}
}

func TestReorderDownloadArgsForceRestartDoesNotSwallowURL(t *testing.T) {
	t.Parallel()

	in := []string{"--force-restart", "https://example.com/a.bin", "-o", "a.bin"}
	want := []string{"--force-restart", "-o", "a.bin", "https://example.com/a.bin"}
	got := reorderDownloadArgs(in)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}
