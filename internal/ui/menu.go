package ui

import (
	"fmt"
	"strings"

	"aleguy02/spotify-tui/internal/gestures"
	sp "aleguy02/spotify-tui/internal/spotify"

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
	tabAgent
)

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

var (
	selectedSpinnerStyle = lipgloss.NewStyle().Padding(0, 1).Foreground(ColorSpotifyGreen)
	alertStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4B4B"))
	successAlertStyle    = lipgloss.NewStyle().Foreground(ColorSpotifyGreen)
)

// Menu is the central state machine owning all views and the terminal input.
type Menu struct {
	spinner         spinner.Model
	textInput       textinput.Model
	state           menuState
	menuKeys        menuModeKeyMap
	termKeys        terminalModeKeyMap
	help            help.Model
	modalitiesList  ModalitiesModel
	alert           string
	successAlert    string
	searchResults   InteractiveSearchResultsModel
	spotifyItem     SpotifyItemModel
	spotifyItemPrev menuState

	currentTab tabIndex
	nowPlaying nowPlaying
	guide      guide
	agentTab   agentTabModel
	width      int
	height     int
}

func NewMenu() Menu {
	ti := textinput.New()
	ti.Placeholder = "command..."
	ti.Prompt = ": "
	ti.SetWidth(100)
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = selectedSpinnerStyle

	return Menu{
		spinner:   s,
		textInput: ti,
		state:     menuMode,
		menuKeys: menuModeKeyMap{
			Terminal: key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "command")),
			TabNext:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
			TabPrev:  key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev tab")),
		},
		termKeys: terminalModeKeyMap{
			Submit: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
			Exit:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		},
		help:           help.New(),
		modalitiesList: NewModalities(),
		currentTab:     tabModalities,
		nowPlaying:     NewNowPlaying(),
		guide:          NewGuide(),
		agentTab:       newAgentTabModel(),
	}
}

// IsInTerminalMode reports whether the terminal command input is currently active.
func (m Menu) IsInTerminalMode() bool {
	return m.state == terminalMode
}

// IsOnAgentTab reports whether the agent chat tab is currently active.
// Used by the root model to suppress global key bindings (e.g. 'q') that
// conflict with chat input.
func (m Menu) IsOnAgentTab() bool {
	return m.state == menuMode && m.currentTab == tabAgent
}

func (m Menu) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.nowPlaying.Init())
}

