package media

import "testing"

func TestIsYouTubeURL(t *testing.T) {
	t.Parallel()
	ok := []string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ",
		"https://music.youtube.com/watch?v=abc",
	}
	for _, u := range ok {
		if !IsYouTubeURL(u) {
			t.Fatalf("expected youtube: %s", u)
		}
	}
	if IsYouTubeURL("https://example.com/a.mp4") {
		t.Fatal("example.com must not match")
	}
}

func TestLooksLikePlaylist(t *testing.T) {
	t.Parallel()
	if !LooksLikePlaylist("https://www.youtube.com/playlist?list=PLtest") {
		t.Fatal("playlist")
	}
	if LooksLikePlaylist("https://www.youtube.com/watch?v=abc") {
		t.Fatal("watch should not be playlist")
	}
}
