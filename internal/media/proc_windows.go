//go:build windows

package media

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// hideHelperConsole prevents yt-dlp/ffmpeg from flashing a black cmd window
// when launched from the GUI desktop app.
func hideHelperConsole(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}
