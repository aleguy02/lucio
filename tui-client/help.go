package main

import (
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

type help struct {
	list list.Model
}

func NewHelp() help {
	items := []list.Item{
		item{title: "q / ctrl+c", desc: "Quit"},
		item{title: "h", desc: "Help page"},
	}
	list := list.New(items, list.NewDefaultDelegate(), 0, 0)
	list.Title = "Help"
	return help{
		list: list,
	}
}

func (m help) Init() tea.Cmd {
	return nil
}

func (m help) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return m, SwitchViewCmd(MenuViewIdx)
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m help) View() tea.View {
	v := tea.NewView(docStyle.Render(m.list.View()))
	v.AltScreen = true
	return v
}
