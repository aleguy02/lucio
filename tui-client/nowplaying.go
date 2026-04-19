package main

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type nowPlaying struct {
	width  int
	height int
}

func NewNowPlaying() nowPlaying {
	return nowPlaying{}
}

func (m nowPlaying) Init() tea.Cmd {
	return nil
}

func (m nowPlaying) Update(msg tea.Msg) (nowPlaying, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

var (
	npContextStyle = lipgloss.NewStyle().Faint(true).Foreground(ColorMidGray)
	npGenreStyle   = lipgloss.NewStyle().Faint(true).Foreground(ColorDarkGray)
	npVisualStyle  = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), true).
			BorderForeground(lipgloss.Color("#FF00FF")).
			Width(30).
			Height(8).
			Align(lipgloss.Center, lipgloss.Center)
	npSongStyle     = lipgloss.NewStyle().Bold(true).Foreground(ColorWhite)
	npAlbumStyle    = lipgloss.NewStyle().Foreground(ColorSpotifyGreen)
	npArtistStyle   = lipgloss.NewStyle().Faint(true).Foreground(ColorMidGray)
	npProgressStyle = lipgloss.NewStyle().Foreground(ColorDarkGray)
)

func (m nowPlaying) View() tea.View {
	const contentWidth = 34

	center := func(s lipgloss.Style, text string) string {
		return lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center).Inherit(s).Render(text)
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		center(npContextStyle, "▶  PLAYING ALBUM"),
		center(npGenreStyle, "Electronic · Ambient"),
		"",
		center(lipgloss.NewStyle(), npVisualStyle.Render("")),
		"",
		center(npSongStyle, "Midnight Drive"),
		center(npAlbumStyle, "Neon Dusk"),
		center(npArtistStyle, "The Architects"),
		"",
		center(npProgressStyle, "1:23 ──────────●──────── 4:07"),
	)

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
