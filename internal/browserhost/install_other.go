//go:build !windows

package browserhost

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func hostManifestTargets(home string) []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			filepath.Join(home, "Library/Application Support/Google/Chrome/NativeMessagingHosts", HostName+".json"),
			filepath.Join(home, "Library/Application Support/Chromium/NativeMessagingHosts", HostName+".json"),
			filepath.Join(home, "Library/Application Support/Mozilla/NativeMessagingHosts", HostName+".json"),
		}
	default:
		return []string{
			filepath.Join(home, ".config/google-chrome/NativeMessagingHosts", HostName+".json"),
			filepath.Join(home, ".config/chromium/NativeMessagingHosts", HostName+".json"),
			filepath.Join(home, ".mozilla/native-messaging-hosts", HostName+".json"),
		}
	}
}

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

	targets := hostManifestTargets(home)
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
func Uninstall() (UninstallResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return UninstallResult{}, err
	}
	wasInstalled := IsInstalled()
	var removed []string
	for _, path := range hostManifestTargets(home) {
		if err := os.Remove(path); err == nil {
			removed = append(removed, path)
		}
	}
	return UninstallResult{RemovedPaths: removed, WasInstalled: wasInstalled || len(removed) > 0}, nil
}

// IsInstalled reports whether any host manifest file exists.
func IsInstalled() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	for _, path := range hostManifestTargets(home) {
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}
