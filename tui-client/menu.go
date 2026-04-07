package main

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type menu struct {
	items     []string
	selected  int
	textInput textinput.Model
}

func NewMenu() menu {
	ti := textinput.New()
	ti.Placeholder = "Type a command..."

	return menu{
		items:     []string{"H", "V", "A"},
		selected:  0,
		textInput: ti,
	}
}

func (m menu) Init() tea.Cmd {
	return nil
}

func (m menu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)

	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "left":
			if m.selected > 0 {
				m.selected--
			}
		case "right":
			if m.selected < len(m.items)-1 {
				m.selected++
			}
		}
	}

	return m, cmd
}

func (m menu) View() tea.View {
	itemStyle := lipgloss.NewStyle().Padding(0, 1)
	selectedStyle := lipgloss.NewStyle().Padding(0, 1).Bold(true).Foreground(lipgloss.Color("2"))

	renderedItems := make([]string, len(m.items))
	for i, item := range m.items {
		if i == m.selected {
			renderedItems[i] = selectedStyle.Render(item)
		} else {
			renderedItems[i] = itemStyle.Render(item)
		}
	}

	horizontalList := lipgloss.JoinHorizontal(lipgloss.Top, renderedItems...)

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left,
		"NAME",
		horizontalList,
		m.textInput.View(),
	))
}
