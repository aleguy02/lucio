package ui

import (
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

var helpSectionLabels = []string{"TERMINAL COMMANDS", "HAND GESTURES", "VOICE COMMANDS", "AGENT"}

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
				key.WithKeys("esc", "backspace"),
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
			m.section = helpSection((int(m.section) + 1) % len(helpSectionLabels))
		case key.Matches(msg, m.keys.PrevSection):
			m.section = helpSection((int(m.section) + len(helpSectionLabels) - 1) % len(helpSectionLabels))
		}
	}
	return m, nil
}

var (
	guideTabActiveStyle = lipgloss.NewStyle().Foreground(ColorSpotifyGreen).Padding(0, 2)
	guideTabStyle       = lipgloss.NewStyle().Foreground(ColorDarkGray).Padding(0, 2)
	guideCmdStyle       = lipgloss.NewStyle().Foreground(ColorWhite).Width(34)
	guideDescStyle      = lipgloss.NewStyle().Foreground(ColorMidGray)
	guideFaintStyle     = lipgloss.NewStyle().Faint(true).Foreground(ColorDarkGray)
)

func (m guide) View() tea.View {
	var tabs []string
	for i, label := range helpSectionLabels {
		if helpSection(i) == m.section {
			tabs = append(tabs, guideTabActiveStyle.Render(label))
		} else {
			tabs = append(tabs, guideTabStyle.Render(label))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	content := lipgloss.JoinVertical(lipgloss.Left, tabBar, "", m.sectionContent())

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
			{"play", "Resume playback"},
			{"pause", "Pause playback"},
			{"skipf", "Skip to next track"},
			{"skipb", "Skip to previous track"},
			{"seekf <seconds>", "Seek forward N seconds"},
			{"seekb <seconds>", "Seek backward N seconds"},
			{"search artist <query>", "Search for an artist"},
			{"search album <query>", "Search for an album"},
			{"search track <query>", "Search for a track"},
			{"details", "Open details for current song"},
			{"devices list", "List available playback devices"},
			{"devices select <id>", "Transfer playback to device"},
			{"theme [default|minimalist|vibes]", "Set visual theme"},
			{"help", "Open help"},
		}
		var lines []string
		for _, r := range rows {
			lines = append(lines, guideCmdStyle.Render(r[0])+guideDescStyle.Render(r[1]))
		}
		return lipgloss.JoinVertical(lipgloss.Left, lines...)

	case helpGesture:
		rows := [][]string{
			{"✋ open palm", "Resume playback"},
			{"✊ closed fist", "Pause playback"},
			{"👍 thumb up", "Skip to next track"},
			{"👎 thumb down", "Skip to previous track"},
		}
		var lines []string
		for _, r := range rows {
			lines = append(lines, guideCmdStyle.Render(r[0])+guideDescStyle.Render(r[1]))
		}
		return lipgloss.JoinVertical(lipgloss.Left, lines...)

	case helpVoice, helpAgent:
		return guideFaintStyle.Render("Coming soon.")
	}
	return ""
}
