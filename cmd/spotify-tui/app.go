package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"aleguy02/spotify-tui/internal/gestures"
	sp "aleguy02/spotify-tui/internal/spotify"
	"aleguy02/spotify-tui/internal/ui"

	tea "charm.land/bubbletea/v2"
)

// Paths to the gesture client and its Python interpreter, relative to CWD
// when the app is launched (repo root).
const (
	gesturePython     = ".venv/bin/python"
	gestureSenderPath = "scripts/sender.py"
)

type PSTickMsg string

func doTick() tea.Cmd {
	return tea.Tick(time.Second * 3600, func(t time.Time) tea.Msg {
		return PSTickMsg("hi")
	})
}

type Model struct {
	active        int
	views         []tea.Model
	spotifyClient *sp.SpotifyClient
	gestureCancel context.CancelFunc
	gestureChan   chan tea.Msg
	gestureProc   *os.Process
}

func newModel(client *sp.SpotifyClient) *Model {
	return &Model{
		active:        0,
		views:         []tea.Model{ui.NewMenu()},
		spotifyClient: client,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.views[m.active].Init(), doTick())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.views[m.active], cmd = m.views[m.active].Update(msg)

	switch msg := msg.(type) {
	case sp.SpotifyActionMsg:
		if msg.Command == sp.CmdSearch {
			results, err := m.spotifyClient.HandleSearch(msg)
			if err != nil {
				errMsg := sp.SpotifyRouteErrorMsg(err.Error())
				return m, tea.Batch(func() tea.Msg { return errMsg }, gestures.WaitForGestureCmd(m.gestureChan))
			}
			return m, tea.Batch(
				func() tea.Msg { return sp.SearchResultsMsg(results) },
				gestures.WaitForGestureCmd(m.gestureChan),
			)
		}
		result, err := m.spotifyClient.Route(msg)
		if err != nil {
			errMsg := sp.SpotifyRouteErrorMsg(err.Error())
			return m, tea.Batch(func() tea.Msg { return errMsg }, gestures.WaitForGestureCmd(m.gestureChan))
		}
		if result != "" {
			return m, tea.Batch(
				func() tea.Msg { return sp.DevicesResultMsg(result) },
				gestures.WaitForGestureCmd(m.gestureChan),
			)
		}
		return m, gestures.WaitForGestureCmd(m.gestureChan)

	case sp.PlaybackMsg:
		if err := m.spotifyClient.ExecutePlayback(msg); err != nil {
			errMsg := sp.SpotifyRouteErrorMsg(err.Error())
			return m, tea.Batch(func() tea.Msg { return errMsg }, gestures.WaitForGestureCmd(m.gestureChan))
		}
		return m, gestures.WaitForGestureCmd(m.gestureChan)

	case sp.QueueMsg:
		if err := m.spotifyClient.QueueSong(msg); err != nil {
			errMsg := sp.SpotifyRouteErrorMsg(err.Error())
			return m, tea.Batch(func() tea.Msg { return errMsg }, gestures.WaitForGestureCmd(m.gestureChan))
		}
		label := "queued"
		if msg.Name != "" {
			label = fmt.Sprintf("queued: %s", msg.Name)
		}
		return m, tea.Batch(func() tea.Msg { return sp.QueueSuccessMsg(label) }, gestures.WaitForGestureCmd(m.gestureChan))

	case PSTickMsg:
		state, err := m.spotifyClient.GetPlaybackState()
		if err != nil {
			log.Println("[nowPlaying]", err)
			return m, tea.Batch(func() tea.Msg { return err }, gestures.WaitForGestureCmd(m.gestureChan), doTick())
		}
		stateMsg := sp.SpotifyPlaybackStateMsg{State: state}
		m.views[0], _ = m.views[0].Update(stateMsg)
		return m, tea.Batch(gestures.WaitForGestureCmd(m.gestureChan), doTick())

	case gestures.ToggleGesturesMsg:
		if bool(msg) {
			ch := make(chan tea.Msg)
			ctx, cancel := context.WithCancel(context.Background())
			m.gestureChan = ch
			m.gestureCancel = cancel

			if err := gestures.StartGestureServer(ctx, ch); err != nil {
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
			return m, gestures.WaitForGestureCmd(ch)
		}
		m.stopGestureClient()

	case gestures.GestureClientExitedMsg:
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

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			m.stopGestureClient()
			if m.gestureCancel != nil {
				m.gestureCancel()
			}
			return m, tea.Quit
		case "q":
			if activeMenu, ok := m.views[m.active].(ui.Menu); ok && (activeMenu.IsInTerminalMode() || activeMenu.IsOnAgentTab()) {
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

// stopGestureClient sends SIGTERM to the gesture subprocess and cancels the server context.
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
// Inherits the parent's CWD (repo root) so sender.py can find gesture_recognizer.task.
func launchGestureClient(pythonPath, senderPath string, ch chan tea.Msg) (*os.Process, error) {
	cmd := exec.Command(pythonPath, senderPath, "--socket", gestures.SocketPath)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	log.Printf("gesture client started, PID %d\n", cmd.Process.Pid)

	go func() {
		err := cmd.Wait()
		log.Printf("gesture client exited: %v\n", err)
		ch <- gestures.GestureClientExitedMsg{Ch: ch, Err: err}
	}()

	return cmd.Process, nil
}

func (m Model) View() tea.View {
	return m.views[m.active].View()
}
