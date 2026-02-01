# 🎬 CineCLI Web

A web interface for browsing and downloading movies and TV shows, with IMDB metadata enrichment.

![Screenshot](screenshot.png)

## ✨ Features

- 🔍 **Multiple search sources:**
  - **YTS** - High quality movie torrents
  - **Torrents-CSV** - General torrents (movies, TV shows, and more)
- 🎬📺 View movie & TV show details with posters, ratings, cast, and plot (via OMDB/IMDB)
- 📺 **TV Show Support:**
  - Automatic detection of TV series vs movies
  - Season count and episode runtime display
  - Creator information for series
  - Type badges (📺 Series) in search results
- 📊 Search results sorted by IMDB popularity
- 📄 **Pagination** - Navigate through unlimited results with page controls
- 🧲 Generate magnet links (with copy to clipboard)
- ⬇ Download `.torrent` files to server
- 💾 Download `.torrent` files to your browser/computer
- 🎬 Click poster to open IMDB page

## 🚀 Usage

```bash
# Build
go build -o c-cli-web .

# Run with OMDB API key (recommended)
OMDB_API_KEY=your_key ./c-cli-web

# Run without OMDB (basic mode)
./c-cli-web
```

Then open http://localhost:8000

## ⚙️ Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8000` | Server port |
| `HOST` | `127.0.0.1` | Bind address (use `0.0.0.0` for all interfaces) |
| `DOWNLOAD_DIR` | `$HOME` | Directory for server-side torrent downloads |
| `OMDB_API_KEY` | _(none)_ | OMDB API key for IMDB metadata ([get one free](https://www.omdbapi.com/apikey.aspx)) |

With OMDB enabled:
- Search results sorted by IMDB popularity (vote count)
- Returns up to 50 results (default)
- Full movie details: rating, runtime, director, cast, plot
- Full TV show details: rating, seasons, episode runtime, creator, cast, plot
- Type detection (movie vs series vs episode)

### Example

```bash
PORT=3000 DOWNLOAD_DIR=/data/torrents OMDB_API_KEY=abc123 ./c-cli-web
```

## 📡 API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /` | Web UI |
| `GET /api/search?q=<query>&source=<yts\|torrents-csv>&page=<n>&per_page=<n>` | Search with pagination (default: page=1, per_page=20) |
| `GET /api/movie/<id>` | Get movie details (with OMDB data if configured) |
| `GET /api/omdb?i=<imdb_id>` or `?t=<title>&y=<year>` | Lookup OMDB data directly |
| `GET /api/magnet?hash=<hash>&name=<name>` | Generate magnet link |
| `GET /api/download?url=<url>&title=<title>&quality=<quality>` | Download .torrent to server |
| `GET /api/download-file?url=<url>&title=<title>&quality=<quality>` | Download .torrent to browser |
| `GET /api/save-magnet?infohash=<hash>&title=<title>` | Save magnet link to server (for torrents-csv) |

## 🛠 Tech Stack

- **Go** - No external web frameworks (stdlib only)
- **Embedded static files** - Single binary deployment
- **YTS API** - Movie and torrent data
- **Torrents-CSV API** - General torrent search
- **OMDB API** - IMDB metadata (optional)

## 📄 License

Apache License 2.0 - see [LICENSE](../LICENSE) and [NOTICE](../NOTICE) for details.
