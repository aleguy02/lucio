package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	tea "charm.land/bubbletea/v2"
)

// Paths to the gesture client and its Python interpreter, relative to the
// working directory when the TUI is launched (typically tui-client/).
const (
	gesturePython     = "../.venv/bin/python"
	gestureSenderPath = "../sender.py"
)

// TODO
// multiple views
// minimalist "vibe" view
// active session (see how long they've been listening to music, avg session length, and sum total session time)

func main() {
	// Always log to a file: once BubbleTea takes over the terminal,
	// anything written to stderr is invisible.
	f, err := tea.LogToFile("debug.log", "")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer f.Close()

	spotifyClient, err := NewSpotifyClient()
	if err != nil {
		log.Fatal("Spotify setup failed: ", err)
	}

	p := tea.NewProgram(NewModel(spotifyClient))
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

const (
	MenuViewIdx int = iota
)

type Model struct {
	// menu, vibe, help, stats
	active        int
	views         []tea.Model
	spotifyClient *SpotifyClient
	gestureCancel context.CancelFunc // nil when gesture server is not running
	gestureChan   chan tea.Msg
	gestureProc   *os.Process // nil when gesture client subprocess is not running
}

func NewModel(spotifyClient *SpotifyClient) *Model {
	return &Model{
		active: MenuViewIdx,
		views: []tea.Model{
			NewMenu(),
		},
		spotifyClient: spotifyClient,
	}
}

func (m Model) Init() tea.Cmd {
	return m.views[m.active].Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.views[m.active], cmd = m.views[m.active].Update(msg)

	switch msg := msg.(type) {
	case SpotifyActionMsg:
		if msg.Command == CmdSearch {
			results, err := m.spotifyClient.HandleSearch(msg)
			if err != nil {
				errMsg := SpotifyRouteErrorMsg(err.Error())
				return m, tea.Batch(func() tea.Msg { return errMsg }, WaitForGestureCmd(m.gestureChan))
			}
			return m, tea.Batch(
				func() tea.Msg { return SearchResultsMsg(results) },
				WaitForGestureCmd(m.gestureChan),
			)
		}

		if err := m.spotifyClient.Route(msg); err != nil {
			errMsg := SpotifyRouteErrorMsg(err.Error())
			return m, tea.Batch(func() tea.Msg { return errMsg }, WaitForGestureCmd(m.gestureChan))
		}
		return m, WaitForGestureCmd(m.gestureChan)

	case PlaybackMsg:
		if err := m.spotifyClient.ExecutePlayback(msg); err != nil {
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
				cancel()
				m.gestureCancel = nil
				m.gestureChan = nil
				return m, nil
			}

			pythonAbs, _ := filepath.Abs(gesturePython)
			senderAbs, _ := filepath.Abs(gestureSenderPath)
			proc, err := launchGestureClient(pythonAbs, senderAbs, ch)
			if err != nil {
				log.Printf("gesture client failed to start: %v\n", err)
				cancel()
				m.gestureCancel = nil
				m.gestureChan = nil
				return m, nil
			}
			m.gestureProc = proc
			return m, WaitForGestureCmd(ch)

		} else {
			m.stopGestureClient()
			// gestureChan stays open; it will be closed by GestureClientExitedMsg
			// once the subprocess actually exits and the monitoring goroutine fires.
		}

	case GestureClientExitedMsg:
		// Ignore stale notifications from a previous session.
		if msg.Ch != m.gestureChan {
			break
		}
		if msg.Err != nil {
			log.Printf("gesture client exited with error: %v\n", msg.Err)
		}
		m.gestureProc = nil
		if m.gestureCancel != nil {
			m.gestureCancel()
			m.gestureCancel = nil
		}
		close(m.gestureChan)
		m.gestureChan = nil
		// Channel is closed — do NOT issue waitForGesture.

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			m.stopGestureClient()
			if m.gestureCancel != nil {
				m.gestureCancel()
			}
			return m, tea.Quit
		case "q":
			// Block quit while the menu's text input is active.
			if activeMenu, ok := m.views[m.active].(menu); ok && activeMenu.state == terminalMode {
				break
			}
			m.stopGestureClient()
			if m.gestureCancel != nil {
				m.gestureCancel()
			}
			return m, tea.Quit
		}
	}
	return m, cmd
}

// stopGestureClient sends SIGTERM to the gesture subprocess and cancels the
// server context.  The channel is left open; it will be closed by the
// GestureClientExitedMsg handler once cmd.Wait() returns.
func (m *Model) stopGestureClient() {
	if m.gestureProc != nil {
		if err := m.gestureProc.Signal(syscall.SIGTERM); err != nil {
			log.Printf("failed to signal gesture client: %v\n", err)
		}
		m.gestureProc = nil
	}
	if m.gestureCancel != nil {
		m.gestureCancel()
		m.gestureCancel = nil
	}
}

// launchGestureClient starts sender.py under the venv Python interpreter.
// It sets the subprocess working directory to the folder containing sender.py
// so that relative asset paths (gesture_recognizer.task) resolve correctly.
// A monitoring goroutine sends GestureClientExitedMsg to ch when the process exits.
func launchGestureClient(pythonPath, senderPath string, ch chan tea.Msg) (*os.Process, error) {
	cmd := exec.Command(pythonPath, senderPath, "--socket", SOCKET_PATH) // TODO: make headless an option? probably not but we can make the window look nice
	cmd.Dir = filepath.Dir(senderPath)
	// Stdout/Stderr are nil → discarded (BubbleTea owns the terminal).
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	log.Printf("gesture client started, PID %d\n", cmd.Process.Pid)

	go func() {
		err := cmd.Wait()
		log.Printf("gesture client exited: %v\n", err)
		ch <- GestureClientExitedMsg{Ch: ch, Err: err}
	}()

	return cmd.Process, nil
}

func (m Model) View() tea.View {
	return m.views[m.active].View()
}
