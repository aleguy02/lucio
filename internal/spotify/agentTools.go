package spotify

import (
	"google.golang.org/adk/tool"
)

// credit to https://github.com/marcelmarais/spotify-mcp-server
///////// READ TOOLS /////////

type SearchSpotifyToolArgs struct {
	SearchType int    `json:"spotify_type"          jsonschema:"The Spotify item type as an integer. album=1 artist=2 playlist=4 track=8"`
	Query      string `json:"query"     jsonschema:"Search query"`
}

type SearchSpotifyToolResult struct {
	Results []SpotifyItem `json:"search_results"`
	Success bool          `json:"success"`
}

// SearchSpotify searches for tracks, albums, artists, or playlists on Spotify
// based on the provided query and item type.
//
// It returns a slice of matching items containing their IDs, names, and
// additional metadata.
func (c *SpotifyClient) SearchSpotify(ctx tool.Context, args SearchSpotifyToolArgs) (SearchSpotifyToolResult, error) {
	// pack type and query into spotifyactionmsg
	var subcommand string
	switch args.SearchType {
	case 1:
		subcommand = "album"
	case 2:
		subcommand = "artist"
	case 4:
		subcommand = "playlist"
	case 8:
		subcommand = "track"
	}
	msg := SpotifyActionMsg{
		Command: "SEARCH", // This parameter is used in the TUI logic. Here, it is just for documentation
		Arg:     subcommand + " " + args.Query,
	}
	res, err := c.HandleSearch(msg)
	if err != nil {
		return SearchSpotifyToolResult{Success: false}, err
	}
	return SearchSpotifyToolResult{Results: res, Success: true}, nil
}

type GetNowPlayingToolResult struct{}

func (c *SpotifyClient) GetNowPlaying(ctx tool.Context, _ struct{}) (GetNowPlayingToolResult, error) {
	return GetNowPlayingToolResult{}, nil
}

type GetUserPlaylistsToolArgs struct {
	Limit  int `json:"limit" jsonschema:"Maximum number of playlists to return. Default value is 10 for single tool calls"`
	Offset int `json:"offset" jsonschema:"Index of the first playlist to return."`
}

type GetUserPlaylistsToolResult struct {
	Playlists []SpotifyItem `json:"playlists"`
	Total     int           `json:"total"`
	HasMore   bool          `json:"has_more"`
	Success   bool          `json:"success"`
}

// Get a list of the current user's playlists on Spotify.
//
// It returns an array of playlists packed into SpotifyItems.
func (c *SpotifyClient) GetUserPlaylistsTool(ctx tool.Context, args GetUserPlaylistsToolArgs) (GetUserPlaylistsToolResult, error) {
	results, total, hasMore, err := c.HandlePlaylists(args.Limit, args.Offset)
	if err != nil {
		return GetUserPlaylistsToolResult{Success: false}, err
	}
	return GetUserPlaylistsToolResult{
		Playlists: results,
		Total:     total,
		HasMore:   hasMore,
		Success:   true,
	}, nil
}

type GetPlaylistTracksToolResult struct{}

func (c *SpotifyClient) GetPlaylistTracks(ctx tool.Context, _ struct{}) (GetPlaylistTracksToolResult, error) {
	return GetPlaylistTracksToolResult{}, nil
}

type GetRecentlyPlayedToolResult struct{}

func (c *SpotifyClient) GetRecentlyPlayed(ctx tool.Context, _ struct{}) (GetRecentlyPlayedToolResult, error) {
	return GetRecentlyPlayedToolResult{}, nil
}

type RemoveUsersSavedTracksToolResult struct{}

func (c *SpotifyClient) RemoveUsersSavedTracks(ctx tool.Context, _ struct{}) (RemoveUsersSavedTracksToolResult, error) {
	return RemoveUsersSavedTracksToolResult{}, nil
}

// /////// PLAY/CREATE TOOLS /////////
type PlayItemToolArgs = PlaybackMsg

type PlayItemToolResult struct {
	Success bool `json:"success"`
}

