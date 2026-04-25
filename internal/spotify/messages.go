package spotify

import tea "charm.land/bubbletea/v2"

type SpotifyCommand string

const (
	CmdPlay    SpotifyCommand = "PLAY"
	CmdPause   SpotifyCommand = "PAUSE"
	CmdSkipF   SpotifyCommand = "SKIPF"
	CmdSkipB   SpotifyCommand = "SKIPB"
	CmdSeekF   SpotifyCommand = "SEEKF"
	CmdSeekB   SpotifyCommand = "SEEKB"
	CmdSearch  SpotifyCommand = "SEARCH"
	CmdDevices SpotifyCommand = "DEVICES"
	CmdPlaylists SpotifyCommand = "PLAYLISTS"
)

var validSpotifyCommands = map[SpotifyCommand]bool{
	CmdPlay: true, CmdPause: true, CmdSkipF: true, CmdSkipB: true,
	CmdSeekF: true, CmdSeekB: true, CmdSearch: true, CmdDevices: true,
	CmdPlaylists: true,
}

func IsValidSpotifyCommand(cmd SpotifyCommand) bool {
	return validSpotifyCommands[cmd]
}

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

// SpotifyRouteErrorMsg is sent back to the active view when SpotifyClient.Route returns an error.
type SpotifyRouteErrorMsg string

// SearchResultsMsg carries results from a completed search back to the active view.
type SearchResultsMsg []SpotifyItem

// PlaybackMsg requests playback of a Spotify item.
type PlaybackMsg struct {
	Item SpotifyItem
}

type QueueMsg struct {
	Id   string
	Name string
}

type QueueSuccessMsg string

// DevicesResultMsg carries the comma-separated list of available Spotify devices.
type DevicesResultMsg string
