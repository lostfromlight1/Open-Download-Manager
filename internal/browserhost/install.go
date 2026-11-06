package browserhost

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Manifest describes the native messaging host JSON file.
type Manifest struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Path              string   `json:"path"`
	Type              string   `json:"type"`
	AllowedOrigins    []string `json:"allowed_origins,omitempty"`
	AllowedExtensions []string `json:"allowed_extensions,omitempty"`
}

// BuildManifest creates host metadata for Chrome and/or Firefox.
func BuildManifest(hostPath, chromeExtensionID string) (Manifest, error) {
	hostPath = strings.TrimSpace(hostPath)
	if hostPath == "" {
		return Manifest{}, fmt.Errorf("host binary path is required")
	}
	abs, err := filepath.Abs(hostPath)
	if err != nil {
		return Manifest{}, err
	}
	m := Manifest{
		Name:              HostName,
		Description:       "Open Download Manager native messaging host",
		Path:              abs,
		Type:              "stdio",
		AllowedExtensions: []string{FirefoxExtensionID},
	}
	id := strings.TrimSpace(chromeExtensionID)
	if id != "" {
		m.AllowedOrigins = []string{"chrome-extension://" + id + "/"}
	}
	return m, nil
}

// WriteManifestFile writes JSON to path with mode 0644.
func WriteManifestFile(path string, m Manifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

// ResolveHostBinary finds odm-nmhost next to the current executable or on PATH.
func ResolveHostBinary() (string, error) {
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		candidates := []string{
			filepath.Join(dir, "odm-nmhost"),
			filepath.Join(dir, "odm-nmhost.exe"),
		}
		for _, c := range candidates {
			if st, err := os.Stat(c); err == nil && !st.IsDir() {
				return filepath.Abs(c)
			}
		}
	}
	if p, err := exec.LookPath("odm-nmhost"); err == nil {
		return filepath.Abs(p)
	}
	if p, err := exec.LookPath("odm-nmhost.exe"); err == nil {
		return filepath.Abs(p)
	}
	return "", fmt.Errorf("odm-nmhost not found next to the app or on PATH")
}

// InstallOptions configures host registration for the current user.
type InstallOptions struct {
	HostBinaryPath    string
	ChromeExtensionID string
}

// InstallResult reports where manifests were written.
type InstallResult struct {
	ManifestPaths []string
	HostPath      string
}
