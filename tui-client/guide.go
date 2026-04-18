package main

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type helpSection int

const (
	helpTerminal helpSection = iota
	helpGesture
	helpVoice
	helpAgent
)

var helpSectionLabels = []string{"Terminal Mode", "Gesture Mode", "Voice Mode", "Agent Mode"}

type guideKeyMap struct {
	NextSection key.Binding
	PrevSection key.Binding
	Back        key.Binding
}

func (k guideKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.NextSection, k.PrevSection, k.Back}
}

func (k guideKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.NextSection, k.PrevSection, k.Back}}
}

type guide struct {
	section helpSection
	keys    guideKeyMap
	width   int
	height  int
}

func NewGuide() guide {
	return guide{
		section: helpTerminal,
		keys: guideKeyMap{
			NextSection: key.NewBinding(
				key.WithKeys("tab"),
				key.WithHelp("tab", "next section"),
			),
			PrevSection: key.NewBinding(
				key.WithKeys("shift+tab"),
				key.WithHelp("shift+tab", "prev section"),
			),
			Back: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "back"),
			),
		},
	}
}

func (m guide) Init() tea.Cmd {
	return nil
}

func (m guide) Update(msg tea.Msg) (guide, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Back):
			return m, func() tea.Msg { return backToMenuMsg{} }
		case key.Matches(msg, m.keys.NextSection):
			m.section = helpSection((int(m.section) + 1) % 4)
		case key.Matches(msg, m.keys.PrevSection):
			m.section = helpSection((int(m.section) + 3) % 4)
		}
	}
	return m, nil
}

var (
	guideTabActiveStyle = lipgloss.NewStyle().
				Foreground(ColorSpotifyGreen).
				Underline(true).
				Padding(0, 2)
	guideTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#535353")).
			Padding(0, 2)
	guideDividerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#535353"))
	guideCmdStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Width(34)
	guideDescStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#B3B3B3"))
	guideFaintStyle = lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("#535353"))
)

func (m guide) View() tea.View {
	// Tab bar
	var tabs []string
	for i, label := range helpSectionLabels {
		if helpSection(i) == m.section {
			tabs = append(tabs, guideTabActiveStyle.Render(label))
		} else {
			tabs = append(tabs, guideTabStyle.Render(label))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	divider := guideDividerStyle.Render(strings.Repeat("─", 60))

	content := lipgloss.JoinVertical(lipgloss.Left,
		tabBar,
		divider,
		"",
		m.sectionContent(),
	)

	var placed string
	if m.width > 0 && m.height > 0 {
		placed = lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Center, content)
	} else {
		placed = content
	}

	v := tea.NewView(placed)
	v.AltScreen = true
	return v
}

func (m guide) sectionContent() string {
	switch m.section {
	case helpTerminal:
		rows := [][]string{
			{"PLAY", "Resume playback"},
			{"PAUSE", "Pause playback"},
			{"SKIPF", "Skip to next track"},
			{"SKIPB", "Skip to previous track"},
			{"SEEKF <seconds>", "Seek forward N seconds"},
			{"SEEKB <seconds>", "Seek backward N seconds"},
			{"SEARCH artist <query>", "Search for an artist"},
			{"SEARCH album <query>", "Search for an album"},
			{"SEARCH track <query>", "Search for a track"},
			{"details", "Open details for current song"},
			{"theme default|minimalist|vibes", "Set visual theme"},
		}
		var lines []string
		for _, r := range rows {
			lines = append(lines, guideCmdStyle.Render(r[0])+guideDescStyle.Render(r[1]))
		}
		return lipgloss.JoinVertical(lipgloss.Left, lines...)

	case helpGesture:
		rows := [][]string{
			{"Open_Palm", "PLAY"},
			{"Closed_Fist", "PAUSE"},
			{"Thumb_Up", "SKIPF"},
			{"Thumb_Down", "SKIPB"},
		}
		var lines []string
		for _, r := range rows {
			lines = append(lines, guideCmdStyle.Render(r[0])+" → "+guideDescStyle.Render(r[1]))
		}
		return lipgloss.JoinVertical(lipgloss.Left, lines...)

	case helpVoice, helpAgent:
		return guideFaintStyle.Render("Coming soon.")
	}
	return ""
}
