# 🎬 C-CLI - Movie Browser TUI

A terminal user interface (TUI) for browsing and downloading movies from YTS.

This is a Go rewrite of [cinecli](https://github.com/eyeblech/cinecli), built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## ✨ Features

- 🔍 Search movies from YTS
- 🎥 View detailed movie information
- 🧲 Launch magnet links directly into your torrent client
- 📦 Download `.torrent` files
- ⚡ Auto-select best torrent (highest quality + healthy seeds)
- 🖥 Cross-platform (Linux, macOS, Windows)
- 🎨 Beautiful terminal UI

## 📦 Installation

```bash
go install github.com/yourusername/c-cli@latest
```

Or build from source:

```bash
git clone https://github.com/yourusername/c-cli.git
cd c-cli
go build -o c-cli .
```

## 🚀 Usage

Simply run:

```bash
./c-cli
```

### Navigation

| Key | Action |
|-----|--------|
| `↑`/`↓` or `j`/`k` | Navigate lists |
| `Enter` | Select/Confirm |
| `Tab` | Switch between sections |
| `Esc` or `q` | Go back |
| `a` | Auto-select best torrent |
| `m` | Open magnet link |
| `t` | Download torrent file |

### Workflow

1. **Search** - Enter a movie name
2. **Select** - Choose from search results
3. **View Details** - See movie info and available torrents
4. **Download** - Select torrent and choose magnet or .torrent file

## ⚙️ Configuration

Create a config file at `~/.config/c-cli/config.toml`:

```toml
default_action = "magnet"  # or "torrent"
search_limit = 20
```

## 🛠 Tech Stack

- **Go** - Programming language
- **Bubble Tea** - TUI framework
- **Lip Gloss** - Styling
- **YTS API** - Movie data source

## 📄 License

MIT
