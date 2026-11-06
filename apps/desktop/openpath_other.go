//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func openFile(path string) error {
	path = filepath.Clean(path)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

func revealInFolder(path string) error {
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	dir := path
	if err == nil && !info.IsDir() {
		dir = filepath.Dir(path)
	} else if err != nil {
		dir = filepath.Dir(path)
	}
	switch runtime.GOOS {
	case "darwin":
		if err == nil && !info.IsDir() {
			return exec.Command("open", "-R", path).Start()
		}
		return exec.Command("open", dir).Start()
	default:
		return exec.Command("xdg-open", dir).Start()
	}
}
