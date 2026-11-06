//go:build !windows

package browserhost

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Install registers native messaging host manifests for Chrome and Firefox.
func Install(opts InstallOptions) (InstallResult, error) {
	hostPath := opts.HostBinaryPath
	if hostPath == "" {
		var err error
		hostPath, err = ResolveHostBinary()
		if err != nil {
			return InstallResult{}, err
		}
	}
	m, err := BuildManifest(hostPath, opts.ChromeExtensionID)
	if err != nil {
		return InstallResult{}, err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return InstallResult{}, err
	}

	var targets []string
	switch runtime.GOOS {
	case "darwin":
		targets = []string{
			filepath.Join(home, "Library/Application Support/Google/Chrome/NativeMessagingHosts", HostName+".json"),
			filepath.Join(home, "Library/Application Support/Chromium/NativeMessagingHosts", HostName+".json"),
			filepath.Join(home, "Library/Application Support/Mozilla/NativeMessagingHosts", HostName+".json"),
		}
	default: // linux and others
		targets = []string{
			filepath.Join(home, ".config/google-chrome/NativeMessagingHosts", HostName+".json"),
			filepath.Join(home, ".config/chromium/NativeMessagingHosts", HostName+".json"),
			filepath.Join(home, ".mozilla/native-messaging-hosts", HostName+".json"),
		}
	}

	written := make([]string, 0, len(targets))
	for _, path := range targets {
		if err := WriteManifestFile(path, m); err != nil {
			return InstallResult{}, fmt.Errorf("write %s: %w", path, err)
		}
		abs, _ := filepath.Abs(path)
		written = append(written, abs)
	}
	return InstallResult{ManifestPaths: written, HostPath: hostPath}, nil
}

// Uninstall removes host manifests for Chrome and Firefox.
func Uninstall() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	var targets []string
	switch runtime.GOOS {
	case "darwin":
		targets = []string{
			filepath.Join(home, "Library/Application Support/Google/Chrome/NativeMessagingHosts", HostName+".json"),
			filepath.Join(home, "Library/Application Support/Chromium/NativeMessagingHosts", HostName+".json"),
			filepath.Join(home, "Library/Application Support/Mozilla/NativeMessagingHosts", HostName+".json"),
		}
	default:
		targets = []string{
			filepath.Join(home, ".config/google-chrome/NativeMessagingHosts", HostName+".json"),
			filepath.Join(home, ".config/chromium/NativeMessagingHosts", HostName+".json"),
			filepath.Join(home, ".mozilla/native-messaging-hosts", HostName+".json"),
		}
	}
	for _, path := range targets {
		_ = os.Remove(path)
	}
	return nil
}

// IsInstalled reports whether any host manifest file exists.
func IsInstalled() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	candidates := []string{
		filepath.Join(home, "Library/Application Support/Google/Chrome/NativeMessagingHosts", HostName+".json"),
		filepath.Join(home, ".config/google-chrome/NativeMessagingHosts", HostName+".json"),
		filepath.Join(home, ".mozilla/native-messaging-hosts", HostName+".json"),
		filepath.Join(home, "Library/Application Support/Mozilla/NativeMessagingHosts", HostName+".json"),
	}
	for _, path := range candidates {
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}
