package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

func DownloadTorrentFile(torrentURL, movieTitle, quality string) (string, error) {
	resp, err := http.Get(torrentURL)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Sanitize filename
	safeTitle := sanitizeFilename(movieTitle)
	filename := fmt.Sprintf("%s.%s.torrent", safeTitle, quality)
	filepath := filepath.Join(config.DownloadDir, filename)

	out, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filepath, nil
}

func sanitizeFilename(name string) string {
	// Replace characters that are problematic in filenames
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
	)
	return replacer.Replace(name)
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
