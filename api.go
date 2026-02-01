package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ytsBaseURL     = "https://yts.bz/api/v2"
	torrentsCSVURL = "https://torrents-csv.com/service/search"
)

type Movie struct {
	ID          int          `json:"id"`
	Title       string       `json:"title"`
	Year        int          `json:"year"`
	Rating      float64      `json:"rating"`
	Runtime     int          `json:"runtime"`
	Genres      []string     `json:"genres"`
	Summary     string       `json:"summary"`
	Description string       `json:"description_full"`
	IMDBCode    string       `json:"imdb_code"`
	Torrents    []Torrent    `json:"torrents"`
	OMDB        *OMDBMovie
	Source      SearchSource // "yts" or "torrents-csv"
	Infohash    string       // For torrents-csv results
	Size        string       // For torrents-csv results
	Seeders     int          // For torrents-csv results
	Leechers    int          // For torrents-csv results
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

type OMDBMovie struct {
	Title        string `json:"Title"`
	Year         string `json:"Year"`
	Rated        string `json:"Rated"`
	Runtime      string `json:"Runtime"`
	Genre        string `json:"Genre"`
	Director     string `json:"Director"`
	Writer       string `json:"Writer"`
	Actors       string `json:"Actors"`
	Plot         string `json:"Plot"`
	IMDBRating   string `json:"imdbRating"`
	IMDBVotes    string `json:"imdbVotes"`
	IMDBID       string `json:"imdbID"`
	Type         string `json:"Type"`         // "movie", "series", or "episode"
	TotalSeasons string `json:"totalSeasons"` // Only for series
	Response     string `json:"Response"`
}

type searchResponse struct {
	Status string `json:"status"`
	Data   struct {
		MovieCount int     `json:"movie_count"`
		Movies     []Movie `json:"movies"`
	} `json:"data"`
}

// SearchResult contains paginated search results
type SearchResult struct {
	Movies       []Movie
	Total        int
	Page         int
	PerPage      int
	TotalPages   int
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
	Infohash  string `json:"infohash"`
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
	Seeders   int    `json:"seeders"`
	Leechers  int    `json:"leechers"`
}

// SearchSource type
type SearchSource string

const (
	SourceYTS         SearchSource = "yts"
	SourceTorrentsCSV SearchSource = "torrents-csv"
)

var httpClient = &http.Client{
	Timeout: 15 * time.Second,
}

func SearchMovies(query string, page, perPage int, source SearchSource) (SearchResult, error) {
	switch source {
	case SourceTorrentsCSV:
		return searchTorrentsCSV(query, page, perPage)
	default:
		return searchYTS(query, page, perPage)
	}
}

func searchYTS(query string, page, perPage int) (SearchResult, error) {
	params := url.Values{}
	params.Set("query_term", query)
	params.Set("limit", fmt.Sprintf("%d", perPage))
	params.Set("page", fmt.Sprintf("%d", page))

	resp, err := httpClient.Get(fmt.Sprintf("%s/list_movies.json?%s", ytsBaseURL, params.Encode()))
	if err != nil {
		return SearchResult{}, err
	}
	defer resp.Body.Close()

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return SearchResult{}, err
	}

	movies := result.Data.Movies
	for i := range movies {
		movies[i].Source = SourceYTS
	}

	// Enrich with OMDB data and sort by popularity if API key is configured
	if config.OMDBAPIKey != "" && len(movies) > 0 {
		movies = enrichAndSortMovies(movies)
	}

	total := result.Data.MovieCount
	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	return SearchResult{
		Movies:     movies,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

func searchTorrentsCSV(query string, page, perPage int) (SearchResult, error) {
	// Fetch larger batch and paginate locally (Torrents-CSV uses cursor pagination)
	fetchSize := 200
	params := url.Values{}
	params.Set("q", query)
	params.Set("size", fmt.Sprintf("%d", fetchSize))

	resp, err := httpClient.Get(fmt.Sprintf("%s?%s", torrentsCSVURL, params.Encode()))
	if err != nil {
		return SearchResult{}, err
	}
	defer resp.Body.Close()

	var result TorrentsCSVResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return SearchResult{}, err
	}

	allMovies := make([]Movie, 0, len(result.Torrents))
	for _, t := range result.Torrents {
		year := extractYear(t.Name)
		allMovies = append(allMovies, Movie{
			Title:    cleanTorrentName(t.Name),
			Year:     year,
			Source:   SourceTorrentsCSV,
			Infohash: t.Infohash,
			Size:     formatBytes(t.SizeBytes),
			Seeders:  t.Seeders,
			Leechers: t.Leechers,
			// Create a single "torrent" entry for consistency
			Torrents: []Torrent{{
				Hash:    t.Infohash,
				Quality: "Full",
				Size:    formatBytes(t.SizeBytes),
				Seeds:   t.Seeders,
				Peers:   t.Leechers,
			}},
		})
	}

	total := len(allMovies)
	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	// Paginate
	start := (page - 1) * perPage
	end := start + perPage
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	pageMovies := allMovies[start:end]

	// Enrich only current page with OMDB
	if config.OMDBAPIKey != "" && len(pageMovies) > 0 {
		pageMovies = enrichTorrentsCSVMovies(pageMovies)
	}

	return SearchResult{
		Movies:     pageMovies,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

func GetMovieDetails(movieID int) (*Movie, error) {
	params := url.Values{}
	params.Set("movie_id", fmt.Sprintf("%d", movieID))
	params.Set("with_images", "true")
	params.Set("with_cast", "true")

	resp, err := httpClient.Get(fmt.Sprintf("%s/movie_details.json?%s", ytsBaseURL, params.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result detailResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	movie := &result.Data.Movie

	// Fetch OMDB data if API key is configured
	if config.OMDBAPIKey != "" && movie.IMDBCode != "" {
		if omdb, err := fetchOMDBInfo(movie.IMDBCode); err == nil {
			movie.OMDB = omdb
		}
	}

	return movie, nil
}

func fetchOMDBInfo(imdbID string) (*OMDBMovie, error) {
	if config.OMDBAPIKey == "" || imdbID == "" {
		return nil, nil
	}

	omdbURL := fmt.Sprintf("http://www.omdbapi.com/?i=%s&apikey=%s", imdbID, config.OMDBAPIKey)
	resp, err := httpClient.Get(omdbURL)
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
		return parseVotes(movies[i].OMDB) > parseVotes(movies[j].OMDB)
	})

	return movies
}

func parseVotes(omdb *OMDBMovie) int {
	if omdb == nil || omdb.IMDBVotes == "" || omdb.IMDBVotes == "N/A" {
		return 0
	}
	voteStr := strings.ReplaceAll(omdb.IMDBVotes, ",", "")
	votes, _ := strconv.Atoi(voteStr)
	return votes
}

func extractYear(name string) int {
	// Match years like (2010) or just 2010
	for i := 0; i < len(name)-4; i++ {
		if (name[i] == '(' || name[i] == ' ' || i == 0) && 
		   (name[i:i+2] == "19" || name[i:i+2] == "20") {
			start := i
			if name[i] == '(' || name[i] == ' ' {
				start++
			}
			if start+4 <= len(name) {
				if year, err := strconv.Atoi(name[start:start+4]); err == nil && year >= 1900 && year <= 2100 {
					return year
				}
			}
		}
	}
	return 0
}

func cleanTorrentName(name string) string {
	// Remove common torrent tags
	patterns := []string{
		"1080p", "720p", "480p", "2160p", "4K",
		"BluRay", "BRRip", "WEB-DL", "WEBRip", "HDTV", "DVDRip", "BDRip",
		"x264", "x265", "HEVC", "H.264", "H.265", "H264", "H265", "AVC",
		"AAC", "DTS", "AC3", "FLAC", "TrueHD", "Atmos",
		"5.1", "7.1", "10bit",
		"YIFY", "YTS", "YTS.MX", "RARBG", "FGT", "EVO", "SPARKS",
		".mkv", ".mp4", ".avi",
	}

	result := name
	for _, p := range patterns {
		result = strings.ReplaceAll(result, p, "")
		result = strings.ReplaceAll(result, strings.ToLower(p), "")
	}

	// Replace dots, underscores with spaces
	result = strings.ReplaceAll(result, ".", " ")
	result = strings.ReplaceAll(result, "_", " ")

	// Remove brackets content like [xxx] or (xxx) at the end
	for {
		idx := strings.LastIndex(result, "[")
		if idx > 0 {
			result = result[:idx]
		} else {
			break
		}
	}

	// Clean up multiple spaces
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}

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

func enrichTorrentsCSVMovies(movies []Movie) []Movie {
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := range movies {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// For TV shows, extract the show name for better OMDB matching
			searchTitle := movies[idx].Title
			if looksLikeTVShow(movies[idx].Title) {
				searchTitle = extractShowName(movies[idx].Title)
			}
			if omdb, err := searchOMDB(searchTitle, movies[idx].Year); err == nil && omdb != nil {
				mu.Lock()
				movies[idx].OMDB = omdb
				movies[idx].IMDBCode = omdb.IMDBID
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	// Sort by IMDB votes
	sort.Slice(movies, func(i, j int) bool {
		return parseVotes(movies[i].OMDB) > parseVotes(movies[j].OMDB)
	})

	return movies
}

func searchOMDB(title string, year int) (*OMDBMovie, error) {
	if config.OMDBAPIKey == "" {
		return nil, nil
	}

	// First try without type restriction
	result := searchOMDBWithType(title, year, "")
	if result != nil {
		return result, nil
	}

	// If no result and title looks like a TV show, try searching as series
	if looksLikeTVShow(title) {
		return searchOMDBWithType(title, year, "series"), nil
	}

	return nil, nil
}

func searchOMDBWithType(title string, year int, mediaType string) *OMDBMovie {
	params := url.Values{}
	params.Set("t", title)
	params.Set("apikey", config.OMDBAPIKey)
	if year > 0 {
		params.Set("y", strconv.Itoa(year))
	}
	if mediaType != "" {
		params.Set("type", mediaType)
	}

	resp, err := httpClient.Get("http://www.omdbapi.com/?" + params.Encode())
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var movie OMDBMovie
	if err := json.NewDecoder(resp.Body).Decode(&movie); err != nil {
		return nil
	}

	if movie.Response == "False" {
		return nil
	}

	return &movie
}

// looksLikeTVShow checks if a torrent name suggests it's a TV show
func looksLikeTVShow(name string) bool {
	lower := strings.ToLower(name)
	patterns := []string{
		`s\d{1,2}e\d{1,2}`,     // S01E01
		`s\d{1,2}`,              // S01 (full season)
		`season\s*\d`,           // Season 1
		`episode\s*\d`,          // Episode 1
		`\d{1,2}x\d{1,2}`,       // 1x01
		`complete\s*series`,
		`complete\s*season`,
		`all\s*seasons`,
	}
	for _, p := range patterns {
		if matched, _ := regexp.MatchString(p, lower); matched {
			return true
		}
	}
	return false
}

// extractShowName extracts the show/movie name from a torrent title
func extractShowName(title string) string {
	patterns := []string{
		`(?i)\s*s\d{1,2}e\d{1,2}.*$`,
		`(?i)\s*s\d{1,2}\s*$`,
		`(?i)\s*season\s*\d+.*$`,
		`(?i)\s*\d{1,2}x\d{1,2}.*$`,
		`(?i)\s*complete\s*(series|season).*$`,
		`(?i)\s*all\s*seasons.*$`,
	}
	
	result := title
	for _, p := range patterns {
		re := regexp.MustCompile(p)
		result = re.ReplaceAllString(result, "")
	}
	
	return strings.TrimSpace(result)
}