func (m Menu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

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
		// Agent tab gets the content-area height (terminal height minus the 1-line help bar).
		agentMsg := tea.WindowSizeMsg{Width: msg.Width, Height: max(1, msg.Height-1)}
		m.agentTab, _ = m.agentTab.Update(agentMsg)
		return m, nil

	case backToMenuMsg:
		m.state = menuMode
		return m, nil

	case sp.SpotifyRouteErrorMsg:
		m.alert = string(msg)
		m.successAlert = ""
		return m, nil

	case sp.QueueSuccessMsg:
		m.successAlert = string(msg)
		m.alert = ""
		return m, nil

	// case sp.LikeSuccessMsg:
	// 	m.successAlert = string(msg)
	// 	m.alert = ""
	// 	return m, nil

	case gestures.GestureClientExitedMsg:
		m.modalitiesList.Modalities[0].Enabled = false
		return m, nil

	case sp.DevicesResultMsg:
		m.successAlert = string(msg)
		m.alert = ""
		return m, nil

	case sp.SearchResultsMsg:
		m.searchResults = NewInteractiveSearchResultsModel([]sp.SpotifyItem(msg))
		m.state = searchResultsMode
		return m, nil

	case sp.SpotifyPlaybackStateMsg:
		m.nowPlaying, _ = m.nowPlaying.Update(msg)
		return m, nil

	case AgentChunkMsg:
		m.agentTab, cmd = m.agentTab.Update(msg)
		return m, cmd

	case tea.PasteMsg:
		if m.state == terminalMode {
			m.textInput, cmd = m.textInput.Update(msg)
		} else if m.currentTab == tabAgent && m.state == menuMode {
			m.agentTab, cmd = m.agentTab.Update(msg)
		}
		return m, cmd

	case tea.KeyPressMsg:
		if key.Matches(msg, m.menuKeys.Terminal) &&
			m.state != searchResultsMode &&
			m.state != spotifyItemMode &&
			m.state != helpMode &&
			(m.currentTab != tabAgent || m.state != menuMode) {
			m.state = terminalMode
			m.alert = ""
			m.successAlert = ""
			m.textInput.Focus()
			return m, nil
		}

		switch m.state {
		case menuMode:
			switch {
			case key.Matches(msg, m.menuKeys.TabNext):
				// TODO(refactor): the number of tabs (3) is hardcoded which is bad. >:(
				m.currentTab = (m.currentTab + 1) % 3
				return m, nil
			case key.Matches(msg, m.menuKeys.TabPrev):
				m.currentTab = (m.currentTab + 2) % 3
				return m, nil
			}
			switch m.currentTab {
			case tabModalities:
				m.modalitiesList, cmd = m.modalitiesList.Update(msg)
			case tabAgent:
				m.agentTab, cmd = m.agentTab.Update(msg)
			}

		case searchResultsMode:
			switch {
			case key.Matches(msg, m.searchResults.Keys.Back):
				m.state = menuMode
				m.successAlert = ""
				m.alert = ""
			case key.Matches(msg, m.searchResults.Keys.Detail):
				m.spotifyItem = NewSpotifyItemModel(m.searchResults.Selected())
				m.spotifyItemPrev = searchResultsMode
				m.state = spotifyItemMode
			case key.Matches(msg, m.searchResults.Keys.Select):
				item := m.searchResults.Selected()
				if item.URI != "" {
					m.state = menuMode
					return m, func() tea.Msg { return sp.PlaybackMsg{Type: item.Type, ID: item.ID, URI: item.URI} }
				}
				TerminalLog.Println("Warning: selected search result does not have URI")
			case key.Matches(msg, m.searchResults.Keys.AltSelect):
				item := m.searchResults.Selected()
				// 1 is albums, 4 is playlists
				if item.Type == 1 || item.Type == 4 {
					errMsg := sp.SpotifyRouteErrorMsg("this item type does not support queueing")
					return m, func() tea.Msg { return errMsg }
				}
				if item.ID != "" {
					name := ""
					if len(item.ShortViewItems) > 0 {
						name = item.ShortViewItems[0]
					}
					return m, func() tea.Msg { return sp.QueueMsg{Id: string(item.ID), Name: name} }
				}
				TerminalLog.Println("Warning: selected search result does not have ID")
			default:
				m.searchResults, cmd = m.searchResults.Update(msg)
			}

		case spotifyItemMode:
			switch {
			case key.Matches(msg, m.spotifyItem.Keys.Back):
				m.state = m.spotifyItemPrev
			case key.Matches(msg, m.spotifyItem.Keys.Select):
				if m.spotifyItem.details != nil {
					item := m.spotifyItem.details.RawItem()
					if item.URI != "" {
						m.state = menuMode
						return m, func() tea.Msg { return sp.PlaybackMsg{Type: item.Type, ID: item.ID, URI: item.URI} }
					}
					TerminalLog.Println("Warning: selected search result does not have URI")
				}
			case key.Matches(msg, m.spotifyItem.Keys.AltSelect):
				item := m.spotifyItem.details.RawItem()
				// 1 is albums, 4 is playlists
				if item.Type == 1 || item.Type == 4 {
					errMsg := sp.SpotifyRouteErrorMsg("this item type does not support queueing")
					return m, func() tea.Msg { return errMsg }
				}
				if item.ID != "" {
					name := ""
					if len(item.ShortViewItems) > 0 {
						name = item.ShortViewItems[0]
					}
					return m, func() tea.Msg { return sp.QueueMsg{Id: string(item.ID), Name: name} }
				}
				TerminalLog.Println("Warning: selected search result does not have ID")
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
					track := m.nowPlaying.playback.Track
					if track.ID == "" {
						m.alert = "no track currently playing"
						return m, nil
					}
					m.spotifyItem = NewSpotifyItemModel(track)
					m.spotifyItemPrev = menuMode
					m.state = spotifyItemMode
					return m, nil

					// case "like":
					// 	track := m.nowPlaying.playback.Track
					// 	if track.ID == "" {
					// 		m.alert = "no track currently playing"
					// 		return m, nil
					// 	}
					// 	return m, sp.SpotifyActionCmd(sp.SpotifyActionMsg{Command: sp.CmdLike, Arg: string(track.ID)})
				}

				cmdStr := sp.SpotifyCommand(strings.ToUpper(parts[0]))
				if !sp.IsValidSpotifyCommand(cmdStr) {
					TerminalLog.Printf("unknown command: %q\n", parts[0])
					m.alert = fmt.Sprintf("unknown command: %q", parts[0])
					return m, nil
				}

				TerminalLog.Printf("command: %q arg: %q\n", cmdStr, arg)
				return m, sp.SpotifyActionCmd(sp.SpotifyActionMsg{Command: cmdStr, Arg: arg})

			default:
				m.textInput, cmd = m.textInput.Update(msg)
			}
		}

	// Forward unrecognized messages to textInput when in terminal mode so that
	// textinput-internal types (e.g. pasteMsg from its ctrl+v Paste command) are handled.
	default:
		if m.state == terminalMode {
			m.textInput, cmd = m.textInput.Update(msg)
		}
	}

	return m, cmd
}

