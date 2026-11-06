//go:build windows

package browserhost

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// Install registers the native messaging host for Chrome and Firefox (current user).
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

	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return InstallResult{}, fmt.Errorf("LOCALAPPDATA is not set")
	}
	base := filepath.Join(localAppData, "ODM", "native-messaging")
	manifestPath := filepath.Join(base, HostName+".json")
	if err := WriteManifestFile(manifestPath, m); err != nil {
		return InstallResult{}, err
	}

	absManifest, err := filepath.Abs(manifestPath)
	if err != nil {
		return InstallResult{}, err
	}

	keys := []string{
		`Software\Google\Chrome\NativeMessagingHosts\` + HostName,
		`Software\Microsoft\Edge\NativeMessagingHosts\` + HostName,
		`Software\Mozilla\NativeMessagingHosts\` + HostName,
	}
	for _, keyPath := range keys {
		key, _, err := registry.CreateKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
		if err != nil {
			return InstallResult{}, fmt.Errorf("registry %s: %w", keyPath, err)
		}
		if err := key.SetStringValue("", absManifest); err != nil {
			_ = key.Close()
			return InstallResult{}, err
		}
		_ = key.Close()
	}

	return InstallResult{ManifestPaths: []string{absManifest}, HostPath: hostPath}, nil
}

// Uninstall removes host registry entries and the manifest file when present.
func Uninstall() error {
	keys := []string{
		`Software\Google\Chrome\NativeMessagingHosts\` + HostName,
		`Software\Microsoft\Edge\NativeMessagingHosts\` + HostName,
		`Software\Mozilla\NativeMessagingHosts\` + HostName,
	}
	for _, keyPath := range keys {
		_ = registry.DeleteKey(registry.CURRENT_USER, keyPath)
	}
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" {
		_ = os.Remove(filepath.Join(localAppData, "ODM", "native-messaging", HostName+".json"))
	}
	return nil
}

// IsInstalled reports whether a Chrome or Firefox host registry key exists.
func IsInstalled() bool {
	keys := []string{
		`Software\Google\Chrome\NativeMessagingHosts\` + HostName,
		`Software\Mozilla\NativeMessagingHosts\` + HostName,
	}
	for _, keyPath := range keys {
		key, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
		if err == nil {
			_ = key.Close()
			return true
		}
	}
	return false
}
