package engine

import "testing"

func TestFilenameFromContentDisposition(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{`attachment; filename="odm-amd64-installer.exe"`, "odm-amd64-installer.exe"},
		{`attachment; filename=odm-amd64-installer.exe`, "odm-amd64-installer.exe"},
		{`attachment; filename*=UTF-8''odm-amd64-installer.exe`, "odm-amd64-installer.exe"},
		{`attachment; filename="../../../etc/passwd"`, "passwd"},
		{`inline`, ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := FilenameFromContentDisposition(tc.in); got != tc.want {
			t.Fatalf("%q: got %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestLooksLikeOpaqueCDNName(t *testing.T) {
	t.Parallel()
	if !LooksLikeOpaqueCDNName("03e81db1-5f46-4bc6-b88c-6c7ab3514425") {
		t.Fatal("expected UUID")
	}
	if !LooksLikeOpaqueCDNName("03e81db1-5f46-4bc6-b88c-6c7ab3514425.bin") {
		t.Fatal("expected UUID with ext")
	}
	if LooksLikeOpaqueCDNName("odm-amd64-installer.exe") {
		t.Fatal("real name")
	}
}

func TestPreferReadableFilename(t *testing.T) {
	t.Parallel()
	got := PreferReadableFilename(
		"03e81db1-5f46-4bc6-b88c-6c7ab3514425",
		"odm-amd64-installer.exe",
	)
	if got != "odm-amd64-installer.exe" {
		t.Fatalf("got %q", got)
	}
}
