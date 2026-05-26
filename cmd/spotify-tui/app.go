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
	adkagent "google.golang.org/adk/agent"
	adkrunner "google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// Paths to the gesture client and its Python interpreter, relative to CWD
// when the app is launched (repo root).
const (
	gesturePython     = ".venv/bin/python"
	gestureSenderPath = "scripts/sender.py"
)

type PSTickMsg string

func doTick() tea.Cmd {
	return tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
		return PSTickMsg("hi")
	})
}

// agentUserID and agentSessionID are fixed for this single-user TUI app.
const (
	agentUserID    = "local"
	agentSessionID = "default"
)

type Model struct {
	active         int
	views          []tea.Model
	spotifyClient  *sp.SpotifyClient
	gestureCancel  context.CancelFunc
	gestureChan    chan tea.Msg
	gestureProc    *os.Process
	agentRunner    *adkrunner.Runner
	agentChan      chan tea.Msg
	sessionService session.Service
}

func newModel(client *sp.SpotifyClient, sesh session.Service, agentRunner *adkrunner.Runner) *Model {
	return &Model{
		active:         0,
		views:          []tea.Model{ui.NewMenu()},
		spotifyClient:  client,
		agentRunner:    agentRunner,
		sessionService: sesh,
	}
}

func waitForAgentChunkCmd(ch chan tea.Msg) tea.Cmd {
	// no active agent query
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-ch // blocks until the next streamed agent event arrives
		if !ok {
			return ui.AgentChunkMsg{Done: true}
		}
		return msg
	}
}

func runAgentStream(r *adkrunner.Runner, ctx context.Context, text string, streamCh chan tea.Msg) {
	defer close(streamCh)

	userMsg := &genai.Content{
		Parts: []*genai.Part{{Text: text}},
		Role:  "user",
	}
	for event, err := range r.Run(ctx, agentUserID, agentSessionID, userMsg, adkagent.RunConfig{StreamingMode: adkagent.StreamingModeSSE}) {
		if err != nil {
			streamCh <- ui.AgentChunkMsg{Err: err, Done: true}
			return
		}
		if event == nil || event.Content == nil || event.Author == "user" {
			continue
		}
		for _, part := range event.Content.Parts {
			switch {
			case part.FunctionCall != nil:
				streamCh <- ui.AgentChunkMsg{ToolName: part.FunctionCall.Name}
			case part.Text != "" && event.Partial:
				// Only forward partial (streaming) tokens; the final non-partial event
				// contains the same text fully accumulated — forwarding it would duplicate.
				streamCh <- ui.AgentChunkMsg{Text: part.Text}
			}
		}
	}
	streamCh <- ui.AgentChunkMsg{Done: true}
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
		if msg.Command == sp.CmdPlaylists {
			results, _, _, err := m.spotifyClient.HandlePlaylists(7, 0)
			if err != nil {
				errMsg := sp.SpotifyRouteErrorMsg(err.Error())
				return m, tea.Batch(func() tea.Msg { return errMsg }, gestures.WaitForGestureCmd(m.gestureChan))
			}
			return m, tea.Batch(
				func() tea.Msg { return sp.SearchResultsMsg(results) },
				gestures.WaitForGestureCmd(m.gestureChan),
			)
		}
		// if msg.Command == sp.CmdLike {
		// 	if err := m.spotifyClient.LikeTrack(msg.Arg); err != nil {
		// 		errMsg := sp.SpotifyRouteErrorMsg(err.Error())
		// 		return m, tea.Batch(func() tea.Msg { return errMsg }, gestures.WaitForGestureCmd(m.gestureChan))
		// 	}
		// 	return m, tea.Batch(
		// 		func() tea.Msg { return sp.LikeSuccessMsg("added to liked songs") },
		// 		gestures.WaitForGestureCmd(m.gestureChan),
		// 	)
		// }
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

	case ui.AgentQueryMsg:
		if msg.Text == "/clear" {
			ch := make(chan tea.Msg, 2)
			m.agentChan = ch
			if m.sessionService != nil {
				if err := m.sessionService.Delete(context.Background(), &session.DeleteRequest{
					AppName:   "spotify-tui",
					UserID:    agentUserID,
					SessionID: agentSessionID,
				}); err != nil {
					ch <- ui.AgentChunkMsg{Err: err, Done: true}
					return m, tea.Batch(cmd, waitForAgentChunkCmd(ch))
				}
			}
			ch <- ui.AgentChunkMsg{Text: "Session cleared. You can still scroll up to view our past messages but I won't remember any of them."}
			ch <- ui.AgentChunkMsg{Done: true}
			return m, tea.Batch(cmd, waitForAgentChunkCmd(ch))
		}
		if m.agentRunner != nil {
			ch := make(chan tea.Msg)
			m.agentChan = ch
			go runAgentStream(m.agentRunner, context.Background(), msg.Text, ch)
			return m, tea.Batch(cmd, waitForAgentChunkCmd(ch))
		}
		return m, cmd

	case ui.AgentChunkMsg:
		if msg.Done || msg.Err != nil {
			m.agentChan = nil
			return m, cmd
		}
		return m, tea.Batch(cmd, waitForAgentChunkCmd(m.agentChan))

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
