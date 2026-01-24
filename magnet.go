package main

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
)

var trackers = []string{
	"udp://open.demonii.com:1337/announce",
	"udp://tracker.openbittorrent.com:80/announce",
	"udp://tracker.coppersurfer.tk:6969/announce",
	"udp://glotorrents.pw:6969/announce",
	"udp://tracker.opentrackr.org:1337/announce",
}

func BuildMagnet(hash, name string) string {
	var trackerParams strings.Builder
	for _, t := range trackers {
		trackerParams.WriteString("&tr=")
		trackerParams.WriteString(url.QueryEscape(t))
	}

	return fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s%s",
		hash, url.QueryEscape(name), trackerParams.String())
}

func OpenURL(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

func SelectBestTorrent(torrents []Torrent) *Torrent {
	if len(torrents) == 0 {
		return nil
	}

	qualityRank := map[string]int{
		"2160p": 3,
		"1080p": 2,
		"720p":  1,
	}

	best := &torrents[0]
	bestScore := qualityRank[best.Quality]*1000 + best.Seeds

	for i := 1; i < len(torrents); i++ {
		score := qualityRank[torrents[i].Quality]*1000 + torrents[i].Seeds
		if score > bestScore {
			best = &torrents[i]
			bestScore = score
		}
	}

	return best
}
