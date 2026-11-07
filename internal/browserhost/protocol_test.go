package browserhost

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseRequestRejectsUnknownFieldsAndActions(t *testing.T) {
	t.Parallel()

	if _, err := ParseRequest([]byte(`{"v":1,"action":"shell","cmd":"rm"}`)); err == nil {
		t.Fatal("expected reject for unknown action/fields")
	}
	if _, err := ParseRequest([]byte(`{"v":2,"action":"ping"}`)); err == nil {
		t.Fatal("expected reject for bad version")
	}
	if _, err := ParseRequest([]byte(`{`)); err == nil {
		t.Fatal("expected reject for malformed json")
	}
	req, err := ParseRequest([]byte(`{"v":1,"action":"ping"}`))
	if err != nil || req.Action != "ping" {
		t.Fatalf("ping: %+v %v", req, err)
	}
	req, err = ParseRequest([]byte(`{"v":1,"action":"media_pick","url":"https://youtu.be/x"}`))
	if err != nil || req.Action != "media_pick" {
		t.Fatalf("media_pick: %+v %v", req, err)
	}
}

func TestParseRequestRejectsOversized(t *testing.T) {
	t.Parallel()
	big := []byte(`{"v":1,"action":"enqueue","url":"https://example.com/` + strings.Repeat("a", MaxMessageBytes) + `"}`)
	if _, err := ParseRequest(big); err == nil {
		t.Fatal("expected oversized reject")
	}
}

func TestSanitizeFilename(t *testing.T) {
	t.Parallel()
	if got := SanitizeFilename(`..\..\evil.txt`); got != "evil.txt" {
		t.Fatalf("got %q", got)
	}
	if got := SanitizeFilename("a:b*c?.bin"); got != "a_b_c_.bin" {
		t.Fatalf("got %q", got)
	}
	if FilenameFromURL("https://ex.com/path/file.zip?x=1") != "file.zip" {
		t.Fatalf("url name: %q", FilenameFromURL("https://ex.com/path/file.zip?x=1"))
	}
}

func TestNativeMessageRoundTrip(t *testing.T) {
	t.Parallel()
	payload, _ := json.Marshal(Request{V: 1, Action: "ping"})
	var buf bytes.Buffer
	if err := WriteMessage(&buf, payload); err != nil {
		t.Fatal(err)
	}
	var lenBuf [4]byte
	if _, err := buf.Read(lenBuf[:]); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(lenBuf[:]) != uint32(len(payload)) {
		t.Fatal("bad length prefix")
	}
	// rewrite for ReadMessage
	var frame bytes.Buffer
	_ = WriteMessage(&frame, payload)
	got, err := ReadMessage(&frame)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("round-trip %v %q", err, got)
	}
}
