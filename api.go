package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const baseURL = "https://yts.bz/api/v2"

type Movie struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Year        int       `json:"year"`
	Rating      float64   `json:"rating"`
	Runtime     int       `json:"runtime"`
	Genres      []string  `json:"genres"`
	Summary     string    `json:"summary"`
	Description string    `json:"description_full"`
	IMDBCode    string    `json:"imdb_code"`
	Torrents    []Torrent `json:"torrents"`
	OMDB        *OMDBMovie
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
	Title      string `json:"Title"`
	Year       string `json:"Year"`
	Rated      string `json:"Rated"`
	Runtime    string `json:"Runtime"`
	Genre      string `json:"Genre"`
	Director   string `json:"Director"`
	Actors     string `json:"Actors"`
	Plot       string `json:"Plot"`
	IMDBRating string `json:"imdbRating"`
	IMDBVotes  string `json:"imdbVotes"`
	Response   string `json:"Response"`
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

var httpClient = &http.Client{
	Timeout: 15 * time.Second,
}

func SearchMovies(query string, limit int) ([]Movie, error) {
	params := url.Values{}
	params.Set("query_term", query)
	params.Set("limit", fmt.Sprintf("%d", limit))

	resp, err := httpClient.Get(fmt.Sprintf("%s/list_movies.json?%s", baseURL, params.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	movies := result.Data.Movies

	// Enrich with OMDB data and sort by popularity if API key is configured
	if config.OMDBAPIKey != "" && len(movies) > 0 {
		movies = enrichAndSortMovies(movies)
	}

	return movies, nil
}

func GetMovieDetails(movieID int) (*Movie, error) {
	params := url.Values{}
	params.Set("movie_id", fmt.Sprintf("%d", movieID))
	params.Set("with_images", "true")
	params.Set("with_cast", "true")

	resp, err := httpClient.Get(fmt.Sprintf("%s/movie_details.json?%s", baseURL, params.Encode()))
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