func (m Menu) View() tea.View {
	switch m.state {
	case helpMode:
		return m.guide.View()

	case searchResultsMode:
		helpBar := m.help.View(m.searchResults.Keys)
		v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left,
			m.searchResults.View(),
			m.searchItemBottom(helpBar),
		))
		v.AltScreen = true
		return v

	case spotifyItemMode:
		helpBar := m.help.View(m.spotifyItem.Keys)
		v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left,
			m.spotifyItem.View(),
			m.searchItemBottom(helpBar),
		))
		v.AltScreen = true
		return v
	}

	// Agent tab fills the content area without center-placement.
	if m.currentTab == tabAgent {
		bottomBar := m.bottomBar()
		composed := lipgloss.JoinVertical(lipgloss.Left, m.agentTab.View(), bottomBar)
		v := tea.NewView(composed)
		v.AltScreen = true
		return v
	}

	tabContent := m.activeTabContent()
	bottomBar := m.bottomBar()
	return m.renderLayout(tabContent, bottomBar)
}

func (m Menu) activeTabContent() string {
	switch m.currentTab {
	case tabNowPlaying:
		return m.nowPlaying.View().Content
	default:
		return m.modalitiesList.View().Content
	}
}

func (m Menu) bottomBar() string {
	if m.currentTab == tabAgent && m.state == menuMode {
		helpBar := m.help.View(m.agentTab.chat.keys)
		if m.alert != "" {
			return lipgloss.JoinVertical(lipgloss.Left, alertStyle.Render("! "+m.alert), helpBar)
		}
		return helpBar
	}

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
		if m.successAlert != "" {
			return lipgloss.JoinVertical(lipgloss.Left,
				successAlertStyle.Render("+ "+m.successAlert),
				m.help.View(m.menuKeys),
			)
		}
		return m.help.View(m.menuKeys)
	}
}

func (m Menu) searchItemBottom(helpBar string) string {
	if m.alert != "" {
		return lipgloss.JoinVertical(lipgloss.Left, alertStyle.Render("! "+m.alert), helpBar)
	}
	if m.successAlert != "" {
		return lipgloss.JoinVertical(lipgloss.Left, successAlertStyle.Render("+ "+m.successAlert), helpBar)
	}
	return helpBar
}

func (m Menu) renderLayout(tabContent, bottomBar string) tea.View {
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
