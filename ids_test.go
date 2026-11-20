package main

import "testing"

func TestParseTitleID(t *testing.T) {
	cases := []struct {
		in   string
		want titleID
		ok   bool
	}{
		{"tt0049730", titleID{IMDb: "tt0049730"}, true},
		{" TT0049730 ", titleID{IMDb: "tt0049730"}, true},
		{"https://www.imdb.com/title/tt0049730/?ref_=nv", titleID{IMDb: "tt0049730"}, true},
		{"https://m.imdb.com/title/tt0903747/", titleID{IMDb: "tt0903747"}, true},
		{"imdb.com/de/title/tt0049730", titleID{IMDb: "tt0049730"}, true},
		{"https://www.themoviedb.org/movie/3114-the-searchers", titleID{TMDB: 3114, Kind: "movie"}, true},
		{"themoviedb.org/tv/1396", titleID{TMDB: 1396, Kind: "tv"}, true},
		{"tmdb:3114", titleID{TMDB: 3114}, true},
		{"tmdb:tv/1396", titleID{TMDB: 1396, Kind: "tv"}, true},
		{"TMDB 603", titleID{TMDB: 603}, true},
		{"3114", titleID{}, false},
		{"the searchers", titleID{}, false},
		{"tt12", titleID{}, false},
	}
	for _, c := range cases {
		got, ok := parseTitleID(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("parseTitleID(%q) = %+v, %v; want %+v, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}
