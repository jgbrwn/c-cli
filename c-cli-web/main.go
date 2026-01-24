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
	"strconv"
	"strings"
	"time"
)

//go:embed static/*
var staticFiles embed.FS

const baseURL = "https://yts.bz/api/v2"

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

func handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		jsonError(w, "missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}

	params := url.Values{}
	params.Set("query_term", query)
	params.Set("limit", strconv.Itoa(limit))

	resp, err := httpClient.Get(fmt.Sprintf("%s/list_movies.json?%s", baseURL, params.Encode()))
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
	jsonResponse(w, movies)
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

	resp, err := httpClient.Get(fmt.Sprintf("%s/movie_details.json?%s", baseURL, params.Encode()))
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
