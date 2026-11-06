package main

import "testing"

// TestShellServiceReportsUnavailableDownloads documents Phase 0/1 desktop status:
// the Wails shell is up, but download management is not wired into the UI yet.
func TestShellServiceReportsUnavailableDownloads(t *testing.T) {
	service := NewShellService()

	if got := service.ProductName(); got != "Open Download Manager" {
		t.Fatalf("ProductName() = %q", got)
	}
	if service.DownloadsAvailable() {
		t.Fatal("DownloadsAvailable() = true, want false")
	}
	if got := service.StatusMessage(); got != "Download management is not available in this build." {
		t.Fatalf("StatusMessage() = %q", got)
	}
}
