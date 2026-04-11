package main

import (
	"fmt"
	"strings"

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
	Terminal key.Binding
}

func (k menuModeKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Terminal}
}

func (k menuModeKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Terminal},
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
			Background(ColorSpotifyGreen).
			Padding(1, 2).
			MarginBottom(1).
			Align(lipgloss.Center)
	selectedSpinnerStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(ColorSpotifyGreen)
	itemStyle  = lipgloss.NewStyle().Padding(0, 1)
	alertStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4B4B"))
)

type menu struct {
	banner         string
	spinner        spinner.Model
	items          []string
	selected       int
	textInput      textinput.Model
	state          menuState
	menuKeys       menuModeKeyMap
	termKeys       terminalModeKeyMap
	help           help.Model
	modalitiesList ModalitiesModel
	alert          string
}

func NewMenu() menu {
	ti := textinput.New()
	ti.Placeholder = "command..."
	ti.Prompt = ": "
	ti.SetWidth(100)
	fig := figure.NewFigure("NAME", "rectangles", true)
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = selectedSpinnerStyle

	return menu{
		banner:    bannerStyle.Render(fig.String()),
		spinner:   s,
		items:     []string{"H", "V", "A"},
		selected:  0,
		textInput: ti,
		state:     menuMode,
		menuKeys: menuModeKeyMap{
			Terminal: key.NewBinding(
				key.WithKeys(":"),
				key.WithHelp(":", "command"),
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
		help:           help.New(),
		modalitiesList: NewModalities(),
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

	switch msg := msg.(type) {
	case SpotifyRouteErrorMsg:
		m.alert = string(msg)
		return m, nil

	case tea.KeyPressMsg:
		switch m.state {
		case menuMode:
			switch {
			case key.Matches(msg, m.menuKeys.Terminal):
				m.state = terminalMode
				m.alert = ""
				m.textInput.Focus()
			}
			m.modalitiesList, cmd = m.modalitiesList.Update(msg)

		case terminalMode:
			switch {
			case key.Matches(msg, m.termKeys.Exit):
				m.state = menuMode
				m.textInput.SetValue("")
				m.textInput.Blur()

			case key.Matches(msg, m.termKeys.Submit):
				input := strings.TrimSpace(m.textInput.Value())
				m.textInput.SetValue("")
				m.state = menuMode
				m.textInput.Blur()

				if input == "" {
					return m, nil
				}

				parts := strings.Fields(input)
				arg := ""
				if len(parts) > 1 {
					arg = strings.Join(parts[1:], " ")
				}

				if (parts[0] == "h") {
					if (arg != "") {
						m.alert = fmt.Sprintf("could not help with %q", arg)
						return m, nil
					}
					return m, SwitchViewCmd(GuideViewIdx)
				}

				cmdStr := SpotifyCommand(strings.ToUpper(parts[0]))
				
				if !IsValidSpotifyCommand(cmdStr) {
					m.alert = fmt.Sprintf("unknown command: %q", parts[0])
					return m, nil
				}

				return m, SpotifyActionCmd(SpotifyActionMsg{Command: cmdStr, Arg: arg})

			default:
				m.textInput, cmd = m.textInput.Update(msg)
			}
		}
	}

	return m, cmd
}

func (m menu) View() tea.View {
	var helpBar string
	switch m.state {
	case menuMode:
		helpBar = m.help.View(m.menuKeys)
	case terminalMode:
		helpBar = m.help.View(m.termKeys)
	}

	rows := []string{
		m.banner,
		m.modalitiesList.View().Content,
		m.textInput.View(),
	}
	if m.alert != "" {
		rows = append(rows, alertStyle.Render("! "+m.alert))
	}
	rows = append(rows, helpBar)

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, rows...))
}
