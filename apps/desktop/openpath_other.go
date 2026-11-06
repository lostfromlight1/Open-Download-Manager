//go:build !windows

package main

import "runtime"

func openFile(path string) error {
	cmd, err := planOpenCommand(runtime.GOOS, path)
	if err != nil {
		return err
	}
	return startFileCommand(cmd)
}

func revealInFolder(path string) error {
	return startFileCommand(planRevealCommand(runtime.GOOS, path))
}
