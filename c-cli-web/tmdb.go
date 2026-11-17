package main

// TMDB metadata: an alternative to OMDb that fills the same OMDBMovie struct, so the rest of the
// program (and the web page) doesn't care which one is used. Get a free key at
// https://www.themoviedb.org/settings/api (sign up, then Settings → API). Either the "API Key" or the
// longer "API Read Access Token" works.
//
// This file is identical in c-cli and c-cli-web (they are separate Go modules).

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const tmdbBase = "https://api.themoviedb.org/3"

// tmdbKey is set at startup; empty means TMDB is off.
var tmdbKey string

var tmdbClient = &http.Client{Timeout: 15 * time.Second}

func tmdbEnabled() bool { return tmdbKey != "" }

// chooseMetadata picks the metadata service from the configured keys. source ("omdb", "tmdb" or
// empty) forces one; otherwise TMDB is used when its key is set, OMDb when only that one is.
// It returns the OMDb key to use (empty when TMDB was chosen) and the service name for logs.
func chooseMetadata(omdbKey, tmdbAPIKey, source string) (string, string) {
	source = strings.ToLower(strings.TrimSpace(source))
	tmdbKey = ""
	switch {
	case source == "omdb" && omdbKey != "":
		return omdbKey, "OMDb"
	case tmdbAPIKey != "" && source != "omdb":
		tmdbKey = strings.TrimSpace(tmdbAPIKey)
		return "", "TMDB"
	case omdbKey != "":
		return omdbKey, "OMDb"
	}
	return "", "disabled"
}

