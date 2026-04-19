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
)

type menuState int

const (
	terminalMode menuState = iota
	menuMode
	searchResultsMode
	spotifyItemMode
	helpMode
)

type tabIndex int

const (
	tabModalities tabIndex = iota
	tabNowPlaying
)

// menuModeKeyMap defines keybindings active while browsing the main tabs.
type menuModeKeyMap struct {
	Terminal key.Binding
	TabNext  key.Binding
	TabPrev  key.Binding
}

func (k menuModeKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Terminal, k.TabNext}
}

func (k menuModeKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Terminal, k.TabNext, k.TabPrev}}
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
	selectedSpinnerStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(ColorSpotifyGreen)
	alertStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4B4B"))
)

type menu struct {
	spinner        spinner.Model
	textInput      textinput.Model
	state          menuState
	menuKeys       menuModeKeyMap
	termKeys       terminalModeKeyMap
	help           help.Model
	modalitiesList ModalitiesModel
	alert          string
	searchResults  InteractiveSearchResultsModel
	spotifyItem    SpotifyItemModel

	currentTab tabIndex
	nowPlaying nowPlaying
	guide      guide
	theme      Theme
	width      int
	height     int
}

func NewMenu() menu {
	ti := textinput.New()
	ti.Placeholder = "command..."
	ti.Prompt = ": "
	ti.SetWidth(100)
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = selectedSpinnerStyle

	return menu{
		spinner:   s,
		textInput: ti,
		state:     menuMode,
		menuKeys: menuModeKeyMap{
			Terminal: key.NewBinding(
				key.WithKeys(":"),
				key.WithHelp(":", "command"),
			),
			TabNext: key.NewBinding(
				key.WithKeys("tab"),
				key.WithHelp("tab", "next tab"),
			),
			TabPrev: key.NewBinding(
				key.WithKeys("shift+tab"),
				key.WithHelp("shift+tab", "prev tab"),
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
		currentTab:     tabModalities,
		nowPlaying:     NewNowPlaying(),
		guide:          NewGuide(),
		theme:          ThemeDefault,
	}
}

func (m menu) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m menu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Route spinner ticks to the menu's own spinner only.
	if msg, ok := msg.(spinner.TickMsg); ok {
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.modalitiesList, _ = m.modalitiesList.Update(msg)
		m.nowPlaying, _ = m.nowPlaying.Update(msg)
		newGuide, _ := m.guide.Update(msg)
		m.guide = newGuide
		return m, nil

	case backToMenuMsg:
		m.state = menuMode
		return m, nil

	case SpotifyRouteErrorMsg:
		m.alert = string(msg)
		return m, nil

	case GestureClientExitedMsg:
		m.modalitiesList.Modalities[0].Enabled = false
		return m, nil

	case SearchResultsMsg:
		m.searchResults = NewInteractiveSearchResultsModel([]SpotifyItem(msg))
		m.state = searchResultsMode
		return m, nil

	case tea.KeyPressMsg:
		// Global ':' intercept — enters terminal mode from any browsing state.
		if key.Matches(msg, m.menuKeys.Terminal) &&
			m.state != searchResultsMode &&
			m.state != spotifyItemMode &&
			m.state != helpMode {
			m.state = terminalMode
			m.alert = ""
			m.textInput.Focus()
			return m, nil
		}

		switch m.state {
		case menuMode:
			switch {
			case key.Matches(msg, m.menuKeys.TabNext):
				m.currentTab = (m.currentTab + 1) % 2
				return m, nil
			case key.Matches(msg, m.menuKeys.TabPrev):
				m.currentTab = (m.currentTab + 1) % 2
				return m, nil
			}
			// Delegate other keys to the active tab's model.
			if m.currentTab == tabModalities {
				m.modalitiesList, cmd = m.modalitiesList.Update(msg)
			}

		case searchResultsMode:
			switch {
			case key.Matches(msg, m.searchResults.Keys.Back):
				m.state = menuMode
			case key.Matches(msg, m.searchResults.Keys.Detail):
				m.spotifyItem = NewSpotifyItemModel(m.searchResults.Selected())
				m.state = spotifyItemMode
			case key.Matches(msg, m.searchResults.Keys.Select):
				item := m.searchResults.Selected()
				if item.URI != "" {
					m.state = menuMode
					return m, func() tea.Msg { return PlaybackMsg{Item: item} }
				}
			default:
				m.searchResults, cmd = m.searchResults.Update(msg)
			}

		case spotifyItemMode:
			switch {
			case key.Matches(msg, m.spotifyItem.Keys.Back):
				m.state = searchResultsMode
			case key.Matches(msg, m.spotifyItem.Keys.Select):
				if m.spotifyItem.details != nil {
					item := m.spotifyItem.details.RawItem()
					if item.URI != "" {
						m.state = menuMode
						return m, func() tea.Msg { return PlaybackMsg{Item: item} }
					}
				}
			}

		case helpMode:
			newGuide, guideCmd := m.guide.Update(msg)
			m.guide = newGuide
			return m, guideCmd

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

				switch parts[0] {
				case "h", "help":
					if arg != "" {
						m.alert = fmt.Sprintf("could not help with %q", arg)
						return m, nil
					}
					m.state = helpMode
					m.guide = NewGuide()
					return m, m.guide.Init()

				case "details":
					// TODO: wire up to currently playing song; for now opens last loaded item.
					if m.spotifyItem.details != nil {
						m.state = spotifyItemMode
					} else {
						m.alert = "no item selected"
					}
					return m, nil

				case "theme":
					if len(parts) < 2 {
						m.alert = "usage: theme default|minimalist|vibes"
						return m, nil
					}
					switch parts[1] {
					case "default":
						m.theme = ThemeDefault
					case "minimalist":
						m.theme = ThemeMinimalist
					case "vibes":
						m.theme = ThemeVibes
					default:
						m.alert = fmt.Sprintf("unknown theme %q: try default, minimalist, vibes", parts[1])
						return m, nil
					}
					TerminalLog.Printf("theme set to %q\n", parts[1])
					return m, nil
				}

				cmdStr := SpotifyCommand(strings.ToUpper(parts[0]))
				if !IsValidSpotifyCommand(cmdStr) {
					TerminalLog.Printf("unknown command: %q\n", parts[0])
					m.alert = fmt.Sprintf("unknown command: %q", parts[0])
					return m, nil
				}

				TerminalLog.Printf("command: %q arg: %q\n", cmdStr, arg)
				return m, SpotifyActionCmd(SpotifyActionMsg{Command: cmdStr, Arg: arg})

			default:
				m.textInput, cmd = m.textInput.Update(msg)
			}
		}
	}

	return m, cmd
}

