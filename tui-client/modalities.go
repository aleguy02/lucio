package main

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Modality represents a feature or microservice that can be toggled.
type Modality struct {
	ID          string
	Name        string
	Description string
	Enabled     bool
}

// modalitiesKeyMap defines keybindings for the modalities component.
type modalitiesKeyMap struct {
	Left   key.Binding
	Right  key.Binding
	Toggle key.Binding
}

func defaultModalitiesKeyMap() modalitiesKeyMap {
	return modalitiesKeyMap{
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "move left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "move right"),
		),
		Toggle: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter/space", "toggle"),
		),
	}
}

type ModalitiesModel struct {
	Modalities []Modality
	Cursor     int
	Keys       modalitiesKeyMap

	SelectedStyle lipgloss.Style
	NormalStyle   lipgloss.Style
	EnabledStyle  lipgloss.Style
	DisabledStyle lipgloss.Style
	TitleStyle    lipgloss.Style
	DescStyle     lipgloss.Style
}

// TODO: refactor to use a pointer instead? look at https://github.com/bensadeh/circumflex/blob/main/view/list/list.go#L100 for reference
func NewModalities() ModalitiesModel {
	// darkGray     := lipgloss.Color("#212121")
	lightGray := lipgloss.Color("#535353")
	white := lipgloss.Color("#FFFFFF")
	accentGray := lipgloss.Color("#282828")

	return ModalitiesModel{
		Modalities: []Modality{
			{ID: "gestures", Name: "HAND GESTURES", Description: "Playback control via hand gestures", Enabled: false},
			{ID: "voice", Name: "VOICE AI", Description: "Playback control via voice commands", Enabled: false},
			{ID: "agent", Name: "AGENT", Description: "Agentic mode", Enabled: false},
		},
		Keys: defaultModalitiesKeyMap(),
		SelectedStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), true).
			BorderForeground(ColorSpotifyGreen).
			Padding(1, 2).
			Width(25).
			Height(8).
			Background(accentGray),
		NormalStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), true).
			BorderForeground(lightGray).
			Padding(1, 2).
			Width(25).
			Height(8),
		EnabledStyle: lipgloss.NewStyle().
			Foreground(ColorSpotifyGreen).
			Bold(true),
		DisabledStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4B4B")),
		TitleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(white),
		DescStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#B3B3B3")).
			Faint(true),
	}
}

func (m ModalitiesModel) Init() tea.Cmd {
	return nil
}

func (m ModalitiesModel) Update(msg tea.Msg) (ModalitiesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.Keys.Left):
			if m.Cursor > 0 {
				m.Cursor--
			}
		case key.Matches(msg, m.Keys.Right):
			if m.Cursor < len(m.Modalities)-1 {
				m.Cursor++
			}
		case key.Matches(msg, m.Keys.Toggle):
			m.Modalities[m.Cursor].Enabled = !m.Modalities[m.Cursor].Enabled
			// TODO: Optionally return a command here to notify parent models of state changes
		}
	}
	return m, nil
}

func (m ModalitiesModel) View() tea.View {
	var cards []string

	for i, mod := range m.Modalities {
		var status string
		if mod.Enabled {
			status = m.EnabledStyle.Render("● ACTIVE")
		} else {
			status = m.DisabledStyle.Render("○ INACTIVE")
		}

		cardContent := lipgloss.JoinVertical(
			lipgloss.Left,
			m.TitleStyle.Render(mod.Name),
			"",
			status,
			"",
			m.DescStyle.Render(mod.Description),
		)

		if i == m.Cursor {
			cards = append(cards, m.SelectedStyle.Render(cardContent))
		} else {
			cards = append(cards, m.NormalStyle.Render(cardContent))
		}
	}

	// Join all modality cards horizontally with gaps
	return tea.NewView(lipgloss.JoinHorizontal(lipgloss.Top, cards...))
}