func tmdbGet(path string, q url.Values, out interface{}) error {
	if q == nil {
		q = url.Values{}
	}
	bearer := strings.HasPrefix(tmdbKey, "eyJ") // the Read Access Token is a JWT; the API Key is 32 hex chars
	if !bearer {
		q.Set("api_key", tmdbKey)
	}
	req, err := http.NewRequest("GET", tmdbBase+path+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	if bearer {
		req.Header.Set("Authorization", "Bearer "+tmdbKey)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := tmdbClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tmdb %s: HTTP %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// tmdbByIMDb looks a title up by its IMDb id (tt…).
func tmdbByIMDb(imdbID string) (*OMDBMovie, error) {
	var found struct {
		Movies   []struct{ ID int } `json:"movie_results"`
		Shows    []struct{ ID int } `json:"tv_results"`
		Episodes []struct {
			ShowID int `json:"show_id"`
		} `json:"tv_episode_results"`
	}
	if err := tmdbGet("/find/"+url.PathEscape(imdbID), url.Values{"external_source": {"imdb_id"}}, &found); err != nil {
		return nil, err
	}
	switch {
	case len(found.Movies) > 0:
		return tmdbDetails("movie", found.Movies[0].ID)
	case len(found.Shows) > 0:
		return tmdbDetails("tv", found.Shows[0].ID)
	case len(found.Episodes) > 0:
		return tmdbDetails("tv", found.Episodes[0].ShowID)
	}
	return nil, nil
}

type tmdbHit struct {
	ID           int    `json:"id"`
	MediaType    string `json:"media_type"`
	Title        string `json:"title"`
	Name         string `json:"name"`
	ReleaseDate  string `json:"release_date"`
	FirstAirDate string `json:"first_air_date"`
}

// tmdbSearch finds a title the way OMDb's ?t=&y=&type= does. mediaType is "movie", "series" or "".
func tmdbSearch(title string, year int, mediaType string) *OMDBMovie {
	var res struct {
		Results []tmdbHit `json:"results"`
	}
	q := url.Values{"query": {title}, "include_adult": {"false"}}
	path := "/search/multi"
	switch mediaType {
	case "movie":
		path = "/search/movie"
		if year > 0 {
			q.Set("primary_release_year", strconv.Itoa(year))
		}
	case "series":
		path = "/search/tv"
		if year > 0 {
			q.Set("first_air_date_year", strconv.Itoa(year))
		}
	}
	if err := tmdbGet(path, q, &res); err != nil {
		return nil
	}
	var hits []tmdbHit
	for _, h := range res.Results {
		switch {
		case mediaType == "movie":
			h.MediaType = "movie"
		case mediaType == "series":
			h.MediaType = "tv"
		case h.MediaType != "movie" && h.MediaType != "tv":
			continue // people
		}
		if year > 0 && path == "/search/multi" && !strings.HasPrefix(h.ReleaseDate+h.FirstAirDate, strconv.Itoa(year)) {
			continue
		}
		hits = append(hits, h)
	}
	if len(hits) == 0 {
		return nil
	}
	best := hits[0] // TMDB orders by relevance; an exact title match wins over it
	want := tmdbNorm(title)
	for _, h := range hits {
		if tmdbNorm(h.Title+h.Name) == want {
			best = h
			break
		}
	}
	m, err := tmdbDetails(best.MediaType, best.ID)
	if err != nil {
		return nil
	}
	return m
}

func tmdbNorm(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

type tmdbNamed struct {
	Name string `json:"name"`
	Job  string `json:"job"`
}

// tmdbDetails fetches one movie or show (kind "movie" or "tv") and maps it onto OMDb's fields.
func tmdbDetails(kind string, id int) (*OMDBMovie, error) {
	var d struct {
		Title        string      `json:"title"`
		Name         string      `json:"name"`
		ReleaseDate  string      `json:"release_date"`
		FirstAirDate string      `json:"first_air_date"`
		LastAirDate  string      `json:"last_air_date"`
		Status       string      `json:"status"`
		Runtime      int         `json:"runtime"`
		EpisodeRun   []int       `json:"episode_run_time"`
		Genres       []tmdbNamed `json:"genres"`
		Overview     string      `json:"overview"`
		PosterPath   string      `json:"poster_path"`
		VoteAverage  float64     `json:"vote_average"`
		VoteCount    int         `json:"vote_count"`
		IMDbID       string      `json:"imdb_id"`
		Seasons      int         `json:"number_of_seasons"`
		CreatedBy    []tmdbNamed `json:"created_by"`
		ExternalIDs  struct {
			IMDbID string `json:"imdb_id"`
		} `json:"external_ids"`
		Credits struct {
			Cast []tmdbNamed `json:"cast"`
			Crew []tmdbNamed `json:"crew"`
		} `json:"credits"`
		ReleaseDates struct {
			Results []struct {
				Country string `json:"iso_3166_1"`
				Dates   []struct {
					Certification string `json:"certification"`
				} `json:"release_dates"`
			} `json:"results"`
		} `json:"release_dates"`
		ContentRatings struct {
			Results []struct {
				Country string `json:"iso_3166_1"`
				Rating  string `json:"rating"`
			} `json:"results"`
		} `json:"content_ratings"`
	}
	extra := "credits,external_ids,release_dates"
	if kind == "tv" {
		extra = "credits,external_ids,content_ratings"
	}
	if err := tmdbGet(fmt.Sprintf("/%s/%d", kind, id), url.Values{"append_to_response": {extra}}, &d); err != nil {
		return nil, err
	}

	na := func(s string) string {
		if strings.TrimSpace(s) == "" {
			return "N/A"
		}
		return s
	}
	names := func(list []tmdbNamed, max int, jobs ...string) string {
		var out []string
		seen := map[string]bool{}
		for _, p := range list {
			if len(jobs) > 0 && !containsStr(jobs, p.Job) {
				continue
			}
			if p.Name == "" || seen[p.Name] {
				continue
			}
			seen[p.Name] = true
			out = append(out, p.Name)
			if len(out) == max {
				break
			}
		}
		return strings.Join(out, ", ")
	}

	m := &OMDBMovie{Response: "True", Type: "movie", Plot: na(d.Overview), IMDBRating: "N/A", IMDBVotes: "N/A"}
	var genres []string
	for _, g := range d.Genres {
		genres = append(genres, g.Name)
	}
	m.Genre = na(strings.Join(genres, ", "))
	m.Actors = na(names(d.Credits.Cast, 4))
	if d.VoteCount > 0 {
		m.IMDBRating = fmt.Sprintf("%.1f", d.VoteAverage)
		m.IMDBVotes = commas(d.VoteCount)
	}
	if d.PosterPath != "" {
		m.Poster = "https://image.tmdb.org/t/p/w342" + d.PosterPath
	} else {
		m.Poster = "N/A"
	}
	m.IMDBID = d.IMDbID
	if m.IMDBID == "" {
		m.IMDBID = d.ExternalIDs.IMDbID
	}

	if kind == "tv" {
		m.Type = "series"
		m.Title = d.Name
		m.Year = yearOf(d.FirstAirDate)
		if m.Year != "" && d.Status != "Returning Series" && d.Status != "In Production" {
			if end := yearOf(d.LastAirDate); end != "" && end != m.Year {
				m.Year += "–" + end
			}
		} else if m.Year != "" {
			m.Year += "–"
		}
		if len(d.EpisodeRun) > 0 && d.EpisodeRun[0] > 0 {
			m.Runtime = fmt.Sprintf("%d min", d.EpisodeRun[0])
		}
		if d.Seasons > 0 {
			m.TotalSeasons = strconv.Itoa(d.Seasons)
		}
		m.Writer = names(d.CreatedBy, 3)
		m.Director = "N/A"
		for _, c := range d.ContentRatings.Results {
			if c.Country == "US" && c.Rating != "" {
				m.Rated = c.Rating
			}
		}
	} else {
		m.Title = d.Title
		m.Year = yearOf(d.ReleaseDate)
		if d.Runtime > 0 {
			m.Runtime = fmt.Sprintf("%d min", d.Runtime)
		}
		m.Director = names(d.Credits.Crew, 3, "Director")
		m.Writer = names(d.Credits.Crew, 3, "Screenplay", "Writer", "Story", "Novel")
		for _, c := range d.ReleaseDates.Results {
			if c.Country != "US" {
				continue
			}
			for _, rd := range c.Dates {
				if rd.Certification != "" {
					m.Rated = rd.Certification
					break
				}
			}
		}
	}
	m.Runtime, m.Director, m.Writer, m.Rated = na(m.Runtime), na(m.Director), na(m.Writer), na(m.Rated)
	return m, nil
}

func yearOf(date string) string {
	if len(date) >= 4 {
		return date[:4]
	}
	return ""
}

func commas(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func containsStr(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
