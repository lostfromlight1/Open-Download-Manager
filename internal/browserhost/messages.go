package browserhost

import (
	"fmt"
	"strings"
)

// UninstallResult summarizes what host registration cleanup removed.
type UninstallResult struct {
	RemovedPaths []string
	WasInstalled bool
}

// FormatInstallMessage is the user-facing success text after host registration.
func FormatInstallMessage(r InstallResult, chromeExtensionID string) string {
	var b strings.Builder
	b.WriteString("Browser host installed for this user. ")
	b.WriteString("Chrome, Edge, and Firefox can connect through the ODM extension. ")
	if strings.TrimSpace(chromeExtensionID) == "" {
		b.WriteString("Add your Chrome extension ID and save again so Chrome/Edge can connect. ")
	}
	b.WriteString(fmt.Sprintf("Host binary: %s", r.HostPath))
	return strings.TrimSpace(b.String())
}

// FormatUninstallMessage is the user-facing text after host removal.
func FormatUninstallMessage(r UninstallResult) string {
	if !r.WasInstalled && len(r.RemovedPaths) == 0 {
		return "Browser integration disabled. No native messaging host was registered for this user."
	}
	return "Browser host uninstalled for this user. The extension can no longer send downloads to ODM until you enable integration again."
}

// FormatInstallError clarifies common registration failures.
func FormatInstallError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if strings.Contains(msg, "odm-nmhost not found") {
		return "Could not install the browser host: odm-nmhost was not found next to the app or on PATH. Build it first (see SETUP.md), then save again."
	}
	return "Could not install the browser host: " + msg
}
