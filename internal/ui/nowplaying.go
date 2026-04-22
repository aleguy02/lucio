package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	sp "aleguy02/spotify-tui/internal/spotify"
)

type nowPlaying struct {
	width    int
	height   int
	playback sp.PlaybackState
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
	case sp.SpotifyPlaybackStateMsg:
		m.playback = msg.State
	}
	return m, nil
}

func progressBar(progress, duration int, width int) string {
	if duration <= 0 {
		return lipgloss.NewStyle().Foreground(ColorDarkGray).Render(strings.Repeat("█", width))
	}
	filled := int(float64(progress) / float64(duration) * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	filledPart := lipgloss.NewStyle().Foreground(ColorWhite).Render(strings.Repeat("█", filled))
	emptyPart := lipgloss.NewStyle().Foreground(ColorDarkGray).Render(strings.Repeat("█", width-filled))
	return filledPart + emptyPart
}

var (
	npBoxStyle    = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(ColorWhite).Padding(1, 2)
	npSongStyle   = lipgloss.NewStyle().Bold(true).Foreground(ColorWhite)
	npAlbumStyle  = lipgloss.NewStyle().Foreground(ColorLightYellow)
	npArtistStyle = lipgloss.NewStyle().Foreground(ColorLightGrey)
	npTimeStyle   = lipgloss.NewStyle().Foreground(ColorMidGray)
)

func (m nowPlaying) View() tea.View {
	const horizontalPadding = 2
	const borderWidth = 2
	boxWidth := m.width
	if boxWidth < 20 {
		boxWidth = 20
	}
	contentWidth := boxWidth - horizontalPadding*2 - borderWidth

	t := m.playback.Track
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

	song := npSongStyle.Render(trackName)
	album := npAlbumStyle.Render("  " + albumName)
	titleLine := lipgloss.NewStyle().Width(contentWidth).Render(
		lipgloss.JoinHorizontal(lipgloss.Top, song, album),
	)

	timeStr := msToMMSS(m.playback.Progress) + " "
	timeEnd := " " + msToMMSS(duration)
	barWidth := contentWidth - len(timeStr) - len(timeEnd)
	if barWidth < 1 {
		barWidth = 1
	}
	timeBefore := npTimeStyle.Render(timeStr)
	timeAfter := npTimeStyle.Render(timeEnd)
	bar := timeBefore + progressBar(m.playback.Progress, duration, barWidth) + timeAfter

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleLine,
		npArtistStyle.Render(artistName),
		"",
		bar,
	)

	rendered := npBoxStyle.Width(boxWidth).Render(content)

	v := tea.NewView(rendered)
	v.AltScreen = true
	return v
}
