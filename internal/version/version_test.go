package version

import "testing"

// TestStringIncludesProductAndVersion locks the CLI identity string format.
//
// If someone changes ProductName or Version, this test fails intentionally so
// packaging/docs stay in sync.
func TestStringIncludesProductAndVersion(t *testing.T) {
	got := String()
	if got != "ODM — Open Download Manager 1.0.0" {
		// %q shows the string with quotes and escape sequences (helpful for
		// invisible characters).
		t.Fatalf("String() = %q", got)
	}
}

