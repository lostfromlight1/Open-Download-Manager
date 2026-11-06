package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// fileCommand is the OS process that should be started for open/reveal.
type fileCommand struct {
	Name string
	Args []string
}

// prepareOpenPath cleans path and requires that it exists (file or directory).
func prepareOpenPath(path string) (string, error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return "", fmt.Errorf("open file: empty path")
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	return path, nil
}

// planOpenCommand returns the command used to open a path with the default app.
func planOpenCommand(goos, path string) (fileCommand, error) {
	path, err := prepareOpenPath(path)
	if err != nil {
		return fileCommand{}, err
	}
	switch goos {
	case "windows":
		// start "" <path> — empty title arg is required by cmd start.
		return fileCommand{Name: "cmd", Args: []string{"/c", "start", "", path}}, nil
	case "darwin":
		return fileCommand{Name: "open", Args: []string{path}}, nil
	default:
		return fileCommand{Name: "xdg-open", Args: []string{path}}, nil
	}
}

// planRevealCommand returns the command used to show a path in the file manager.
// When the file is missing, the parent directory is opened instead.
func planRevealCommand(goos, path string) fileCommand {
	path = filepath.Clean(path)
	info, err := os.Stat(path)

	switch goos {
	case "windows":
		if err != nil {
			return fileCommand{Name: "explorer", Args: []string{filepath.Dir(path)}}
		}
		if info.IsDir() {
			return fileCommand{Name: "explorer", Args: []string{path}}
		}
		return fileCommand{Name: "explorer", Args: []string{"/select,", path}}
	case "darwin":
		if err == nil && !info.IsDir() {
			return fileCommand{Name: "open", Args: []string{"-R", path}}
		}
		dir := filepath.Dir(path)
		if err == nil && info.IsDir() {
			dir = path
		}
		return fileCommand{Name: "open", Args: []string{dir}}
	default:
		dir := filepath.Dir(path)
		if err == nil && info.IsDir() {
			dir = path
		}
		return fileCommand{Name: "xdg-open", Args: []string{dir}}
	}
}

func startFileCommand(cmd fileCommand) error {
	return exec.Command(cmd.Name, cmd.Args...).Start()
}
