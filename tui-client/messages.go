// custom bubbletea messages
package main

import (
	tea "charm.land/bubbletea/v2"
)

// Switch View
type SwitchViewMsg int

func SwitchViewCmd(view int) tea.Cmd {
	return func() tea.Msg { return SwitchViewMsg(view) }
}

// Spotify Actions
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

// SpotifyRouteErrorMsg is sent back to the active view when SpotifyClient.Route
// returns an error, so the UI can surface it to the user.
type SpotifyRouteErrorMsg string

// validSpotifyCommands is the authoritative set for command-name validation.
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
