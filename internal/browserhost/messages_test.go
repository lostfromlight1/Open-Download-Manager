package browserhost

import (
	"errors"
	"strings"
	"testing"
)

func TestFormatInstallMessageWarnsWithoutChromeID(t *testing.T) {
	t.Parallel()
	msg := FormatInstallMessage(InstallResult{HostPath: "/bin/odm-nmhost"}, "")
	for _, part := range []string{"installed", "Chrome extension ID", "/bin/odm-nmhost"} {
		if !strings.Contains(msg, part) {
			t.Fatalf("missing %q in %q", part, msg)
		}
	}
	msg2 := FormatInstallMessage(InstallResult{HostPath: "/bin/odm-nmhost"}, "abcdefghijklmnopqrstuvwxyzabcdef")
	if strings.Contains(msg2, "Add your Chrome extension ID") {
		t.Fatalf("should not warn when ID present: %q", msg2)
	}
}

func TestFormatUninstallMessage(t *testing.T) {
	t.Parallel()
	none := FormatUninstallMessage(UninstallResult{})
	if !strings.Contains(none, "disabled") || !strings.Contains(none, "No native messaging host") {
		t.Fatalf("%q", none)
	}
	done := FormatUninstallMessage(UninstallResult{WasInstalled: true, RemovedPaths: []string{"x"}})
	if !strings.Contains(done, "uninstalled") || !strings.Contains(done, "can no longer send") {
		t.Fatalf("%q", done)
	}
}

func TestFormatInstallError(t *testing.T) {
	t.Parallel()
	msg := FormatInstallError(errors.New("odm-nmhost not found next to the app or on PATH"))
	if !strings.Contains(msg, "Could not install") || !strings.Contains(msg, "Build it first") {
		t.Fatalf("%q", msg)
	}
}
