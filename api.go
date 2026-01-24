package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const baseURL = "https://yts.bz/api/v2"

type Movie struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Year        int      `json:"year"`
	Rating      float64  `json:"rating"`
	Runtime     int      `json:"runtime"`
	Genres      []string `json:"genres"`
	Summary     string   `json:"summary"`
	Description string   `json:"description_full"`
	Torrents    []Torrent `json:"torrents"`
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

	return result.Data.Movies, nil
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

	return &result.Data.Movie, nil
}
