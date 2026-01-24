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
	viewAction
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

// Model
type Model struct {
	state       viewState
	textInput   textinput.Model
	spinner     spinner.Model
	movies      []Movie
	selected    int
	movie       *Movie
	torrents    []Torrent
	torrentIdx  int
	actionIdx   int
	err         error
	message     string
	width       int
	height      int
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

	return Model{
		state:     viewSearch,
		textInput: ti,
		spinner:   s,
		width:     80,
		height:    24,
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
		m.state = viewDetails
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
		default:
			// Pass all other keys to text input
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
	}

	// Non-search mode key handling
	switch msg.String() {
	case "ctrl+c", "q":
		if m.state == viewLoading {
			return m, tea.Quit
		}
		return m.goBack(), nil

	case "esc":
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
			m.state = viewAction
			m.actionIdx = 0
		} else if m.state == viewAction {
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
		// Open magnet directly
		if (m.state == viewTorrents || m.state == viewDetails || m.state == viewAction) && len(m.torrents) > 0 {
			return m, m.openMagnet()
		}
		return m, nil

	case "t":
		// Open torrent file directly
		if (m.state == viewTorrents || m.state == viewDetails || m.state == viewAction) && len(m.torrents) > 0 {
			return m, m.openTorrent()
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
	case viewDetails, viewTorrents, viewAction:
		m.state = viewResults
		m.movie = nil
		m.err = nil
		m.message = ""
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
	case viewTorrents:
		if m.torrentIdx > 0 {
			m.torrentIdx--
		}
	case viewAction:
		if m.actionIdx > 0 {
			m.actionIdx--
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
	case viewTorrents:
		if m.torrentIdx < len(m.torrents)-1 {
			m.torrentIdx++
		}
	case viewAction:
		if m.actionIdx < 1 {
			m.actionIdx++
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
		m.state = viewLoading
		return m, tea.Batch(m.spinner.Tick, m.fetchMovieDetails(m.movies[m.selected].ID))

	case viewTorrents:
		m.state = viewAction
		m.actionIdx = 0
		return m, nil

	case viewAction:
		if m.actionIdx == 0 {
			return m, m.openMagnet()
		} else {
			return m, m.openTorrent()
		}
	}

	return m, nil
}

func (m Model) searchMovies(query string) tea.Cmd {
	return func() tea.Msg {
		movies, err := SearchMovies(query, 20)
		return searchResultMsg{movies: movies, err: err}
	}
}

func (m Model) fetchMovieDetails(id int) tea.Cmd {
	return func() tea.Msg {
		movie, err := GetMovieDetails(id)
		return movieDetailsMsg{movie: movie, err: err}
	}
}

func (m Model) openMagnet() tea.Cmd {
	return func() tea.Msg {
		if len(m.torrents) == 0 {
			return actionCompleteMsg{err: fmt.Errorf("no torrents available")}
		}
		torrent := m.torrents[m.torrentIdx]
		magnet := BuildMagnet(torrent.Hash, fmt.Sprintf("%s %s", m.movie.Title, torrent.Quality))
		err := OpenURL(magnet)
		if err != nil {
			return actionCompleteMsg{err: err}
		}
		return actionCompleteMsg{message: "🧲 Magnet link opened in your torrent client!"}
	}
}

func (m Model) openTorrent() tea.Cmd {
	return func() tea.Msg {
		if len(m.torrents) == 0 {
			return actionCompleteMsg{err: fmt.Errorf("no torrents available")}
		}
		torrent := m.torrents[m.torrentIdx]
		err := OpenURL(torrent.URL)
		if err != nil {
			return actionCompleteMsg{err: err}
		}
		return actionCompleteMsg{message: "⬇ Torrent file download started in browser!"}
	}
}
