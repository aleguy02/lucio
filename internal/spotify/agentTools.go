package spotify

import "google.golang.org/adk/tool"

// credit to https://github.com/marcelmarais/spotify-mcp-server
///////// READ TOOLS /////////
type SearchSpotifyToolResult struct {}

func (c *SpotifyClient) SearchSpotify(ctx tool.Context, _ struct{}) (SearchSpotifyToolResult, error) {
	return SearchSpotifyToolResult{}, nil
}

type GetNowPlayingToolResult struct {}

func (c *SpotifyClient) GetNowPlaying(ctx tool.Context, _ struct{}) (GetNowPlayingToolResult, error) {
	return GetNowPlayingToolResult{}, nil
}

type GetMyPlaylistsToolResult struct {}

func (c *SpotifyClient) GetMyPlaylists(ctx tool.Context, _ struct{}) (GetMyPlaylistsToolResult, error) {
	return GetMyPlaylistsToolResult{}, nil
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
type SkipfWrapperResult struct {
	Success bool `json:"success"`
}

func (c *SpotifyClient) SkipfWrapper(ctx tool.Context, _ struct{}) (SkipfWrapperResult, error) {
	if err := c.skipForward(); err != nil {
		return SkipfWrapperResult{
			Success: false,
		}, err
	}

	return SkipfWrapperResult{
		Success: true,
	}, nil
}

type SkipbWrapperResult struct {
	Success bool `json:"success"`
}

func (c *SpotifyClient) SkipbWrapper(ctx tool.Context, _ struct{}) (SkipbWrapperResult, error) {
	if err := c.skipBack(); err != nil {
		return SkipbWrapperResult{
			Success: false,
		}, err
	}

	return SkipbWrapperResult{
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