func (c *SpotifyClient) PlayItemTool(ctx tool.Context, args PlayItemToolArgs) (PlayItemToolResult, error) {
	if err := c.ExecutePlayback(args); err != nil {
		return PlayItemToolResult{Success: false}, err
	}
	return PlayItemToolResult{Success: true}, nil
}

type SkipNextToolResult struct {
	Success bool `json:"success"`
}

func (c *SpotifyClient) SkipNextTool(ctx tool.Context, _ struct{}) (SkipNextToolResult, error) {
	if err := c.skipForward(); err != nil {
		return SkipNextToolResult{
			Success: false,
		}, err
	}

	return SkipNextToolResult{
		Success: true,
	}, nil
}

type SkipPreviousToolResult struct {
	Success bool `json:"success"`
}

func (c *SpotifyClient) SkipPreviousTool(ctx tool.Context, _ struct{}) (SkipPreviousToolResult, error) {
	if err := c.skipBack(); err != nil {
		return SkipPreviousToolResult{
			Success: false,
		}, err
	}

	return SkipPreviousToolResult{
		Success: true,
	}, nil
}

type CreatePlaylistToolResult struct{}

func (c *SpotifyClient) CreatePlaylist(ctx tool.Context, _ struct{}) (CreatePlaylistToolResult, error) {
	return CreatePlaylistToolResult{}, nil
}

type AddTracksToPlaylistToolResult struct{}

func (c *SpotifyClient) AddTracksToPlaylist(ctx tool.Context, _ struct{}) (AddTracksToPlaylistToolResult, error) {
	return AddTracksToPlaylistToolResult{}, nil
}

type AddToQueueToolArgs = QueueMsg

type AddToQueueToolResult struct {
	Success bool `json:"success"`
}

func (c *SpotifyClient) AddToQueue(ctx tool.Context, args AddToQueueToolArgs) (AddToQueueToolResult, error) {
	err := c.QueueSong(args)
	if err != nil {
		return AddToQueueToolResult{Success: false}, err
	}
	return AddToQueueToolResult{Success: true}, nil
}

// /////// ALBUM TOOLS /////////
type GetAlbumsToolResult struct{}

func (c *SpotifyClient) GetAlbums(ctx tool.Context, _ struct{}) (GetAlbumsToolResult, error) {
	return GetAlbumsToolResult{}, nil
}

type GetAlbumTracksToolResult struct{}

func (c *SpotifyClient) GetAlbumTracks(ctx tool.Context, _ struct{}) (GetAlbumTracksToolResult, error) {
	return GetAlbumTracksToolResult{}, nil
}

type SaveOrRemoveAlbumsForUserToolResult struct{}

func (c *SpotifyClient) SaveOrRemoveAlbumsForUser(ctx tool.Context, _ struct{}) (SaveOrRemoveAlbumsForUserToolResult, error) {
	return SaveOrRemoveAlbumsForUserToolResult{}, nil
}

type CheckUsersSavedAlbumsToolResult struct{}

func (c *SpotifyClient) CheckUsersSavedAlbums(ctx tool.Context, _ struct{}) (CheckUsersSavedAlbumsToolResult, error) {
	return CheckUsersSavedAlbumsToolResult{}, nil
}

// /////// PLAYLIST TOOLS /////////
type GetPlaylistToolResult struct{}

func (c *SpotifyClient) GetPlaylist(ctx tool.Context, _ struct{}) (GetPlaylistToolResult, error) {
	return GetPlaylistToolResult{}, nil
}

type UpdatePlaylistToolResult struct{}

func (c *SpotifyClient) UpdatePlaylist(ctx tool.Context, _ struct{}) (UpdatePlaylistToolResult, error) {
	return UpdatePlaylistToolResult{}, nil
}

type RemoveTracksFromPlaylistToolResult struct{}

func (c *SpotifyClient) RemoveTracksFromPlaylist(ctx tool.Context, _ struct{}) (RemoveTracksFromPlaylistToolResult, error) {
	return RemoveTracksFromPlaylistToolResult{}, nil
}
