# 🎬 C-CLI - Movie & TV Show Browser

A Go application for browsing and downloading movies and TV shows. Available as both a **terminal UI (TUI)** and a **web app**.

Inspired by [cinecli](https://github.com/eyeblech/cinecli) by [@eyeblech](https://github.com/eyeblech).

## ✨ Features

- 🔍 **Multiple search sources:**
  - **YTS** - High quality movie torrents
  - **Torrents-CSV** - General torrents (movies, TV shows, and more)
- 🎬📺 View detailed movie & TV show information (posters, ratings, cast and plot via TMDB or OMDb)
- 📺 **TV Show Support** - Automatic detection of TV series with season counts, episode runtimes, creators
- 📊 Search results sorted by popularity
- 📄 **Pagination** - Navigate through large result sets
- 🧲 Generate magnet links
- 📦 Download `.torrent` files
- ⚡ Auto-select best torrent (highest quality + healthy seeds)
- 🖥 Cross-platform (Linux, macOS, Windows, FreeBSD)

---

## 🔑 Metadata: TMDB or OMDb

Posters, ratings, cast, plot and popularity sorting come from a metadata service. Both have free keys; set **one**:

| Variable | Service | Notes |
|---|---|---|
| `TMDB_API_KEY` (or `TMDB_TOKEN`) | [TMDB](https://www.themoviedb.org) | Generous free limits. Ratings and vote counts are TMDB's |
| `OMDB_API_KEY` | [OMDb](https://www.omdbapi.com) | Free tier: 1,000 requests a day. Ratings and vote counts are IMDb's |

If both are set, TMDB is used; `METADATA_SOURCE=omdb` (or `metadata_source = "omdb"` in the TUI config) picks OMDb.
With neither, c-cli still works in basic mode (no posters for Torrents-CSV results, no popularity sorting).

**Getting a free TMDB key**

1. Create an account at [themoviedb.org/signup](https://www.themoviedb.org/signup) and confirm your email.
2. Open [Settings → API](https://www.themoviedb.org/settings/api) and request an API key: choose *Developer*, accept the
   terms, and fill in the short form (personal, non-commercial use is fine; any URL such as your GitHub profile will do).
3. Copy either the **API Key** (32 characters) or the **API Read Access Token** (the long one starting with `eyJ`).
   Either works with c-cli.

Using [watchbox](https://github.com/jgbrwn/watchbox)? Use the same Read Access Token you gave its Worker: one key for both.

**Getting a free OMDb key:** request one at [omdbapi.com/apikey.aspx](https://www.omdbapi.com/apikey.aspx) (FREE tier)
and click the activation link in the email.

<sub>This product uses the TMDB API but is not endorsed or certified by TMDB.</sub>

---

## 💻 TUI Version

Terminal-based interface built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

### Build & Run

```bash
go build -trimpath -o c-cli .

# With a free TMDB key (recommended - enables ratings, cast, plot, sorted by popularity)
TMDB_API_KEY=your_key ./c-cli

# Or with an OMDb key
OMDB_API_KEY=your_key ./c-cli

# Without either (basic mode)
./c-cli
```

### Screenshot

```
🎬 CineCLI - Movie Browser

🔍 Search for movies:

> inception
```

### Keyboard Controls

| Key | Action |
|-----|--------|
| `↑`/`↓` or `j`/`k` | Navigate lists |
| `←`/`→` or `[`/`]` | Previous/Next page (search results) |
| `Enter` | Select / Show magnet link |
| `0-9` | Select torrent by index |
| `Tab` | Switch source (search) / Switch sections |
| `Esc` | Go back |
| `a` | Auto-select best torrent |
| `m` | Show magnet link |
| `t` | Download `.torrent` file |
| `Ctrl+C` | Quit |

### Configuration

Create `~/.config/c-cli/config.toml`:

```toml
search_limit = 50
download_dir = "~/Downloads"
tmdb_api_key = "your_key_here"  # Optional (or TMDB_API_KEY env var); see "Metadata" above
# omdb_api_key = "your_key"     # Or OMDb instead (or OMDB_API_KEY env var)
# metadata_source = "omdb"      # Only if both keys are set and you want OMDb
search_source = "yts"           # "yts" or "torrents-csv"
```

With TMDB or OMDb enabled:
- Search results sorted by popularity (vote count)
- Full movie/TV show details: rating, runtime, director/creator, cast, plot
- TV shows display season count and episode runtime
- TMDB or IMDb ratings instead of YTS ratings

Search sources:
- **yts** - High quality movie torrents (default)
- **torrents-csv** - General torrents including TV shows

---

## 🌐 Web Version

Web-based interface with TMDB or OMDb metadata for rich movie and TV show details.

![Screenshot](screenshot.png)

### Build & Run

```bash
cd c-cli-web
go build -trimpath -o c-cli-web .

# With a free TMDB key (recommended - enables posters, ratings, cast, plot)
TMDB_API_KEY=your_key ./c-cli-web

# Or with an OMDb key
OMDB_API_KEY=your_key ./c-cli-web

# Without either (basic mode)
./c-cli-web
```

Then open http://localhost:8000

### Features

- 🔍 **Multiple search sources:** YTS (movies) or Torrents-CSV (all)
- 🎬📺 Movie and TV show posters in search results and details
- 📺 **TV Show Support:**
  - Automatic detection of TV series vs movies
  - Season count display
  - Episode runtime
  - Creator information (instead of director)
  - Series type badges in search results
- ⭐ Ratings, runtime, genres, director/creator, cast
- 📊 Results sorted by popularity
- 📝 Full plot descriptions
- 🧲 Magnet links with copy to clipboard
- ⬇ Download `.torrent` to server
- 💾 Download `.torrent` to your browser
- 🧲 **Torrent Cache Integration** - Fetches .torrent files from cache services for Torrents-CSV
- 🔗 Click poster to open IMDB page
- 🌙 Dark theme UI

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8000` | Server port |
| `HOST` | `127.0.0.1` | Bind address |
| `DOWNLOAD_DIR` | `$HOME` | Server download directory |
| `TMDB_API_KEY` | _(none)_ | TMDB API Key or Read Access Token ([how to get one free](#-metadata-tmdb-or-omdb)) |
| `OMDB_API_KEY` | _(none)_ | Or OMDb ([get free key](https://www.omdbapi.com/apikey.aspx)) |
| `METADATA_SOURCE` | _(auto)_ | `omdb` to prefer OMDb when both keys are set |

See [c-cli-web/README.md](./c-cli-web/README.md) for full documentation.

---

## 🛠 Tech Stack

- **Go** - Programming language
- **Bubble Tea** - TUI framework
- **Lip Gloss** - TUI styling  
- **YTS API** - Movie/torrent data
- **Torrents-CSV API** - General torrent search
- **TMDB API** or **OMDb API** - metadata (optional, both versions)

## 📄 License

Apache License 2.0 - see [LICENSE](./LICENSE) and [NOTICE](./NOTICE) for details.
