package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed static/*
var staticFiles embed.FS

const (
	ytsBaseURL     = "https://yts.bz/api/v2"
	torrentsCSVURL = "https://torrents-csv.com/service/search"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}
var downloadDir string
var omdbAPIKey string

func main() {
	// Default download dir to home directory
	downloadDir, _ = os.UserHomeDir()
	if dir := os.Getenv("DOWNLOAD_DIR"); dir != "" {
		downloadDir = dir
	}

	port := "8000"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/api/search", handleSearch)
	http.HandleFunc("/api/movie/", handleMovieDetails)
	http.HandleFunc("/api/magnet", handleMagnet)
	http.HandleFunc("/api/download", handleDownloadToServer)
	http.HandleFunc("/api/download-file", handleDownloadToClient)
	http.HandleFunc("/api/save-magnet", handleSaveMagnet)

	omdbAPIKey = os.Getenv("OMDB_API_KEY")

	host := "127.0.0.1"
	if h := os.Getenv("HOST"); h != "" {
		host = h
	}

	addr := host + ":" + port
	omdbStatus := "disabled"
	if omdbAPIKey != "" {
		omdbStatus = "enabled"
	}
	log.Printf("Starting server on %s (downloads to: %s, OMDB: %s)", addr, downloadDir, omdbStatus)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, _ := staticFiles.ReadFile("static/index.html")
	w.Header().Set("Content-Type", "text/html")
	w.Write(data)
}

// API Types
type Movie struct {
	ID                int        `json:"id"`
	Title             string     `json:"title"`
	Year              int        `json:"year"`
	Rating            float64    `json:"rating"`
	Runtime           int        `json:"runtime"`
	Genres            []string   `json:"genres"`
	Summary           string     `json:"summary"`
	Description       string     `json:"description_full"`
	IMDBCode          string     `json:"imdb_code"`
	SmallCover        string     `json:"small_cover_image"`
	MediumCover       string     `json:"medium_cover_image"`
	LargeCover        string     `json:"large_cover_image"`
	Torrents          []Torrent  `json:"torrents"`
	OMDB              *OMDBMovie `json:"omdb,omitempty"`
}

type Torrent struct {
	URL     string `json:"url"`
	Hash    string `json:"hash"`
	Quality string `json:"quality"`
	Type    string `json:"type"`
	Size    string `json:"size"`
	Seeds   int    `json:"seeds"`
	Peers   int    `json:"peers"`
}

type searchResponse struct {
	Status string `json:"status"`
	Data   struct {
		Movies []Movie `json:"movies"`
	} `json:"data"`
}

type detailResponse struct {
	Status string `json:"status"`
	Data   struct {
		Movie Movie `json:"movie"`
	} `json:"data"`
}

// TorrentsCSV types
type TorrentsCSVResponse struct {
	Torrents []TorrentsCSVItem `json:"torrents"`
}

type TorrentsCSVItem struct {
	Infohash    string `json:"infohash"`
	Name        string `json:"name"`
	SizeBytes   int64  `json:"size_bytes"`
	Seeders     int    `json:"seeders"`
	Leechers    int    `json:"leechers"`
	CreatedUnix int64  `json:"created_unix"`
}

// Generic search result for unified API
type SearchResult struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Year      int        `json:"year,omitempty"`
	Source    string     `json:"source"` // "yts" or "torrents-csv"
	Infohash  string     `json:"infohash,omitempty"`
	Size      string     `json:"size,omitempty"`
	Seeders   int        `json:"seeders,omitempty"`
	Leechers  int        `json:"leechers,omitempty"`
	IMDBCode  string     `json:"imdb_code,omitempty"`
	OMDB      *OMDBMovie `json:"omdb,omitempty"`
	// YTS specific
	SmallCover  string    `json:"small_cover_image,omitempty"`
	MediumCover string    `json:"medium_cover_image,omitempty"`
	Torrents    []Torrent `json:"torrents,omitempty"`
}

// OMDB types
type OMDBMovie struct {
	Title      string `json:"Title"`
	Year       string `json:"Year"`
	Rated      string `json:"Rated"`
	Released   string `json:"Released"`
	Runtime    string `json:"Runtime"`
	Genre      string `json:"Genre"`
	Director   string `json:"Director"`
	Actors     string `json:"Actors"`
	Plot       string `json:"Plot"`
	Poster     string `json:"Poster"`
	IMDBRating string `json:"imdbRating"`
	IMDBVotes  string `json:"imdbVotes"`
	IMDBID     string `json:"imdbID"`
	Response   string `json:"Response"`
	Error      string `json:"Error"`
}

func fetchOMDBInfo(imdbID string) (*OMDBMovie, error) {
	if omdbAPIKey == "" || imdbID == "" {
		return nil, nil
	}

	url := fmt.Sprintf("http://www.omdbapi.com/?i=%s&apikey=%s", imdbID, omdbAPIKey)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var movie OMDBMovie
	if err := json.NewDecoder(resp.Body).Decode(&movie); err != nil {
		return nil, err
	}

	if movie.Response == "False" {
		return nil, nil
	}

	return &movie, nil
}

func enrichAndSortMovies(movies []Movie) []Movie {
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Fetch OMDB data concurrently
	for i := range movies {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if movies[idx].IMDBCode != "" {
				if omdb, err := fetchOMDBInfo(movies[idx].IMDBCode); err == nil && omdb != nil {
					mu.Lock()
					movies[idx].OMDB = omdb
					mu.Unlock()
				}
			}
		}(i)
	}
	wg.Wait()

	// Sort by IMDB votes (descending)
	sort.Slice(movies, func(i, j int) bool {
		votesI := parseVotes(movies[i].OMDB)
		votesJ := parseVotes(movies[j].OMDB)
		return votesI > votesJ
	})

	return movies
}

func parseVotes(omdb *OMDBMovie) int {
	if omdb == nil || omdb.IMDBVotes == "" || omdb.IMDBVotes == "N/A" {
		return 0
	}
	// Remove commas from vote count (e.g., "1,234,567" -> "1234567")
	voteStr := strings.ReplaceAll(omdb.IMDBVotes, ",", "")
	votes, _ := strconv.Atoi(voteStr)
	return votes
}

func extractYearAndIMDB(name string) (int, string) {
	year := 0
	imdbCode := ""

	// Try to extract year from patterns like (2010) or 2010
	yearRegex := regexp.MustCompile(`[\(\[]?(19\d{2}|20\d{2})[\)\]]?`)
	if match := yearRegex.FindString(name); match != "" {
		match = strings.Trim(match, "()[]")
		year, _ = strconv.Atoi(match)
	}

	// Try to extract IMDB code if present (rare in torrent names)
	imdbRegex := regexp.MustCompile(`tt\d{7,}`)
	if match := imdbRegex.FindString(name); match != "" {
		imdbCode = match
	}

	return year, imdbCode
}

func cleanTorrentName(name string) string {
	// Remove common torrent tags and clean up the name
	cleaners := []string{
		`\[.*?\]`,
		`\(.*?(rip|YIFY|MX|HDR|BluRay|WEB|HDTV|DVDRip|BRRip|x264|x265|HEVC|AAC|DTS|10bit|5\.1|7\.1).*?\)`,
		`(1080p|720p|480p|2160p|4K|HDR|HDR10|BluRay|BRRip|WEB-DL|WEBRip|HDTV|DVDRip)`,
		`(x264|x265|HEVC|H\.?264|H\.?265|AVC)`,
		`(AAC|DTS|AC3|FLAC|TrueHD|Atmos|5\.1|7\.1)`,
		`(YIFY|YTS|MX|RARBG|SPARKS|FGT|EVO|GECKOS|AMZN|NF|iNTERNAL)`,
		`\s*-\s*$`,
		`\.mkv$|\.mp4$|\.avi$`,
	}

	result := name
	for _, pattern := range cleaners {
		re := regexp.MustCompile(`(?i)` + pattern)
		result = re.ReplaceAllString(result, "")
	}

	// Replace dots and underscores with spaces
	result = strings.ReplaceAll(result, ".", " ")
	result = strings.ReplaceAll(result, "_", " ")

	// Clean up multiple spaces
	spaceRegex := regexp.MustCompile(`\s+`)
	result = spaceRegex.ReplaceAllString(result, " ")

	return strings.TrimSpace(result)
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func enrichTorrentsCSVResults(results []SearchResult) []SearchResult {
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Try to find OMDB data by searching for title+year
	for i := range results {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			r := &results[idx]

			// If we have an IMDB code, use it directly
			if r.IMDBCode != "" {
				if omdb, err := fetchOMDBInfo(r.IMDBCode); err == nil && omdb != nil {
					mu.Lock()
					r.OMDB = omdb
					mu.Unlock()
				}
				return
			}

			// Otherwise try to search by title and year
			if omdb, err := searchOMDB(r.Title, r.Year); err == nil && omdb != nil {
				mu.Lock()
				r.OMDB = omdb
				r.IMDBCode = omdb.IMDBID
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	// Sort by IMDB votes
	sort.Slice(results, func(i, j int) bool {
		return parseVotes(results[i].OMDB) > parseVotes(results[j].OMDB)
	})

	return results
}

func searchOMDB(title string, year int) (*OMDBMovie, error) {
	if omdbAPIKey == "" {
		return nil, nil
	}

	params := url.Values{}
	params.Set("t", title)
	params.Set("apikey", omdbAPIKey)
	if year > 0 {
		params.Set("y", strconv.Itoa(year))
	}

	resp, err := httpClient.Get("http://www.omdbapi.com/?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var movie OMDBMovie
	if err := json.NewDecoder(resp.Body).Decode(&movie); err != nil {
		return nil, err
	}

	if movie.Response == "False" {
		return nil, nil
	}

	return &movie, nil
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		jsonError(w, "missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	source := r.URL.Query().Get("source")
	if source == "" {
		source = "yts" // default
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	switch source {
	case "yts":
		handleYTSSearch(w, query, limit)
	case "torrents-csv", "tcsv":
		handleTorrentsCSVSearch(w, query, limit)
	default:
		jsonError(w, "invalid source, use 'yts' or 'torrents-csv'", http.StatusBadRequest)
	}
}

func handleYTSSearch(w http.ResponseWriter, query string, limit int) {
	params := url.Values{}
	params.Set("query_term", query)
	params.Set("limit", strconv.Itoa(limit))

	resp, err := httpClient.Get(fmt.Sprintf("%s/list_movies.json?%s", ytsBaseURL, params.Encode()))
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	movies := result.Data.Movies
	if movies == nil {
		movies = []Movie{}
	}

	// If OMDB is configured, fetch vote counts and sort by popularity
	if omdbAPIKey != "" && len(movies) > 0 {
		movies = enrichAndSortMovies(movies)
	}

	jsonResponse(w, movies)
}

func handleTorrentsCSVSearch(w http.ResponseWriter, query string, limit int) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("size", strconv.Itoa(limit))

	resp, err := httpClient.Get(fmt.Sprintf("%s?%s", torrentsCSVURL, params.Encode()))
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var result TorrentsCSVResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to SearchResult format
	results := make([]SearchResult, 0, len(result.Torrents))
	for _, t := range result.Torrents {
		year, imdbCode := extractYearAndIMDB(t.Name)
		results = append(results, SearchResult{
			ID:       t.Infohash,
			Title:    cleanTorrentName(t.Name),
			Year:     year,
			Source:   "torrents-csv",
			Infohash: t.Infohash,
			Size:     formatBytes(t.SizeBytes),
			Seeders:  t.Seeders,
			Leechers: t.Leechers,
			IMDBCode: imdbCode,
		})
	}

	// Enrich with OMDB if available
	if omdbAPIKey != "" && len(results) > 0 {
		results = enrichTorrentsCSVResults(results)
	}

	jsonResponse(w, results)
}

func handleMovieDetails(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/movie/")
	movieID, err := strconv.Atoi(idStr)
	if err != nil {
		jsonError(w, "invalid movie ID", http.StatusBadRequest)
		return
	}

	params := url.Values{}
	params.Set("movie_id", strconv.Itoa(movieID))
	params.Set("with_images", "true")
	params.Set("with_cast", "true")

	resp, err := httpClient.Get(fmt.Sprintf("%s/movie_details.json?%s", ytsBaseURL, params.Encode()))
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	var result detailResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	movie := result.Data.Movie

	// Fetch OMDB data if API key is configured
	if omdbAPIKey != "" && movie.IMDBCode != "" {
		if omdb, err := fetchOMDBInfo(movie.IMDBCode); err == nil && omdb != nil {
			movie.OMDB = omdb
		}
	}

	jsonResponse(w, movie)
}

var trackers = []string{
	"udp://open.demonii.com:1337/announce",
	"udp://tracker.openbittorrent.com:80/announce",
	"udp://tracker.coppersurfer.tk:6969/announce",
	"udp://glotorrents.pw:6969/announce",
	"udp://tracker.opentrackr.org:1337/announce",
}

func buildMagnet(hash, name string) string {
	var trackerParams strings.Builder
	for _, t := range trackers {
		trackerParams.WriteString("&tr=")
		trackerParams.WriteString(url.QueryEscape(t))
	}
	return fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s%s", hash, url.QueryEscape(name), trackerParams.String())
}

func handleMagnet(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Query().Get("hash")
	name := r.URL.Query().Get("name")
	if hash == "" {
		jsonError(w, "missing hash parameter", http.StatusBadRequest)
		return
	}
	magnet := buildMagnet(hash, name)
	jsonResponse(w, map[string]string{"magnet": magnet})
}

func handleDownloadToServer(w http.ResponseWriter, r *http.Request) {
	torrentURL := r.URL.Query().Get("url")
	title := r.URL.Query().Get("title")
	quality := r.URL.Query().Get("quality")

	if torrentURL == "" {
		jsonError(w, "missing url parameter", http.StatusBadRequest)
		return
	}

	resp, err := httpClient.Get(torrentURL)
	if err != nil {
		jsonError(w, fmt.Sprintf("download failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		jsonError(w, fmt.Sprintf("download failed with status: %d", resp.StatusCode), http.StatusBadGateway)
		return
	}

	safeTitle := sanitizeFilename(title)
	filename := fmt.Sprintf("%s.%s.torrent", safeTitle, quality)
	filepath := filepath.Join(downloadDir, filename)

	out, err := os.Create(filepath)
	if err != nil {
		jsonError(w, fmt.Sprintf("failed to create file: %v", err), http.StatusInternalServerError)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		jsonError(w, fmt.Sprintf("failed to write file: %v", err), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]string{"filepath": filepath, "filename": filename})
}

func handleDownloadToClient(w http.ResponseWriter, r *http.Request) {
	torrentURL := r.URL.Query().Get("url")
	title := r.URL.Query().Get("title")
	quality := r.URL.Query().Get("quality")

	if torrentURL == "" {
		http.Error(w, "missing url parameter", http.StatusBadRequest)
		return
	}

	resp, err := httpClient.Get(torrentURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("download failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("download failed with status: %d", resp.StatusCode), http.StatusBadGateway)
		return
	}

	safeTitle := sanitizeFilename(title)
	filename := fmt.Sprintf("%s.%s.torrent", safeTitle, quality)

	w.Header().Set("Content-Type", "application/x-bittorrent")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	io.Copy(w, resp.Body)
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "-", "\\", "-", ":", "-", "*", "-",
		"?", "-", "\"", "-", "<", "-", ">", "-", "|", "-",
	)
	return replacer.Replace(name)
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func handleSaveMagnet(w http.ResponseWriter, r *http.Request) {
	infohash := r.URL.Query().Get("infohash")
	title := r.URL.Query().Get("title")

	if infohash == "" {
		jsonError(w, "missing infohash", http.StatusBadRequest)
		return
	}

	if title == "" {
		title = infohash
	}

	magnet := fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", infohash, url.QueryEscape(title))
	for _, t := range trackers {
		magnet += "&tr=" + url.QueryEscape(t)
	}

	safeTitle := sanitizeFilename(title)
	filename := fmt.Sprintf("%s.magnet", safeTitle)
	filepath := filepath.Join(downloadDir, filename)

	if err := os.WriteFile(filepath, []byte(magnet), 0644); err != nil {
		jsonError(w, fmt.Sprintf("failed to save: %v", err), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]string{"filepath": filepath, "filename": filename})
}
