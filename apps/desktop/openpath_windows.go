package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// openFile opens a file with the OS default application.
func openFile(path string) error {
	path = filepath.Clean(path)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	// start "" <path> — empty title arg is required by cmd start.
	cmd := exec.Command("cmd", "/c", "start", "", path)
	return cmd.Start()
}

// revealInFolder selects the file in Explorer (or opens the folder).
func revealInFolder(path string) error {
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		// Fall back to parent directory when the file is gone.
		dir := filepath.Dir(path)
		return exec.Command("explorer", dir).Start()
	}
	if info.IsDir() {
		return exec.Command("explorer", path).Start()
	}
	return exec.Command("explorer", "/select,", path).Start()
}
