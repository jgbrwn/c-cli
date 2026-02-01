package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View states
type viewState int

const (
	viewSearch viewState = iota
	viewLoading
	viewResults
	viewDetails
	viewTorrents
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("212")).
		Bold(true)

	normalStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	dimStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")).
		Bold(true)

	boxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2)

	headerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("99")).
		Bold(true)

	ratingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("220"))
)

// Messages
type searchResultMsg struct {
	movies []Movie
	err    error
}

type movieDetailsMsg struct {
	movie *Movie
	err   error
}

type actionCompleteMsg struct {
	message string
	err     error
}

type torrentDownloadedMsg struct {
	filepath string
	err      error
}

// Model
type Model struct {
	state        viewState
	textInput    textinput.Model
	spinner      spinner.Model
	movies       []Movie
	selected     int
	movie        *Movie
	torrents     []Torrent
	torrentIdx   int
	err          error
	message      string
	magnetLink   string
	width        int
	height       int
	searchSource SearchSource
}

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Enter movie name..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 40

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Default source from config or env
	source := SourceYTS
	if config.SearchSource == "torrents-csv" {
		source = SourceTorrentsCSV
	}

	return Model{
		state:        viewSearch,
		textInput:    ti,
		spinner:      s,
		width:        80,
		height:       24,
		searchSource: source,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case searchResultMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = viewSearch
			return m, nil
		}
		if len(msg.movies) == 0 {
			m.err = fmt.Errorf("no movies found")
			m.state = viewSearch
			return m, nil
		}
		m.movies = msg.movies
		m.selected = 0
		m.state = viewResults
		m.err = nil
		return m, nil

	case movieDetailsMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = viewResults
			return m, nil
		}
		m.movie = msg.movie
		m.torrents = msg.movie.Torrents
		m.torrentIdx = 0
		m.state = viewDetails
		m.err = nil
		return m, nil

	case actionCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.message = msg.message
		}
		return m, nil

	case torrentDownloadedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.message = fmt.Sprintf("⬇ Downloaded: %s", msg.filepath)
		}
		return m, nil
	}

	// Update text input
	if m.state == viewSearch {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// In search mode, pass most keys to text input first
	if m.state == viewSearch {
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m.handleEnter()
		case "tab":
			// Toggle search source
			if m.searchSource == SourceYTS {
				m.searchSource = SourceTorrentsCSV
			} else {
				m.searchSource = SourceYTS
			}
			return m, nil
		default:
			// Pass all other keys to text input
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
	}

	// Non-search mode key handling
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		if m.state == viewResults {
			// From results, go back to search
			m.state = viewSearch
			m.movies = nil
			m.err = nil
			m.message = ""
			return m, nil
		}
		return m.goBack(), nil

	case "enter":
		return m.handleEnter()

	case "up", "k":
		return m.handleUp(), nil

	case "down", "j":
		return m.handleDown(), nil

	case "tab":
		if m.state == viewDetails {
			m.state = viewTorrents
		} else if m.state == viewTorrents {
			m.state = viewDetails
		}
		return m, nil

	case "a":
		// Auto-select best torrent
		if m.state == viewTorrents || m.state == viewDetails {
			if best := SelectBestTorrent(m.torrents); best != nil {
				for i, t := range m.torrents {
					if t.Hash == best.Hash {
						m.torrentIdx = i
						break
					}
				}
			}
		}
		return m, nil

	case "m":
		// Show magnet link
		if (m.state == viewTorrents || m.state == viewDetails) && len(m.torrents) > 0 {
			torrent := m.torrents[m.torrentIdx]
			m.magnetLink = BuildMagnet(torrent.Hash, fmt.Sprintf("%s %s", m.movie.Title, torrent.Quality))
			m.message = ""
			m.err = nil
		}
		return m, nil

	case "t":
		// Download torrent file
		if (m.state == viewTorrents || m.state == viewDetails) && len(m.torrents) > 0 {
			return m, m.downloadTorrent()
		}
		return m, nil

	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		// Number keys to select torrent directly
		if m.state == viewDetails || m.state == viewTorrents {
			idx := int(msg.String()[0] - '0')
			if idx < len(m.torrents) {
				m.torrentIdx = idx
			}
		}
		return m, nil
	}

	return m, nil
}

func (m Model) goBack() Model {
	switch m.state {
	case viewResults:
		m.state = viewSearch
		m.movies = nil
		m.err = nil
		m.message = ""
	case viewDetails, viewTorrents:
		m.state = viewResults
		m.movie = nil
		m.err = nil
		m.message = ""
		m.magnetLink = ""
	case viewSearch:
		// Already at root
	}
	return m
}

func (m Model) handleUp() Model {
	switch m.state {
	case viewResults:
		if m.selected > 0 {
			m.selected--
		}
	case viewDetails, viewTorrents:
		if m.torrentIdx > 0 {
			m.torrentIdx--
		}
	}
	return m
}

func (m Model) handleDown() Model {
	switch m.state {
	case viewResults:
		if m.selected < len(m.movies)-1 {
			m.selected++
		}
	case viewDetails, viewTorrents:
		if m.torrentIdx < len(m.torrents)-1 {
			m.torrentIdx++
		}
	}
	return m
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.state {
	case viewSearch:
		query := strings.TrimSpace(m.textInput.Value())
		if query == "" {
			return m, nil
		}
		m.state = viewLoading
		m.err = nil
		return m, tea.Batch(m.spinner.Tick, m.searchMovies(query))

	case viewResults:
		if len(m.movies) == 0 {
			return m, nil
		}
		selectedMovie := m.movies[m.selected]
		if selectedMovie.Source == SourceTorrentsCSV {
			// For torrents-csv, we already have all the info
			m.movie = &selectedMovie
			m.torrents = selectedMovie.Torrents
			m.torrentIdx = 0
			m.state = viewDetails
			return m, nil
		}
		// For YTS, fetch full details
		m.state = viewLoading
		return m, tea.Batch(m.spinner.Tick, m.fetchMovieDetails(selectedMovie.ID))

	case viewDetails, viewTorrents:
		// Show magnet link for selected torrent
		if len(m.torrents) > 0 {
			torrent := m.torrents[m.torrentIdx]
			m.magnetLink = BuildMagnet(torrent.Hash, fmt.Sprintf("%s %s", m.movie.Title, torrent.Quality))
			m.message = ""
			m.err = nil
		}
		return m, nil
	}

	return m, nil
}

func (m Model) searchMovies(query string) tea.Cmd {
	return func() tea.Msg {
		movies, err := SearchMovies(query, config.SearchLimit, m.searchSource)
		return searchResultMsg{movies: movies, err: err}
	}
}

func (m Model) fetchMovieDetails(id int) tea.Cmd {
	return func() tea.Msg {
		movie, err := GetMovieDetails(id)
		return movieDetailsMsg{movie: movie, err: err}
	}
}

func (m Model) downloadTorrent() tea.Cmd {
	return func() tea.Msg {
		if len(m.torrents) == 0 {
			return torrentDownloadedMsg{err: fmt.Errorf("no torrents available")}
		}
		torrent := m.torrents[m.torrentIdx]
		filepath, err := DownloadTorrentFile(torrent.URL, m.movie.Title, torrent.Quality)
		if err != nil {
			return torrentDownloadedMsg{err: err}
		}
		return torrentDownloadedMsg{filepath: filepath}
	}
}
