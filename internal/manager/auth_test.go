package manager_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lostfromlight1/Open-Download-Manager/internal/manager"
	"github.com/lostfromlight1/Open-Download-Manager/internal/task"
)

func TestManagerEphemeralCookieHandoff(t *testing.T) {
	var sawCookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawCookie = r.Header.Get("Cookie")
		if sawCookie != "session=secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte("authed-body"))
	}))
	t.Cleanup(srv.Close)

	m, _ := openManager(t, manager.StaticDecisions{
		Cancel:   manager.FileKeep,
		Remove:   manager.FileKeep,
		Mismatch: manager.MismatchAbort,
	})

	dest := filepath.Join(t.TempDir(), "secret.bin")
	tk, err := m.Add(context.Background(), manager.AddOptions{
		URL:         srv.URL,
		Destination: dest,
		Cookie:      "session=secret",
		Referer:     "https://example.com/page",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = m.RunQueue(ctx, false)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got, err := m.Get(context.Background(), tk.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.State == task.StateCompleted {
			data, err := os.ReadFile(dest)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "authed-body" {
				t.Fatalf("body=%q", data)
			}
			if sawCookie != "session=secret" {
				t.Fatalf("cookie=%q", sawCookie)
			}
			return
		}
		if got.State == task.StateFailed {
			t.Fatalf("failed: %s %s", got.ErrorCode, got.ErrorMessage)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timeout waiting for cookie-auth download")
}
