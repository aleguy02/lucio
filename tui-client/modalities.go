package main

import (
	"os"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/common-nighthawk/go-figure"
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

// styles
var (
	bannerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000ff")).
			Background(ColorSpotifyGreen).
			Padding(1, 2).
			MarginBottom(1).
			Align(lipgloss.Center)
)

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

	width  int
	height int

	banner	string
}

// TODO: refactor to use a pointer instead? look at https://github.com/bensadeh/circumflex/blob/main/view/list/list.go#L100 for reference
func NewModalities() ModalitiesModel {
	opened := true
	file, err := os.Open("ANSI_shadow.flf")
    if err != nil {
		ModalitiesLog.Println("Could not find the font file! Using default font.")
		opened = false
    }
    defer file.Close()


	fig := figure.NewFigure("LUC!O", "ogre", true)
	if opened {
		fig = figure.NewFigureWithFont("LUC!O", file, true)
	}
	
	// TODO: should this be with the rest of the colors
	accentGray := lipgloss.Color("#282828")

	return ModalitiesModel{
		Modalities: []Modality{
			{ID: "gestures", Name: "HAND GESTURES", Description: "Playback control via hand gestures", Enabled: false},
			{ID: "voice", Name: "VOICE COMMANDS", Description: "Playback control via voice commands", Enabled: false},
			{ID: "agent", Name: "AGENT", Description: "Agentic mode\n", Enabled: false},
		},
		Keys: defaultModalitiesKeyMap(),
		// TODO: why are the styles defined here? I guess it makes it so that you can have different selected styles accross pages
		SelectedStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), true).
			BorderForeground(ColorSpotifyGreen).
			Padding(1, 2).
			Width(25).
			Height(8).
			Background(accentGray),
		NormalStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), true).
			BorderForeground(ColorDarkGray).
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
			Foreground(ColorWhite),
		DescStyle: lipgloss.NewStyle().
			Foreground(ColorMidGray).
			Faint(true),
		banner:    bannerStyle.Render(fig.String()),
	}
}

func (m ModalitiesModel) Init() tea.Cmd {
	return nil
}

func (m ModalitiesModel) Update(msg tea.Msg) (ModalitiesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
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

			switch m.Modalities[m.Cursor].ID {
			case "gestures":
				return m, ToggleGesturesCmd(m.Modalities[m.Cursor].Enabled)
			}
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

	row := lipgloss.JoinHorizontal(lipgloss.Top, cards...)
	col := lipgloss.JoinVertical(lipgloss.Center, m.banner, row)
	v := tea.NewView(col)
	v.AltScreen = true
	return v
}
