package main

import (
	tea "charm.land/bubbletea/v2"
	"log"
)

// TODO create tui
// multiple views
// - form/main menu to enable multimodes
// minimalist "vibe" view
// active session (see how long they've been listening to music, avg session length, and sum total session time)

// command ideas: pause, play, skipf, skipb

func main() {
	p := tea.NewProgram(NewModel())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

const (
	MenuViewIdx int = iota
	HelpViewIdx
)

type switchViewMsg int

func SwitchViewCmd(view int) tea.Cmd {
	return func() tea.Msg { return switchViewMsg(view) }
}

type Model struct {
	// menu, vibe, help, stats
	active int
	views  []tea.Model
}

func NewModel() *Model {
	return &Model{
		active: MenuViewIdx,
		views: []tea.Model{
			NewMenu(),
			NewHelp(),
		},
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.views[m.active], cmd = m.views[m.active].Update(msg)

	switch msg := msg.(type) {
	case switchViewMsg:
		m.active = int(msg)
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "h":
			m.active = HelpViewIdx
		case "t":
			// focus command prompt
		}
	}
	return m, cmd
}

func (m Model) View() tea.View {
	if view, ok := m.views[m.active].(tea.Model); ok {
		return view.View()
	}
	return tea.NewView("no view models :(")
}
