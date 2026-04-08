package main

import (
	"fmt"
	"log"
	"os"

	tea "charm.land/bubbletea/v2"
)

// TODO create tui
// multiple views
// - form/main menu to enable multimodes
// minimalist "vibe" view
// active session (see how long they've been listening to music, avg session length, and sum total session time)

// command ideas: pause, play, skipf, skipb

func main() {
	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}
	p := tea.NewProgram(NewModel())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

const (
	MenuViewIdx int = iota
	GuideViewIdx
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
			NewGuide(),
		},
	}
}

func (m Model) Init() tea.Cmd {
	return m.views[m.active].Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.views[m.active], cmd = m.views[m.active].Update(msg)

	// TODO: refactor to propagate messages to proper screen with Update method. Look at help.go 39 too
	switch msg := msg.(type) {
	case switchViewMsg:
		m.active = int(msg)
		return m, m.views[m.active].Init()
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
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
