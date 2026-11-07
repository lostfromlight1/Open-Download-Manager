//go:build !windows

package media

import "os/exec"

func hideHelperConsole(cmd *exec.Cmd) {
	// No-op: Unix helpers do not open a separate console window.
}
