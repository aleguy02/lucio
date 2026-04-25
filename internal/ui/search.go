package ui

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zmb "github.com/zmb3/spotify/v2"

	sp "aleguy02/spotify-tui/internal/spotify"
)

var (
	searchTitleStyle  = lipgloss.NewStyle().Bold(true).MarginBottom(1)
	searchCursorStyle = lipgloss.NewStyle().Foreground(ColorSpotifyGreen).Bold(true)
	searchItemStyle   = lipgloss.NewStyle().PaddingLeft(3)
	detailNameStyle   = lipgloss.NewStyle().Bold(true).Foreground(ColorSpotifyGreen).MarginBottom(1)
	detailLabelStyle  = lipgloss.NewStyle().Faint(true)
)

type searchResultsKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Detail    key.Binding
	Select    key.Binding
	Back      key.Binding
	AltSelect key.Binding
}

func (k searchResultsKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Detail, k.Select, k.AltSelect, k.Back}
}

func (k searchResultsKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Up, k.Down, k.Detail, k.Select, k.AltSelect, k.Back}}
}

func defaultSearchResultsKeyMap() searchResultsKeyMap {
	return searchResultsKeyMap{
		Up:   key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
		Down: key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
		Detail: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "details"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "play"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		AltSelect: key.NewBinding(
			key.WithKeys("shift+enter"),
			key.WithHelp("shift+enter", "queue"),
		),
	}
}

type InteractiveSearchResultsModel struct {
	items  []sp.SpotifyItem
	cursor int
	Keys   searchResultsKeyMap
}

func NewInteractiveSearchResultsModel(items []sp.SpotifyItem) InteractiveSearchResultsModel {
	return InteractiveSearchResultsModel{items: items, Keys: defaultSearchResultsKeyMap()}
}

func (m InteractiveSearchResultsModel) Selected() sp.SpotifyItem {
	if len(m.items) == 0 {
		return sp.SpotifyItem{}
	}
	return m.items[m.cursor]
}

// Update handles cursor movement. State transitions (ESC, TAB, ENTER) are
// managed by Menu.Update so all FSM logic stays in one place.
func (m InteractiveSearchResultsModel) Update(msg tea.Msg) (InteractiveSearchResultsModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(msg, m.Keys.Down):
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case key.Matches(msg, m.Keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		}
	}
	return m, nil
}

func (m InteractiveSearchResultsModel) View() string {
	if len(m.items) == 0 {
		return "No results."
	}

	var b strings.Builder
	b.WriteString(searchTitleStyle.Render(fmt.Sprintf("Results (%d)", len(m.items))))
	b.WriteString("\n")
	for i, item := range m.items {
		label := strings.Join(item.ShortViewItems, " · ")
		if i == m.cursor {
			b.WriteString(searchCursorStyle.Render("> "))
			b.WriteString(label)
		} else {
			b.WriteString(searchItemStyle.Render(label))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// SpotifyItemDetails is the interface all detail screen types must satisfy.
type SpotifyItemDetails interface {
	View() string
	ItemType() string
	RawItem() sp.SpotifyItem
	relatedItems() []sp.SpotifyItem // reserved for future graph navigation
	userStats() map[string]string   // reserved for future user analytics
}

type TrackDetails struct{ raw sp.SpotifyItem }

func (d TrackDetails) ItemType() string               { return "track" }
func (d TrackDetails) RawItem() sp.SpotifyItem        { return d.raw }
func (d TrackDetails) relatedItems() []sp.SpotifyItem { return nil }
func (d TrackDetails) userStats() map[string]string   { return nil }
func (d TrackDetails) View() string                   { return renderDetails("[Track Details]", d.raw.LongView) }

type AlbumDetails struct{ raw sp.SpotifyItem }

func (d AlbumDetails) ItemType() string               { return "album" }
func (d AlbumDetails) RawItem() sp.SpotifyItem        { return d.raw }
func (d AlbumDetails) relatedItems() []sp.SpotifyItem { return nil }
func (d AlbumDetails) userStats() map[string]string   { return nil }
func (d AlbumDetails) View() string                   { return renderDetails("[Album Details]", d.raw.LongView) }

type ArtistDetails struct{ raw sp.SpotifyItem }

func (d ArtistDetails) ItemType() string               { return "artist" }
func (d ArtistDetails) RawItem() sp.SpotifyItem        { return d.raw }
func (d ArtistDetails) relatedItems() []sp.SpotifyItem { return nil }
func (d ArtistDetails) userStats() map[string]string   { return nil }
func (d ArtistDetails) View() string                   { return renderDetails("[Artist Details]", d.raw.LongView) }

type PlaylistDetails struct{ raw sp.SpotifyItem }

func (d PlaylistDetails) ItemType() string               { return "playlist" }
func (d PlaylistDetails) RawItem() sp.SpotifyItem        { return d.raw }
func (d PlaylistDetails) relatedItems() []sp.SpotifyItem { return nil }
func (d PlaylistDetails) userStats() map[string]string   { return nil }
func (d PlaylistDetails) View() string                   { return renderDetails("[Playlist Details]", d.raw.LongView) }

func renderDetails(header string, d sp.Details) string {
	var b strings.Builder
	b.WriteString(detailNameStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(detailNameStyle.Render(d.Name))
	b.WriteString("\n")
	for _, meta := range d.Metadata {
		b.WriteString(detailLabelStyle.Render(meta.Label + ": "))
		if meta.Label == "duration" {
			ms, err := strconv.Atoi(meta.Value)
			if err != nil {
				b.WriteString("could not get duration")
			} else {
				b.WriteString(msToMMSS(ms))
			}
		} else {
			b.WriteString(meta.Value)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func NewSpotifyItemDetails(item sp.SpotifyItem) SpotifyItemDetails {
	switch item.Type {
	case zmb.SearchTypeTrack:
		return TrackDetails{raw: item}
	case zmb.SearchTypeAlbum:
		return AlbumDetails{raw: item}
	case zmb.SearchTypeArtist:
		return ArtistDetails{raw: item}
	case zmb.SearchTypePlaylist:
		return PlaylistDetails{raw: item}
	default:
		return TrackDetails{raw: item}
	}
}

type spotifyItemKeyMap struct {
	Select    key.Binding
	Back      key.Binding
	AltSelect key.Binding
}

func (k spotifyItemKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Select, k.AltSelect, k.Back}
}

func (k spotifyItemKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Select, k.AltSelect, k.Back}}
}

func defaultSpotifyItemKeyMap() spotifyItemKeyMap {
	return spotifyItemKeyMap{
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "play"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		AltSelect: key.NewBinding(
			key.WithKeys("shift+enter"),
			key.WithHelp("shift+enter", "queue"),
		),
	}
}

type SpotifyItemModel struct {
	details SpotifyItemDetails
	Keys    spotifyItemKeyMap
}

func NewSpotifyItemModel(item sp.SpotifyItem) SpotifyItemModel {
	return SpotifyItemModel{details: NewSpotifyItemDetails(item), Keys: defaultSpotifyItemKeyMap()}
}

func (m SpotifyItemModel) View() string {
	if m.details == nil {
		return ""
	}
	return m.details.View()
}
