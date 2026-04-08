package main

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type guideKeyMap struct {
	Back key.Binding
}

func (k guideKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Back}
}

func (k guideKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Back}}
}

type guide struct {
	list    list.Model
	keys    guideKeyMap
	helpBar help.Model
}

func NewGuide() guide {
	const defaultWidth, defaultHeight = 40, 20
	items := []list.Item{
		item{title: "q / ctrl+c", desc: "Quit"},
		item{title: "h", desc: "Help page"},
	}
	l := list.New(items, list.NewDefaultDelegate(), defaultWidth, defaultHeight)
	l.Title = "Help"

	return guide{
		list: l,
		keys: guideKeyMap{
			Back: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "back"),
			),
		},
		helpBar: help.New(),
	}
}

func (m guide) Init() tea.Cmd {
	return nil
}

func (m guide) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, m.keys.Back) {
			return m, SwitchViewCmd(MenuViewIdx)
		}
	case tea.WindowSizeMsg:
		// h, v := docStyle.GetFrameSize()
		// m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m guide) View() tea.View {
	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left,
		docStyle.Render(m.list.View()),
		m.helpBar.View(m.keys),
	))
	v.AltScreen = true
	return v
}
