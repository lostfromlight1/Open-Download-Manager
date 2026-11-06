package browserhost

import (
	"strings"
	"testing"
)

func TestValidateAuthHeader(t *testing.T) {
	t.Parallel()
	if err := ValidateAuthHeader("cookie", ""); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAuthHeader("cookie", "session=abc"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAuthHeader("cookie", "a\r\nb"); err == nil {
		t.Fatal("expected CR/LF reject")
	}
	big := strings.Repeat("x", MaxAuthHeaderBytes+1)
	if err := ValidateAuthHeader("cookie", big); err == nil {
		t.Fatal("expected oversized reject")
	}
}

func TestParseRequestAcceptsCookieReferer(t *testing.T) {
	t.Parallel()
	req, err := ParseRequest([]byte(
		`{"v":1,"action":"enqueue","url":"https://example.com/a.bin","cookie":"a=1","referer":"https://example.com/"}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	if req.Cookie != "a=1" || req.Referer != "https://example.com/" {
		t.Fatalf("%+v", req)
	}
}
