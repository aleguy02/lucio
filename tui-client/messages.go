// custom bubbletea messages
package main

import (
	tea "charm.land/bubbletea/v2"
)

/*
 * Switch View
 */
type SwitchViewMsg int

func SwitchViewCmd(view int) tea.Cmd {
	return func() tea.Msg { return SwitchViewMsg(view) }
}

/*
 * Spotify Actions
 */
type SpotifyCommand string

const (
	CmdPlay   SpotifyCommand = "PLAY"
	CmdPause  SpotifyCommand = "PAUSE"
	CmdSkipF  SpotifyCommand = "SKIPF"
	CmdSkipB  SpotifyCommand = "SKIPB"
	CmdSeekF  SpotifyCommand = "SEEKF"
	CmdSeekB  SpotifyCommand = "SEEKB"
	CmdSearch SpotifyCommand = "SEARCH"
)

type SpotifyActionMsg struct {
	Command SpotifyCommand
	Arg     string
}

func SpotifyActionCmd(msg SpotifyActionMsg) tea.Cmd {
	return func() tea.Msg { return msg }
}

type SpotifyPlaybackStateMsg struct {
	State PlaybackState
}

// SpotifyRouteErrorMsg is sent back to the active view when SpotifyClient.Route
// returns an error, so the UI can surface it to the user.
type SpotifyRouteErrorMsg string

var validSpotifyCommands = map[SpotifyCommand]bool{
	CmdPlay:   true,
	CmdPause:  true,
	CmdSkipF:  true,
	CmdSkipB:  true,
	CmdSeekF:  true,
	CmdSeekB:  true,
	CmdSearch: true,
}

func IsValidSpotifyCommand(cmd SpotifyCommand) bool {
	return validSpotifyCommands[cmd]
}

/*
 * Hand Gesture Server
 */

type ToggleGesturesMsg bool

func ToggleGesturesCmd(toggle bool) tea.Cmd {
	return func() tea.Msg { return ToggleGesturesMsg(toggle) }
}

// WaitForGestureCmd returns a Cmd that blocks until the next message arrives on
// ch.  Returns nil when ch is nil (gesture modality is off), which BubbleTea
// treats as a no-op.
func WaitForGestureCmd(ch chan tea.Msg) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg { return <-ch }
}

// GestureClientExitedMsg is delivered when the gesture subprocess exits (expected or not).
// Ch identifies which gesture session this belongs to so stale notifications from a
// previous session can be ignored if the modality was toggled rapidly.
type GestureClientExitedMsg struct {
	Ch  chan tea.Msg
	Err error
}

/*
 * Search
 */

// SearchResultsMsg carries results from a completed search back to the active view.
type SearchResultsMsg []SpotifyItem

// PlaybackMsg requests playback of a Spotify item.
// The appropriate Spotify endpoint is selected in SpotifyClient.ExecutePlayback based on Item.Type,
// making it easy to add album, artist, and playlist playback in the future.
type PlaybackMsg struct {
	Item SpotifyItem
}

type QueueMsg struct {
	Id   string
	Name string
}

type QueueSuccessMsg string

// backToMenuMsg is sent by sub-views (e.g. guide) to return to the menu browsing state.
type backToMenuMsg struct{}
