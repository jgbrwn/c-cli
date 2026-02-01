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
	case viewDetails, viewTorrents:
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
	sourceLabel := "YTS (Movies)"
	if m.searchSource == SourceTorrentsCSV {
		sourceLabel = "Torrents-CSV (All)"
	}
	return fmt.Sprintf(
		"%s\n\n⏺ Source: %s\n\n> %s",
		headerStyle.Render("🔍 Search for movies or TV shows:"),
		selectedStyle.Render(sourceLabel),
		m.textInput.View(),
	)
}

func (m Model) viewLoading() string {
	return fmt.Sprintf("%s Loading...", m.spinner.View())
}

func (m Model) viewResults() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("🎬 Search Results") + "\n\n")

	if len(m.movies) > 0 && m.movies[0].Source == SourceTorrentsCSV {
		// Torrents-CSV format
		headerRow := fmt.Sprintf("  %-38s %-6s %-10s %-6s %s",
			dimStyle.Render("Title"),
			dimStyle.Render("Year"),
			dimStyle.Render("Size"),
			dimStyle.Render("Seeds"),
			dimStyle.Render("Rating"),
		)
		b.WriteString(headerRow + "\n")
		b.WriteString(dimStyle.Render(strings.Repeat("─", 75)) + "\n")

		for i, movie := range m.movies {
			title := movie.Title
			if len(title) > 36 {
				title = title[:33] + "..."
			}

			year := ""
			if movie.Year > 0 {
				year = fmt.Sprintf("%d", movie.Year)
			}

			rating := ""
			if movie.OMDB != nil && movie.OMDB.IMDBRating != "" && movie.OMDB.IMDBRating != "N/A" {
				rating = ratingStyle.Render(fmt.Sprintf("⭐ %s", movie.OMDB.IMDBRating))
			}

			seedsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
			if movie.Seeders < 10 {
				seedsStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			}

			row := fmt.Sprintf("%-38s %-6s %-10s %s %s",
				title,
				year,
				movie.Size,
				seedsStyle.Render(fmt.Sprintf("%-6d", movie.Seeders)),
				rating,
			)

			if i == m.selected {
				b.WriteString(selectedStyle.Render("▶ "+row) + "\n")
			} else {
				b.WriteString(normalStyle.Render("  "+row) + "\n")
			}
		}
	} else {
		// YTS format
		headerRow := fmt.Sprintf("  %-40s %-6s %-8s %s",
			dimStyle.Render("Title"),
			dimStyle.Render("Year"),
			dimStyle.Render("Rating"),
			dimStyle.Render("Votes"),
		)
		b.WriteString(headerRow + "\n")
		b.WriteString(dimStyle.Render(strings.Repeat("─", 70)) + "\n")

		for i, movie := range m.movies {
			title := movie.Title
			if len(title) > 38 {
				title = title[:35] + "..."
			}

			rating := ""
			if movie.OMDB != nil && movie.OMDB.IMDBRating != "" && movie.OMDB.IMDBRating != "N/A" {
				rating = ratingStyle.Render(fmt.Sprintf("⭐ %s", movie.OMDB.IMDBRating))
			} else if movie.Rating > 0 {
				rating = ratingStyle.Render(fmt.Sprintf("⭐ %.1f", movie.Rating))
			} else {
				rating = dimStyle.Render("  -  ")
			}

			votes := ""
			if movie.OMDB != nil && movie.OMDB.IMDBVotes != "" && movie.OMDB.IMDBVotes != "N/A" {
				votes = dimStyle.Render(movie.OMDB.IMDBVotes)
			}

			row := fmt.Sprintf("%-40s %-6d %-8s %s",
				title,
				movie.Year,
				rating,
				votes,
			)

			if i == m.selected {
				b.WriteString(selectedStyle.Render("▶ "+row) + "\n")
			} else {
				b.WriteString(normalStyle.Render("  "+row) + "\n")
			}
		}
	}

	return b.String()
}

