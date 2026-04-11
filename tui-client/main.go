package main

import (
	"context"
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

type Model struct {
	// menu, vibe, help, stats
	active         int
	views          []tea.Model
	spotifyClient  *SpotifyClient
	gestureCancel  context.CancelFunc // nil when gesture server is not running
}

func NewModel() *Model {
	return &Model{
		active: MenuViewIdx,
		views: []tea.Model{
			NewMenu(),
			NewGuide(),
		},
		spotifyClient: NewSpotifyClient(),
	}
}

func (m Model) Init() tea.Cmd {
	return m.views[m.active].Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.views[m.active], cmd = m.views[m.active].Update(msg)

	switch msg := msg.(type) {
	case SwitchViewMsg:
		m.active = int(msg)
		return m, m.views[m.active].Init()
	case SpotifyActionMsg:
		if err := m.spotifyClient.Route(msg); err != nil {
			errMsg := SpotifyRouteErrorMsg(err.Error())
			return m, func() tea.Msg { return errMsg }
		}
		return m, nil
	case ToggleGesturesMsg:
		if bool(msg) {
			log.Println("enabling gesture server")
			ctx, cancel := context.WithCancel(context.Background())
			m.gestureCancel = cancel
			if err := startGestureServer(ctx); err != nil {
				log.Println("gesture server failed to start:", err)
				m.gestureCancel = nil
			}
		} else {
			log.Println("disabling gesture server")
			if m.gestureCancel != nil {
				m.gestureCancel()
				m.gestureCancel = nil
			}
		}
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.gestureCancel != nil {
				m.gestureCancel()
			}
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
