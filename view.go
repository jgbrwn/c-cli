package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	var b strings.Builder

	// Header
	header := titleStyle.Render("🎬 CineCLI - Movie Browser")
	b.WriteString(header + "\n\n")

	// Main content based on state
	switch m.state {
	case viewSearch:
		b.WriteString(m.viewSearch())
	case viewLoading:
		b.WriteString(m.viewLoading())
	case viewResults:
		b.WriteString(m.viewResults())
	case viewDetails, viewTorrents, viewAction:
		b.WriteString(m.viewMovieDetails())
	}

	// Error/message display
	if m.err != nil {
		b.WriteString("\n" + errorStyle.Render("❌ "+m.err.Error()))
	}
	if m.message != "" {
		b.WriteString("\n" + successStyle.Render(m.message))
	}

	// Footer with help
	b.WriteString("\n\n" + m.viewHelp())

	return b.String()
}

func (m Model) viewSearch() string {
	return fmt.Sprintf(
		"%s\n\n> %s",
		headerStyle.Render("🔍 Search for movies:"),
		m.textInput.View(),
	)
}

func (m Model) viewLoading() string {
	return fmt.Sprintf("%s Loading...", m.spinner.View())
}

func (m Model) viewResults() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("🎬 Search Results") + "\n\n")

	// Table header
	headerRow := fmt.Sprintf("  %-6s %-40s %-6s %s",
		dimStyle.Render("ID"),
		dimStyle.Render("Title"),
		dimStyle.Render("Year"),
		dimStyle.Render("Rating"),
	)
	b.WriteString(headerRow + "\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", 70)) + "\n")

	for i, movie := range m.movies {
		title := movie.Title
		if len(title) > 38 {
			title = title[:35] + "..."
		}

		rating := ratingStyle.Render(fmt.Sprintf("⭐ %.1f", movie.Rating))

		row := fmt.Sprintf("%-6d %-40s %-6d %s",
			movie.ID,
			title,
			movie.Year,
			rating,
		)

		if i == m.selected {
			b.WriteString(selectedStyle.Render("▶ "+row) + "\n")
		} else {
			b.WriteString(normalStyle.Render("  "+row) + "\n")
		}
	}

	return b.String()
}

func (m Model) viewMovieDetails() string {
	if m.movie == nil {
		return "No movie selected"
	}

	var b strings.Builder

	// Movie info panel
	description := m.movie.Summary
	if description == "" {
		description = m.movie.Description
	}
	if description == "" {
		description = "No description available."
	}
	if len(description) > 300 {
		description = description[:297] + "..."
	}

	genres := "N/A"
	if len(m.movie.Genres) > 0 {
		genres = strings.Join(m.movie.Genres, ", ")
	}

	detailsContent := fmt.Sprintf(
		"%s (%d)\n\n"+
			"⭐ Rating: %.1f\n"+
			"⏱ Runtime: %d min\n"+
			"🎭 Genres: %s\n\n"+
			"%s",
		lipgloss.NewStyle().Bold(true).Render(m.movie.Title),
		m.movie.Year,
		m.movie.Rating,
		m.movie.Runtime,
		genres,
		description,
	)

	detailsBox := boxStyle.Render(detailsContent)
	b.WriteString(headerStyle.Render("🎬 Movie Details") + "\n")
	b.WriteString(detailsBox + "\n\n")

	// Torrents table
	b.WriteString(m.viewTorrentsTable())

	// Action selection
	if m.state == viewAction {
		b.WriteString("\n" + m.viewActionSelect())
	}

	return b.String()
}

func (m Model) viewTorrentsTable() string {
	if len(m.torrents) == 0 {
		return errorStyle.Render("❌ No torrents available.")
	}

	var b strings.Builder

	tableTitle := "🧲 Available Torrents"
	if m.state == viewTorrents {
		tableTitle = selectedStyle.Render(tableTitle + " (selecting)")
	} else {
		tableTitle = headerStyle.Render(tableTitle)
	}
	b.WriteString(tableTitle + "\n\n")

	// Table header
	headerRow := fmt.Sprintf("  %-6s %-10s %-12s %-8s %s",
		dimStyle.Render("Idx"),
		dimStyle.Render("Quality"),
		dimStyle.Render("Size"),
		dimStyle.Render("Seeds"),
		dimStyle.Render("Peers"),
	)
	b.WriteString(headerRow + "\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", 55)) + "\n")

	for i, torrent := range m.torrents {
		seedsColor := "82" // green
		if torrent.Seeds < 10 {
			seedsColor = "196" // red
		} else if torrent.Seeds < 50 {
			seedsColor = "220" // yellow
		}

		row := fmt.Sprintf("%-6d %-10s %-12s %s %-8d",
			i,
			torrent.Quality,
			torrent.Size,
			lipgloss.NewStyle().Foreground(lipgloss.Color(seedsColor)).Render(fmt.Sprintf("%-8d", torrent.Seeds)),
			torrent.Peers,
		)

		if i == m.torrentIdx {
			b.WriteString(selectedStyle.Render("▶ "+row) + "\n")
		} else {
			b.WriteString(normalStyle.Render("  "+row) + "\n")
		}
	}

	return b.String()
}

func (m Model) viewActionSelect() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("🎯 Choose Action:") + "\n\n")

	actions := []string{"🧲 Open Magnet Link", "⬇  Download .torrent File"}

	for i, action := range actions {
		if i == m.actionIdx {
			b.WriteString(selectedStyle.Render("▶ "+action) + "\n")
		} else {
			b.WriteString(normalStyle.Render("  "+action) + "\n")
		}
	}

	return b.String()
}

func (m Model) viewHelp() string {
	var help string

	switch m.state {
	case viewSearch:
		help = "enter: search • ctrl+c: quit"
	case viewLoading:
		help = "loading..."
	case viewResults:
		help = "↑/↓: navigate • enter: select • esc: back"
	case viewDetails:
		help = "tab: select torrent • m: magnet • t: torrent • a: auto-select best • esc: back"
	case viewTorrents:
		help = "↑/↓: navigate • enter/tab: choose action • m: magnet • t: torrent • a: auto-select"
	case viewAction:
		help = "↑/↓: navigate • enter: confirm • tab: back to details"
	}

	return dimStyle.Render(help)
}
