package main

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/common-nighthawk/go-figure"
)

type menuState int

const (
	terminalMode menuState = iota
	menuMode
	// TODO: vibeMode (basic skip, minimalist)
)

// menuModeKeyMap defines keybindings active while browsing the menu.
type menuModeKeyMap struct {
	Left     key.Binding
	Right    key.Binding
	Terminal key.Binding
	Help     key.Binding
}

func (k menuModeKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Left, k.Right, k.Terminal, k.Help}
}

func (k menuModeKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Left, k.Right},
		{k.Terminal, k.Help},
	}
}

// terminalModeKeyMap defines keybindings active while the command input is focused.
type terminalModeKeyMap struct {
	Submit key.Binding
	Exit   key.Binding
}

func (k terminalModeKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Submit, k.Exit}
}

func (k terminalModeKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Submit, k.Exit}}
}

// styles
var (
    bannerStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#000000ff")).
        Background(lipgloss.Color("#1DB954")).
        Padding(1, 2).
        MarginBottom(1).
        Align(lipgloss.Center)
	selectedSpinnerStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(lipgloss.Color("#1DB954"))
	itemStyle = lipgloss.NewStyle().Padding(0, 1)
)

type menu struct {
	banner	string
	spinner spinner.Model
	items     []string
	selected  int
	textInput textinput.Model
	state     menuState
	menuKeys  menuModeKeyMap
	termKeys  terminalModeKeyMap
	help      help.Model
}

func NewMenu() menu {
	ti := textinput.New()
	ti.Placeholder = "Press t for terminal mode"
	ti.SetWidth(100)
	fig := figure.NewFigure("NAME", "rectangles", true)
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = selectedSpinnerStyle

	return menu{
		banner: bannerStyle.Render(fig.String()),
		spinner: s,
		items:     []string{"H", "V", "A"},
		selected:  0,
		textInput: ti,
		state:     menuMode,
		menuKeys: menuModeKeyMap{
			Left: key.NewBinding(
				key.WithKeys("left"),
				key.WithHelp("←", "prev"),
			),
			Right: key.NewBinding(
				key.WithKeys("right"),
				key.WithHelp("→", "next"),
			),
			Terminal: key.NewBinding(
				key.WithKeys("t"),
				key.WithHelp("t", "terminal mode"),
			),
			Help: key.NewBinding(
				key.WithKeys("h"),
				key.WithHelp("h", "help"),
			),
		},
		termKeys: terminalModeKeyMap{
			Submit: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "submit"),
			),
			Exit: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "cancel"),
			),
		},
		help: help.New(),
	}
}

func (m menu) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m menu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if msg, ok := msg.(spinner.TickMsg); ok {
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	switch m.state {
	case menuMode:
		if msg, ok := msg.(tea.KeyPressMsg); ok {
			switch {
			case key.Matches(msg, m.menuKeys.Left):
				if m.selected > 0 {
					m.selected--
				}
			case key.Matches(msg, m.menuKeys.Right):
				if m.selected < len(m.items)-1 {
					m.selected++
				}
			case key.Matches(msg, m.menuKeys.Terminal):
				m.state = terminalMode
				m.textInput.Focus()
			case key.Matches(msg, m.menuKeys.Help):
				return m, SwitchViewCmd(GuideViewIdx)
			}
		}

	case terminalMode:
		if msg, ok := msg.(tea.KeyPressMsg); ok {
			switch {
			case key.Matches(msg, m.termKeys.Exit):
				m.state = menuMode
				m.textInput.Blur()
			case key.Matches(msg, m.termKeys.Submit):
				// TODO: send command somewhere, probably to spotify API wrapper
			default:
				m.textInput, cmd = m.textInput.Update(msg)
			}
		}
	}

	return m, cmd
}

func (m menu) View() tea.View {
	renderedItems := make([]string, len(m.items))
	for i, item := range m.items {
		if i == m.selected {
			renderedItems[i] = m.spinner.View()
		} else {
			renderedItems[i] = itemStyle.Render(item)
		}
	}

	horizontalList := lipgloss.JoinHorizontal(lipgloss.Top, renderedItems...)

	var helpBar string
	switch m.state {
	case menuMode:
		helpBar = m.help.View(m.menuKeys)
	case terminalMode:
		helpBar = m.help.View(m.termKeys)
	}

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left,
		m.banner,
		horizontalList,
		m.textInput.View(),
		helpBar,
	))
}
