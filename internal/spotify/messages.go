package spotify

import (
	tea "charm.land/bubbletea/v2"
	zmb "github.com/zmb3/spotify/v2"
)

type SpotifyCommand string

const (
	CmdPlay      SpotifyCommand = "PLAY"
	CmdPause     SpotifyCommand = "PAUSE"
	CmdSkipF     SpotifyCommand = "SKIPF"
	CmdSkipB     SpotifyCommand = "SKIPB"
	CmdSeekF     SpotifyCommand = "SEEKF"
	CmdSeekB     SpotifyCommand = "SEEKB"
	CmdSearch    SpotifyCommand = "SEARCH"
	CmdDevices   SpotifyCommand = "DEVICES"
	CmdPlaylists SpotifyCommand = "PLAYLISTS"
	CmdShuffle   SpotifyCommand = "SHUFFLE"
	// CmdLike      SpotifyCommand = "LIKE"
)

var validSpotifyCommands = map[SpotifyCommand]bool{
	CmdPlay: true, CmdPause: true, CmdSkipF: true, CmdSkipB: true,
	CmdSeekF: true, CmdSeekB: true, CmdSearch: true, CmdDevices: true,
	CmdPlaylists: true, CmdShuffle: true,
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
type SearchResultsMsg struct {
	Items  []SpotifyItem
	Action SpotifyActionMsg // zero value → no lazy loading
}

// SearchMoreMsg is emitted by the search list when the user scrolls past the last item.
type SearchMoreMsg struct {
	Action SpotifyActionMsg
	Offset int
}

// SearchMoreResultsMsg carries the next page of results to append to the list.
type SearchMoreResultsMsg struct {
	Items   []SpotifyItem
	HasMore bool
}

// PlaybackMsg requests playback of a Spotify item.
type PlaybackMsg struct {
	Type zmb.SearchType `json:"spotify_type"          jsonschema:"The Spotify item type as an integer. album=1 artist=2 playlist=4 track=8"`
	ID   zmb.ID         `json:"spotify_id,omitempty"  jsonschema:"The Spotify item ID. Required for tracks."`
	URI  zmb.URI        `json:"spotify_uri,omitempty" jsonschema:"The Spotify URI. Required for albums, artists, and playlists."`
}

type QueueMsg struct {
	Id   string `json:"spotify_id"  jsonschema:"The Spotify track ID"`
	Name string `json:"name"`
}

type QueueSuccessMsg string

// DevicesResultMsg carries the comma-separated list of available Spotify devices.
type DevicesResultMsg string

// type LikeSuccessMsg string
