package task

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Task is one download job tracked by the manager and persisted in SQLite.
//
// Pointer fields like *int64 are used when "unknown / unset" must differ from
// zero. Example: TotalBytes == nil means the server did not tell us the size;
// TotalBytes pointing at 0 would mean "known empty file".
type Task struct {
	ID              string
	URL             string
	DisplayName     string
	DestinationPath string
	State           State

	TotalBytes      *int64
	DownloadedBytes int64

	// Validators help decide if a partial file still matches the remote file.
	ETag         string
	LastModified string

	ErrorCode    string
	ErrorMessage string
	RetryCount   int

	// Media fields (Phase 10). Empty MediaSource means ordinary http(s) download.
	MediaSource     string // "" | "youtube"
	MediaFormatID   string
	MediaWebpageURL string
	MediaPlaylistID string

	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewID creates a 128-bit random hex id (32 characters).
//
// crypto/rand is cryptographically secure; math/rand is not appropriate for ids
// that must be hard to guess.
func NewID() string {
	var b [16]byte
	// Read fills b with random bytes. Ignoring the error is common for rand.Read
	// on modern OSes (it almost never fails); we still check for correctness.
	if _, err := rand.Read(b[:]); err != nil {
		// Extremely rare; fall back to timestamp-ish uniqueness for tests.
		return hex.EncodeToString([]byte(time.Now().Format("150405.000000000")))
	}
	return hex.EncodeToString(b[:])
}

// PartPath returns the in-progress file path for this task.
//
// Example: "video.mp4" → "video.mp4.part"
func (t Task) PartPath() string {
	return t.DestinationPath + ".part"
}