func (m Model) viewMovieDetails() string {
	if m.movie == nil {
		return "No movie selected"
	}

	var b strings.Builder
	omdb := m.movie.OMDB

	// Get best available data (prefer OMDB)
	description := ""
	if omdb != nil && omdb.Plot != "" && omdb.Plot != "N/A" {
		description = omdb.Plot
	} else if m.movie.Summary != "" {
		description = m.movie.Summary
	} else if m.movie.Description != "" {
		description = m.movie.Description
	} else {
		description = "No description available."
	}
	if len(description) > 400 {
		description = description[:397] + "..."
	}

	genres := ""
	if omdb != nil && omdb.Genre != "" && omdb.Genre != "N/A" {
		genres = omdb.Genre
	} else if len(m.movie.Genres) > 0 {
		genres = strings.Join(m.movie.Genres, ", ")
	}

	rating := ""
	if omdb != nil && omdb.IMDBRating != "" && omdb.IMDBRating != "N/A" {
		rating = omdb.IMDBRating
	} else if m.movie.Rating > 0 {
		rating = fmt.Sprintf("%.1f", m.movie.Rating)
	}

	runtime := ""
	if omdb != nil && omdb.Runtime != "" && omdb.Runtime != "N/A" {
		runtime = omdb.Runtime
	} else if m.movie.Runtime > 0 {
		runtime = fmt.Sprintf("%d min", m.movie.Runtime)
	}

	// Build details content
	var details strings.Builder
	details.WriteString(lipgloss.NewStyle().Bold(true).Render(m.movie.Title))
	details.WriteString(fmt.Sprintf(" (%d)\n\n", m.movie.Year))

	if rating != "" {
		details.WriteString(fmt.Sprintf("⭐ Rating: %s", rating))
		if omdb != nil && omdb.IMDBVotes != "" && omdb.IMDBVotes != "N/A" {
			details.WriteString(fmt.Sprintf(" (%s votes)", omdb.IMDBVotes))
		}
		details.WriteString("\n")
	}
	if runtime != "" {
		details.WriteString(fmt.Sprintf("⏱ Runtime: %s\n", runtime))
	}
	if omdb != nil && omdb.Rated != "" && omdb.Rated != "N/A" {
		details.WriteString(fmt.Sprintf("🎫 Rated: %s\n", omdb.Rated))
	}
	if genres != "" {
		details.WriteString(fmt.Sprintf("🎭 Genres: %s\n", genres))
	}
	if omdb != nil && omdb.Director != "" && omdb.Director != "N/A" {
		details.WriteString(fmt.Sprintf("🎬 Director: %s\n", omdb.Director))
	}
	if omdb != nil && omdb.Actors != "" && omdb.Actors != "N/A" {
		details.WriteString(fmt.Sprintf("🎭 Cast: %s\n", omdb.Actors))
	}
	details.WriteString(fmt.Sprintf("\n%s", description))

	detailsBox := boxStyle.Render(details.String())
	b.WriteString(headerStyle.Render("🎬 Movie Details") + "\n")
	b.WriteString(detailsBox + "\n\n")

	// Torrents table
	b.WriteString(m.viewTorrentsTable())

	// Show magnet link if available
	if m.magnetLink != "" {
		b.WriteString("\n" + headerStyle.Render("🧲 Magnet Link:") + "\n")
		b.WriteString(dimStyle.Render(m.magnetLink) + "\n")
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


func (m Model) viewHelp() string {
	var help string

	switch m.state {
	case viewSearch:
		help = "enter: search • tab: switch source • ctrl+c: quit"
	case viewLoading:
		help = "loading..."
	case viewResults:
		help = "↑/↓: navigate • enter: select • esc: back"
	case viewDetails, viewTorrents:
		help = "↑/↓/0-9: select torrent • enter/m: show magnet • t: download .torrent • a: auto-best • esc: back"
	}

	return dimStyle.Render(help)
}
