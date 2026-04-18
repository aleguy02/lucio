package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// --- styles ---

var (
	searchTitleStyle  = lipgloss.NewStyle().Bold(true).MarginBottom(1)
	searchCursorStyle = lipgloss.NewStyle().Foreground(ColorSpotifyGreen).Bold(true)
	searchItemStyle   = lipgloss.NewStyle().PaddingLeft(3)
	detailNameStyle   = lipgloss.NewStyle().Bold(true).Foreground(ColorSpotifyGreen).MarginBottom(1)
	detailLabelStyle  = lipgloss.NewStyle().Faint(true)
)

// --- InteractiveSearchResultsModel ---

type searchResultsKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Detail key.Binding
	Select key.Binding
	Back   key.Binding
}

func (k searchResultsKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Detail, k.Select, k.Back}
}

func (k searchResultsKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Up, k.Down, k.Detail, k.Select, k.Back}}
}

func defaultSearchResultsKeyMap() searchResultsKeyMap {
	return searchResultsKeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "down"),
		),
		Detail: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "details"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "play"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
	}
}

type InteractiveSearchResultsModel struct {
	items  []SpotifyItem
	cursor int
	Keys   searchResultsKeyMap
}

func NewInteractiveSearchResultsModel(items []SpotifyItem) InteractiveSearchResultsModel {
	return InteractiveSearchResultsModel{
		items: items,
		Keys:  defaultSearchResultsKeyMap(),
	}
}

// Selected returns the currently highlighted SpotifyItem.
func (m InteractiveSearchResultsModel) Selected() SpotifyItem {
	if len(m.items) == 0 {
		return SpotifyItem{}
	}
	return m.items[m.cursor]
}

// Update handles cursor movement. State transitions (ESC, TAB, ENTER) are
// managed by menu.Update so all FSM logic stays in one place.
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

// --- SpotifyItemModel ---

type spotifyItemKeyMap struct {
	Select key.Binding
	Back   key.Binding
}

func (k spotifyItemKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Select, k.Back}
}

func (k spotifyItemKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Select, k.Back}}
}

func defaultSpotifyItemKeyMap() spotifyItemKeyMap {
	return spotifyItemKeyMap{
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "play"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
	}
}

type SpotifyItemModel struct {
	item SpotifyItem
	Keys spotifyItemKeyMap
}

func NewSpotifyItemModel(item SpotifyItem) SpotifyItemModel {
	return SpotifyItemModel{item: item, Keys: defaultSpotifyItemKeyMap()}
}

func (m SpotifyItemModel) View() string {
	d := m.item.LongView
	var b strings.Builder
	b.WriteString(detailNameStyle.Render(d.Name))
	b.WriteString("\n")
	for _, meta := range d.Metadata {
		b.WriteString(detailLabelStyle.Render(meta.Label+": "))
		b.WriteString(meta.Value)
		b.WriteString("\n")
	}
	return b.String()
}
