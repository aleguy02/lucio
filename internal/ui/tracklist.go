package ui

import (
	"fmt"
	"strings"

	sp "aleguy02/spotify-tui/internal/spotify"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var tracklistBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder()).
	BorderForeground(ColorSpotifyGreen).
	Padding(0, 1)

type tracklistKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Select    key.Binding
	AltSelect key.Binding
	PlayItem  key.Binding
	Back      key.Binding
}

func (k tracklistKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.AltSelect, k.PlayItem, k.Back}
}

func (k tracklistKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Up, k.Down, k.Select, k.AltSelect, k.PlayItem, k.Back}}
}

func defaultTracklistKeyMap() tracklistKeyMap {
	return tracklistKeyMap{
		Up:   key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
		Down: key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "play track"),
		),
		AltSelect: key.NewBinding(
			key.WithKeys("shift+enter"),
			key.WithHelp("shift+enter", "queue track"),
		),
		PlayItem: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "play context"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
	}
}

// TracklistModel renders a scrollable, bordered list of tracks (e.g. from an
// album or playlist) with cursor navigation for playing/queueing an
// individual track.
type TracklistModel struct {
	tracks    []sp.Track
	cursorPos int
	viewport  viewport.Model
	Keys      tracklistKeyMap
}

func NewTracklistModel(tracks []sp.Track) TracklistModel {
	vp := viewport.New()
	vp.Style = tracklistBoxStyle
	// Cursor movement and scrolling are owned entirely by this model's Update
	// (via EnsureVisible); the viewport's own up/k/down/j bindings would
	// otherwise scroll independently of the highlighted row.
	vp.KeyMap = viewport.KeyMap{}

	m := TracklistModel{tracks: tracks, viewport: vp, Keys: defaultTracklistKeyMap()}
	m.refreshContent()
	return m
}

func (m TracklistModel) Empty() bool {
	return len(m.tracks) == 0
}

func (m TracklistModel) Selected() sp.Track {
	if m.Empty() {
		return sp.Track{}
	}
	return m.tracks[m.cursorPos]
}

// SetSize sets the box's outer width and clamps its outer height to
// min(len(tracks)+2, maxHeight); the +2 covers the top/bottom border rows.
func (m *TracklistModel) SetSize(width, maxHeight int) {
	height := max(3, min(len(m.tracks)+2, maxHeight))
	m.viewport.SetWidth(width)
	m.viewport.SetHeight(height)
	m.refreshContent()
	m.viewport.EnsureVisible(m.cursorPos, 0, 0)
}

func (m TracklistModel) Update(msg tea.Msg) (TracklistModel, tea.Cmd) {
	if m.Empty() {
		return m, nil
	}
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(msg, m.Keys.Down) && m.cursorPos < len(m.tracks)-1:
			m.cursorPos++
			m.refreshContent()
			m.viewport.EnsureVisible(m.cursorPos, 0, 0)
		case key.Matches(msg, m.Keys.Up) && m.cursorPos > 0:
			m.cursorPos--
			m.refreshContent()
			m.viewport.EnsureVisible(m.cursorPos, 0, 0)
		}
	}
	return m, nil
}

func (m TracklistModel) View() string {
	if m.Empty() {
		return ""
	}
	return m.viewport.View()
}

func (m *TracklistModel) refreshContent() {
	lines := make([]string, len(m.tracks))
	for i, t := range m.tracks {
		label := fmt.Sprintf("%d\t%s • %s", i+1, t.Name, t.Artists)
		if i == m.cursorPos {
			lines[i] = searchCursorStyle.Render("> ") + label
		} else {
			lines[i] = searchItemStyle.Render(label)
		}
	}
	m.viewport.SetContent(strings.Join(lines, "\n"))
}
