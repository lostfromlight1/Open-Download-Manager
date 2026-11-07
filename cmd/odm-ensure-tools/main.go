// Command odm-ensure-tools installs/upgrades managed yt-dlp + ffmpeg under the ODM data dir.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lostfromlight1/Open-Download-Manager/internal/appdir"
	"github.com/lostfromlight1/Open-Download-Manager/internal/media"
)

func main() {
	dir, err := appdir.Resolve("")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("data_dir=", dir)
	fmt.Println("tools_dir=", media.ToolsDir(dir))
	fmt.Println("pin=", media.PinnedYtDlpVersion)
	t, err := media.EnsureTools(context.Background(), dir, nil, func(s string) {
		fmt.Println(s)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERR:", err)
		os.Exit(1)
	}
	fmt.Println("yt-dlp=", t.YtDlpPath, "ready=", t.YtDlpReady())
	fmt.Println("ffmpeg=", t.FFmpegPath, "ready=", t.FFmpegReady())
}