func (m menu) View() tea.View {
	switch m.state {
	case helpMode:
		return m.guide.View()

	case searchResultsMode:
		helpBar := m.help.View(m.searchResults.Keys)
		v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left,
			m.searchResults.View(),
			helpBar,
		))
		v.AltScreen = true
		return v

	case spotifyItemMode:
		helpBar := m.help.View(m.spotifyItem.Keys)
		v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left,
			m.spotifyItem.View(),
			helpBar,
		))
		v.AltScreen = true
		return v
	}

	// menuMode or terminalMode: place the active tab content with room for the bottom bar.
	tabContent := m.activeTabContent()
	bottomBar := m.bottomBar()
	return m.renderLayout(tabContent, bottomBar)
}

// activeTabContent returns the raw (unplaced) content string for the current tab.
func (m menu) activeTabContent() string {
	switch m.currentTab {
	case tabNowPlaying:
		return m.nowPlaying.View().Content
	default:
		return m.modalitiesList.View().Content
	}
}

// bottomBar returns the bottom bar content: terminal input in terminalMode,
// or the menu keybind hint bar in menuMode.
func (m menu) bottomBar() string {
	switch m.state {
	case terminalMode:
		var lines []string
		lines = append(lines, m.textInput.View())
		if m.alert != "" {
			lines = append(lines, alertStyle.Render("! "+m.alert))
		}
		lines = append(lines, m.help.View(m.termKeys))
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	default:
		if m.alert != "" {
			return lipgloss.JoinVertical(lipgloss.Left,
				alertStyle.Render("! "+m.alert),
				m.help.View(m.menuKeys),
			)
		}
		return m.help.View(m.menuKeys)
	}
}

// renderLayout places tabContent in the upper portion of the screen and
// bottomBar in the remaining rows at the bottom.
func (m menu) renderLayout(tabContent, bottomBar string) tea.View {
	var composed string
	if m.width > 0 && m.height > 0 {
		barHeight := lipgloss.Height(bottomBar)
		contentHeight := max(1, m.height-barHeight)
		placed := lipgloss.Place(m.width, contentHeight, lipgloss.Center, lipgloss.Center, tabContent)
		composed = lipgloss.JoinVertical(lipgloss.Left, placed, bottomBar)
	} else {
		composed = lipgloss.JoinVertical(lipgloss.Left, tabContent, bottomBar)
	}
	v := tea.NewView(composed)
	v.AltScreen = true
	return v
}
