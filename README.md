# 🎬 C-CLI - Movie Browser

A Go application for browsing and downloading movies from YTS. Available as both a **terminal UI (TUI)** and a **web app**.

Inspired by [cinecli](https://github.com/eyeblech/cinecli) by [@eyeblech](https://github.com/eyeblech).

## ✨ Features

- 🔍 Search movies from YTS
- 🎥 View detailed movie information
- 🧲 Generate magnet links
- 📦 Download `.torrent` files
- ⚡ Auto-select best torrent (highest quality + healthy seeds)
- 🖥 Cross-platform (Linux, macOS, Windows, FreeBSD)

---

## 💻 TUI Version

Terminal-based interface built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

### Build & Run

```bash
go build -o c-cli .

# With OMDB API key (recommended - enables ratings, cast, plot, sorted by popularity)
OMDB_API_KEY=your_key ./c-cli

# Without OMDB (basic mode)
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
| `Enter` | Select / Show magnet link |
| `0-9` | Select torrent by index |
| `Tab` | Switch sections |
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
omdb_api_key = "your_key_here"  # Optional, or use OMDB_API_KEY env var
```

With OMDB enabled:
- Search results sorted by IMDB popularity (vote count)
- Full movie details: rating, runtime, director, cast, plot
- IMDB ratings instead of YTS ratings

---

## 🌐 Web Version

Web-based interface with OMDB/IMDB integration for rich movie metadata.

![Screenshot](screenshot.png)

### Build & Run

```bash
cd c-cli-web
go build -o c-cli-web .

# With OMDB API key (recommended - enables posters, ratings, cast, plot)
OMDB_API_KEY=your_key ./c-cli-web

# Without OMDB (basic mode)
./c-cli-web
```

Then open http://localhost:8000

### Features

- 🎬 Movie posters in search results and details
- ⭐ IMDB ratings, runtime, genres, director, cast
- 📝 Full plot descriptions
- 🧲 Magnet links with copy to clipboard
- ⬇ Download `.torrent` to server
- 💾 Download `.torrent` to your browser
- 🔗 Click poster to open IMDB page
- 🌙 Dark theme UI

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8000` | Server port |
| `HOST` | `127.0.0.1` | Bind address |
| `DOWNLOAD_DIR` | `$HOME` | Server download directory |
| `OMDB_API_KEY` | _(none)_ | [Get free key](https://www.omdbapi.com/apikey.aspx) |

See [c-cli-web/README.md](./c-cli-web/README.md) for full documentation.

---

## 🛠 Tech Stack

- **Go** - Programming language
- **Bubble Tea** - TUI framework
- **Lip Gloss** - TUI styling  
- **YTS API** - Movie/torrent data
- **OMDB API** - IMDB metadata (optional, both versions)

## 📄 License

Apache License 2.0 - see [LICENSE](./LICENSE) and [NOTICE](./NOTICE) for details.
