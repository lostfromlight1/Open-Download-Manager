package main

import "runtime"

// openFile opens a file with the OS default application.
func openFile(path string) error {
	cmd, err := planOpenCommand(runtime.GOOS, path)
	if err != nil {
		return err
	}
	return startFileCommand(cmd)
}

// revealInFolder selects the file in Explorer (or opens the folder).
func revealInFolder(path string) error {
	return startFileCommand(planRevealCommand(runtime.GOOS, path))
}
