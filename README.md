# 🎬 C-CLI - Movie Browser

A Go application for browsing and downloading movies from YTS. Available as both a **terminal UI (TUI)** and a **web app**.

This is a Go rewrite of [cinecli](https://github.com/eyeblech/cinecli), built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## ✨ Features

- 🔍 Search movies from YTS
- 🎥 View detailed movie information (enhanced with OMDB/IMDB data)
- 🧲 Generate magnet links
- 📦 Download `.torrent` files
- ⚡ Auto-select best torrent (highest quality + healthy seeds)
- 🖥 Cross-platform (Linux, macOS, Windows, FreeBSD)

---

## TUI Version

### Build & Run

```bash
go build -o c-cli .
./c-cli
```

### Navigation

| Key | Action |
|-----|--------|
| `↑`/`↓` or `j`/`k` | Navigate lists |
| `Enter` | Select/Confirm |
| `0-9` | Select torrent by index |
| `Tab` | Switch between sections |
| `Esc` | Go back |
| `a` | Auto-select best torrent |
| `m` | Show magnet link |
| `t` | Download torrent file |
| `Ctrl+C` | Quit |

### Configuration

Create `~/.config/c-cli/config.toml`:

```toml
search_limit = 20
download_dir = "~/Downloads"
```

---

## Web Version

See [`c-cli-web/`](./c-cli-web/) for a web-based interface with the same functionality.

### Quick Start

```bash
cd c-cli-web
go build -o c-cli-web .
OMDB_API_KEY=your_key ./c-cli-web
```

Then open http://localhost:8000

### Features

- Movie posters and full IMDB metadata (via OMDB API)
- Download torrents to server or to your browser
- Click poster to open IMDB page
- Responsive dark theme UI

See [c-cli-web/README.md](./c-cli-web/README.md) for full documentation.

---

## 🛠 Tech Stack

- **Go** - Programming language
- **Bubble Tea** - TUI framework
- **Lip Gloss** - TUI styling
- **YTS API** - Movie/torrent data
- **OMDB API** - IMDB metadata (optional)

## 📄 License

MIT
