package main

import (
	"context"
	"fmt"
	"log"
	"os"

	tea "charm.land/bubbletea/v2"
)

// TODO
// multiple views
// minimalist "vibe" view
// active session (see how long they've been listening to music, avg session length, and sum total session time)

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
	gestureChan		chan tea.Msg
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
		return m, tea.Batch(m.views[m.active].Init(), WaitForGestureCmd(m.gestureChan))
	case SpotifyActionMsg:
		if err := m.spotifyClient.Route(msg); err != nil {
			errMsg := SpotifyRouteErrorMsg(err.Error())
			return m, tea.Batch(func() tea.Msg { return errMsg }, WaitForGestureCmd(m.gestureChan))
		}
		
		return m, WaitForGestureCmd(m.gestureChan)
	case ToggleGesturesMsg:
		if bool(msg) {
			ch := make(chan tea.Msg)
			ctx, cancel := context.WithCancel(context.Background())
			m.gestureChan = ch
			m.gestureCancel = cancel

			if err := startGestureServer(ctx, ch); err != nil {
				log.Println("gesture server failed to start:", err)
				m.gestureCancel = nil
				close(m.gestureChan)
				m.gestureChan = nil
				// TODO: send an Update to menu model to print error message and disable modality?
				return m, nil
			}
			return m, WaitForGestureCmd(m.gestureChan)
		} else {
			if m.gestureChan != nil {
				close(m.gestureChan)
				m.gestureChan = nil
			}
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
