package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type nowPlaying struct {
	width    int
	height   int
	playback PlaybackState
}

func NewNowPlaying() nowPlaying {
	return nowPlaying{}
}

func (m nowPlaying) Init() tea.Cmd {
	return nil
}

func (m nowPlaying) Update(msg tea.Msg) (nowPlaying, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case SpotifyPlaybackStateMsg:
		// NowPlayingLog.Println(msg)  // this is printing actual song data
		m.playback = msg.State
	}
	return m, nil
}

func progressBar(progress, duration int, width int) string {
	if duration <= 0 {
		return strings.Repeat("─", width)
	}
	filled := int(float64(progress) / float64(duration) * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("─", filled) + "●" + strings.Repeat("─", width-filled)
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

	t := m.playback.Track
	// NowPlayingLog.Println(t)  // this is outputting 2026/04/20 01:23:47 [nowPlaying] {0   [] { []}}
	trackName := "No track playing"
	albumName := ""
	artistName := ""
	if len(t.ShortViewItems) > 0 {
		trackName = t.ShortViewItems[0]
	}
	if len(t.ShortViewItems) > 1 {
		artistName = t.ShortViewItems[1]
	}
	var duration int
	for _, meta := range t.LongView.Metadata {
		switch meta.Label {
		case "album":
			albumName = meta.Value
		case "duration":
			if _, err := fmt.Sscanf(meta.Value, "%d", &duration); err != nil {
				NowPlayingLog.Printf("failed to scan duration: %v", err)
			}
		}
	}

	playIcon := "▶  PLAYING"
	if !m.playback.IsPlaying {
		playIcon = "⏸  PAUSED"
	}

	const barWidth = 20
	bar := msToMMSS(m.playback.Progress) + " " + progressBar(m.playback.Progress, duration, barWidth) + " " + msToMMSS(duration)

	content := lipgloss.JoinVertical(lipgloss.Center,
		center(npContextStyle, playIcon),
		"",
		center(lipgloss.NewStyle(), npVisualStyle.Render("")),
		"",
		center(npSongStyle, trackName),
		center(npAlbumStyle, albumName),
		center(npArtistStyle, artistName),
		"",
		center(npProgressStyle, bar),
	)

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
