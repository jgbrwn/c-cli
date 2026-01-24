# 🎬 CineCLI Web

A web interface for browsing and downloading movies from YTS.

## Features

- 🔍 Search movies from YTS
- 🎥 View movie details and available torrents
- 🧲 Generate magnet links (with copy to clipboard)
- ⬇ Download .torrent files to server

## Usage

```bash
# Build
go build -o c-cli-web .

# Run (default port 8000, downloads to current directory)
./c-cli-web

# Custom port and download directory
PORT=3000 DOWNLOAD_DIR=/path/to/downloads ./c-cli-web
```

Then open http://localhost:8000 in your browser.

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /` | Web UI |
| `GET /api/search?q=<query>` | Search movies |
| `GET /api/movie/<id>` | Get movie details |
| `GET /api/magnet?hash=<hash>&name=<name>` | Generate magnet link |
| `GET /api/download?url=<url>&title=<title>&quality=<quality>` | Download .torrent file |

## Tech Stack

- Go (stdlib only, no frameworks)
- Embedded static files
- YTS API
