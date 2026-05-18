package spotify

import (
	"google.golang.org/adk/tool"
)

// credit to https://github.com/marcelmarais/spotify-mcp-server
///////// READ TOOLS /////////

type SearchSpotifyToolResult struct {}

// SearchSpotify searches for tracks, albums, artists, or playlists on Spotify
// based on the provided query and item type.
//
// It returns a slice of matching items containing their IDs, names, and
// additional metadata.
func (c *SpotifyClient) SearchSpotify(ctx tool.Context, _ struct{}) (SearchSpotifyToolResult, error) {
	return SearchSpotifyToolResult{}, nil
}

type GetNowPlayingToolResult struct {}

func (c *SpotifyClient) GetNowPlaying(ctx tool.Context, _ struct{}) (GetNowPlayingToolResult, error) {
	return GetNowPlayingToolResult{}, nil
}

// TODO(bug): the agent is omitting these parameters on the first call, every time. Is this an agent intelligence problem or a code problem? See agent.go's genaiDeclToOllamaTool func
type GetUserPlaylistsToolArgs struct {
	Limit int `json:"limit" jsonschema:"Maximum number of playlists to return. Use default value of 10 unless specified"`
	Offset int `json:"offset" jsonschema:"Index of the first playlist to return."`
} 

type GetUserPlaylistsToolResult struct {
	Playlists []SpotifyItem `json:"playlists"`
	Success bool `json:"success"`
}

// Get a list of the current user's playlists on Spotify.
//
// It returns an array of playlists packed into SpotifyItems.
func (c *SpotifyClient) GetUserPlaylistsTool(ctx tool.Context, args GetUserPlaylistsToolArgs) (GetUserPlaylistsToolResult, error) {
	results, err := c.HandlePlaylists(args.Limit, args.Offset)
	if err != nil {
		return GetUserPlaylistsToolResult{Success: false}, err
	}
	return GetUserPlaylistsToolResult{
		Playlists: results,
		Success: true,
	}, nil
}

type GetPlaylistTracksToolResult struct {}

func (c *SpotifyClient) GetPlaylistTracks(ctx tool.Context, _ struct{}) (GetPlaylistTracksToolResult, error) {
	return GetPlaylistTracksToolResult{}, nil
}

type GetRecentlyPlayedToolResult struct {}

func (c *SpotifyClient) GetRecentlyPlayed(ctx tool.Context, _ struct{}) (GetRecentlyPlayedToolResult, error) {
	return GetRecentlyPlayedToolResult{}, nil
}

type RemoveUsersSavedTracksToolResult struct {}

func (c *SpotifyClient) RemoveUsersSavedTracks(ctx tool.Context, _ struct{}) (RemoveUsersSavedTracksToolResult, error) {
	return RemoveUsersSavedTracksToolResult{}, nil
}

///////// PLAY/CREATE TOOLS /////////
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

type CreatePlaylistToolResult struct {}

func (c *SpotifyClient) CreatePlaylist(ctx tool.Context, _ struct{}) (CreatePlaylistToolResult, error) {
	return CreatePlaylistToolResult{}, nil
}

type AddTracksToPlaylistToolResult struct {}

func (c *SpotifyClient) AddTracksToPlaylist(ctx tool.Context, _ struct{}) (AddTracksToPlaylistToolResult, error) {
	return AddTracksToPlaylistToolResult{}, nil
}

type AddToQueueToolResult struct {}

func (c *SpotifyClient) AddToQueue(ctx tool.Context, _ struct{}) (AddToQueueToolResult, error) {
	return AddToQueueToolResult{}, nil
}

///////// ALBUM TOOLS /////////
type GetAlbumsToolResult struct {}

func (c *SpotifyClient) GetAlbums(ctx tool.Context, _ struct{}) (GetAlbumsToolResult, error) {
	return GetAlbumsToolResult{}, nil
}

type GetAlbumTracksToolResult struct {}

func (c *SpotifyClient) GetAlbumTracks(ctx tool.Context, _ struct{}) (GetAlbumTracksToolResult, error) {
	return GetAlbumTracksToolResult{}, nil
}

type SaveOrRemoveAlbumsForUserToolResult struct {}

func (c *SpotifyClient) SaveOrRemoveAlbumsForUser(ctx tool.Context, _ struct{}) (SaveOrRemoveAlbumsForUserToolResult, error) {
	return SaveOrRemoveAlbumsForUserToolResult{}, nil
}

type CheckUsersSavedAlbumsToolResult struct {}

func (c *SpotifyClient) CheckUsersSavedAlbums(ctx tool.Context, _ struct{}) (CheckUsersSavedAlbumsToolResult, error) {
	return CheckUsersSavedAlbumsToolResult{}, nil
}

///////// PLAYLIST TOOLS /////////
type GetPlaylistToolResult struct {}

func (c *SpotifyClient) GetPlaylist(ctx tool.Context, _ struct{}) (GetPlaylistToolResult, error) {
	return GetPlaylistToolResult{}, nil
}

type UpdatePlaylistToolResult struct {}

func (c *SpotifyClient) UpdatePlaylist(ctx tool.Context, _ struct{}) (UpdatePlaylistToolResult, error) {
	return UpdatePlaylistToolResult{}, nil
}

type RemoveTracksFromPlaylistToolResult struct {}

func (c *SpotifyClient) RemoveTracksFromPlaylist(ctx tool.Context, _ struct{}) (RemoveTracksFromPlaylistToolResult, error) {
	return RemoveTracksFromPlaylistToolResult{}, nil
}