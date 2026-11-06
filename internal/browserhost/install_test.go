package browserhost

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildManifestRequiresHostPath(t *testing.T) {
	t.Parallel()
	if _, err := BuildManifest("", "abc"); err == nil {
		t.Fatal("expected error")
	}
	m, err := BuildManifest(t.TempDir()+string(os.PathSeparator)+"odm-nmhost", "abcdefghijklmnopqrstuvwxyzabcdef")
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != HostName || len(m.AllowedExtensions) != 1 || m.AllowedExtensions[0] != FirefoxExtensionID {
		t.Fatalf("%+v", m)
	}
	if len(m.AllowedOrigins) != 1 {
		t.Fatalf("origins=%v", m.AllowedOrigins)
	}
}

func TestWriteManifestFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "host.json")
	m, err := BuildManifest(filepath.Join(dir, "odm-nmhost"), "")
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteManifestFile(path, m); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) == 0 {
		t.Fatalf("read: %v", err)
	}
}
