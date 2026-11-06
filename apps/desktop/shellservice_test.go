package main

import "testing"

func TestShellServiceReportsReadyWhenBound(t *testing.T) {
	svc := NewDownloadService()
	shell := NewShellService(svc)
	if shell.DownloadsAvailable() {
		t.Fatal("expected unavailable before bind")
	}
	if shell.ProductName() != "Open Download Manager" {
		t.Fatalf("name=%q", shell.ProductName())
	}

	svc.ready = true
	if !shell.DownloadsAvailable() {
		t.Fatal("expected available when service ready")
	}
	if shell.StatusMessage() != "Download manager ready" {
		t.Fatalf("status=%q", shell.StatusMessage())
	}
}
