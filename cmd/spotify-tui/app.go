package main

import (
	"context"
	"fmt"
	"log"
	"time"

	sp "aleguy02/spotify-tui/internal/spotify"
	"aleguy02/spotify-tui/internal/ui"

	tea "charm.land/bubbletea/v2"
	adkagent "google.golang.org/adk/agent"
	adkrunner "google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool/toolconfirmation"
	"google.golang.org/genai"
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


// Agent-layer function to produce an event stream
func runAgentStream(r *adkrunner.Runner, ctx context.Context, text string, streamCh chan tea.Msg, funcID string, confirmed bool) {
	defer close(streamCh)

	var msg *genai.Content
	if funcID != "" {
		msg = &genai.Content{
			Role: "user",
			Parts: []*genai.Part{
				{
					FunctionResponse: &genai.FunctionResponse{
						ID:   funcID, 
						Name: "adk_request_confirmation", 
						Response: map[string]any{
							"confirmed": confirmed,
						},
					},
				},
			},
		}
	} else {
		msg = &genai.Content{
			Parts: []*genai.Part{{Text: text}},
			Role:  "user",
		}
	}

	for event, err := range r.Run(ctx, agentUserID, agentSessionID, msg, adkagent.RunConfig{StreamingMode: adkagent.StreamingModeSSE}) {
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
				if part.FunctionCall.Name == toolconfirmation.FunctionCallName {
					orig, err := toolconfirmation.OriginalCallFrom(part.FunctionCall)
					if err != nil {
						log.Printf("[agent] failed to unwrap confirmation request: %v", err)
						streamCh <- ui.AgentChunkMsg{ConfirmRequired: true, ToolName: part.FunctionCall.Name, ToolArgs: part.FunctionCall.Args, ToolID: part.FunctionCall.ID}
						continue
					}
					streamCh <- ui.AgentChunkMsg{ConfirmRequired: true, ToolName: orig.Name, ToolArgs: orig.Args, ToolID: part.FunctionCall.ID}
					continue
				}
				streamCh <- ui.AgentChunkMsg{ToolName: part.FunctionCall.Name, ToolArgs: part.FunctionCall.Args, ToolID: part.FunctionCall.ID}
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
			results, err := m.spotifyClient.HandleSearch(msg, 0)
			if err != nil {
				errMsg := sp.SpotifyRouteErrorMsg(err.Error())
				return m, func() tea.Msg { return errMsg }
			}
			return m, func() tea.Msg { return sp.SearchResultsMsg{Items: results, Action: msg} }
		}
		if msg.Command == sp.CmdPlaylists {
			results, _, hasMore, err := m.spotifyClient.HandlePlaylists(7, 0)
			if err != nil {
				errMsg := sp.SpotifyRouteErrorMsg(err.Error())
				return m, func() tea.Msg { return errMsg }
			}
			var action sp.SpotifyActionMsg
			if hasMore {
				action = sp.SpotifyActionMsg{Command: sp.CmdPlaylists}
			}
			return m, func() tea.Msg { return sp.SearchResultsMsg{Items: results, Action: action} }
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
			return m, func() tea.Msg { return errMsg }
		}
		if result != "" {
			return m, func() tea.Msg { return sp.DevicesResultMsg(result) }
		}
		return m, nil

	case sp.SearchMoreMsg:
		if msg.Action.Command == sp.CmdPlaylists {
			results, _, hasMore, err := m.spotifyClient.HandlePlaylists(7, msg.Offset)
			if err != nil {
				errMsg := sp.SpotifyRouteErrorMsg(err.Error())
				return m, func() tea.Msg { return errMsg }
			}
			return m, func() tea.Msg { return sp.SearchMoreResultsMsg{Items: results, HasMore: hasMore} }
		}
		results, err := m.spotifyClient.HandleSearch(msg.Action, msg.Offset)
		if err != nil {
			errMsg := sp.SpotifyRouteErrorMsg(err.Error())
			return m, func() tea.Msg { return errMsg }
		}
		return m, func() tea.Msg { return sp.SearchMoreResultsMsg{Items: results, HasMore: len(results) > 0} }

	case sp.PlaybackMsg:
		if err := m.spotifyClient.ExecutePlayback(msg); err != nil {
			errMsg := sp.SpotifyRouteErrorMsg(err.Error())
			return m, func() tea.Msg { return errMsg }
		}
		return m, nil

	case sp.QueueMsg:
		if err := m.spotifyClient.QueueSong(msg); err != nil {
			errMsg := sp.SpotifyRouteErrorMsg(err.Error())
			return m, func() tea.Msg { return errMsg }
		}
		label := "queued"
		if msg.Name != "" {
			label = fmt.Sprintf("queued: %s", msg.Name)
		}
		return m, func() tea.Msg { return sp.QueueSuccessMsg(label) }

	case ui.ToolConfirmationMsg:
		if m.agentRunner != nil {
			ch := make(chan tea.Msg)
			m.agentChan = ch
			go runAgentStream(m.agentRunner, context.Background(), "", ch, msg.ID, msg.Confirmed)
			return m, tea.Batch(cmd, waitForAgentChunkCmd(ch))
		}
		return m, cmd

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
			go runAgentStream(m.agentRunner, context.Background(), msg.Text, ch, "", false)
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
			return m, tea.Batch(func() tea.Msg { return err }, doTick())
		}
		stateMsg := sp.SpotifyPlaybackStateMsg{State: state}
		m.views[0], _ = m.views[0].Update(stateMsg)
		return m, doTick()

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if activeMenu, ok := m.views[m.active].(ui.Menu); ok && (activeMenu.IsInTerminalMode() || activeMenu.IsOnAgentTab()) {
				break
			}
			return m, tea.Quit
		}
	}
	return m, cmd
}

func (m Model) View() tea.View {
	return m.views[m.active].View()
}
