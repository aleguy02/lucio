package spotify

import (
	"aleguy02/spotify-tui/internal/toon"
	"context"
	"errors"
	"fmt"
	"strings"

	zmb "github.com/zmb3/spotify/v2"
	"google.golang.org/adk/tool"
)

// spotifyErrStatus extracts the HTTP status code from a zmb.Error, returning 0 if unavailable.
func spotifyErrStatus(err error) int {
	var sErr zmb.Error
	if errors.As(err, &sErr) {
		return sErr.Status
	}
	return 0
}

// TOON Types

// Artist is a simplified representation of a Spotify artist.
type Artist struct {
	URI  string `json:"uri"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Album is a simplified representation of a Spotify album.
type Album struct {
	URI                  string `json:"uri"`
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Artists string `json:"artists"`
	TotalTracks          int    `json:"total_tracks"`
	ReleaseDate          string `json:"release_date"`
}

// Track is a simplified representation of a Spotify track.
type Track struct {
	URI                  string `json:"uri"`
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Artists string `json:"artists"`
	Album                string `json:"album"`
	DurationMs           int    `json:"duration_ms"`
	TrackNumber          int    `json:"track_number"`
}

// Playlist is a simplified representation of a Spotify playlist.
type Playlist struct {
	URI           string `json:"uri"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	Owner         string `json:"owner"`
	Collaborative bool   `json:"collaborative"`
	Description   string `json:"description"`
}

// credit to https://github.com/marcelmarais/spotify-mcp-server
///////// READ TOOLS /////////

type SearchSpotifyToolArgs struct {
	SearchType int    `json:"spotify_type" jsonschema:"The Spotify item type as an integer. album=1 artist=2 playlist=4 track=8"`
	Query      string `json:"query"        jsonschema:"Search query"`
}

type SearchSpotifyToolResult struct {
	ToonOutput string `json:"result_as_toon"`
	Success    bool   `json:"success"`
}

type searchArtistsJSON struct {
	Artists []Artist `json:"artists"`
}

type searchAlbumsJSON struct {
	Albums []Album `json:"albums"`
}

type searchTracksJSON struct {
	Tracks []Track `json:"tracks"`
}

type searchPlaylistsJSON struct {
	Playlists []Playlist `json:"playlists"`
}

// SearchSpotify searches for tracks, albums, artists, or playlists on Spotify
// based on the provided query and item type.
func (c *SpotifyClient) SearchSpotifyTool(ctx tool.Context, args SearchSpotifyToolArgs) (SearchSpotifyToolResult, error) {
	const limit = 7
	const retries = 3

	var searchType zmb.SearchType
	switch args.SearchType {
	case 1:
		searchType = zmb.SearchTypeAlbum
	case 2:
		searchType = zmb.SearchTypeArtist
	case 4:
		searchType = zmb.SearchTypePlaylist
	case 8:
		searchType = zmb.SearchTypeTrack
	default:
		err := fmt.Errorf("unknown search type %d", args.SearchType)
		logger.Printf("SearchSpotifyTool: %v", err)
		return SearchSpotifyToolResult{Success: false}, err
	}

	logger.Printf("SearchSpotifyTool: type=%d query=%q", args.SearchType, args.Query)

	var out string
	var encErr error

	switch searchType {
	case zmb.SearchTypeArtist:
		var artists []Artist
		for offset := 0; len(artists) < limit && offset < retries; offset++ {
			res, err := c.client.Search(context.Background(), args.Query, searchType, zmb.Limit(limit), zmb.Offset(offset*limit))
			if err != nil {
				logger.Printf("SearchSpotifyTool: artist search error (query=%q offset=%d status=%d): %v", args.Query, offset, spotifyErrStatus(err), err)
				return SearchSpotifyToolResult{Success: false}, err
			}
			if res.Artists == nil {
				continue
			}
			for _, a := range res.Artists.Artists {
				if string(a.ID) == "" {
					continue
				}
				artists = append(artists, Artist{URI: string(a.URI), ID: string(a.ID), Name: a.Name})
			}
		}
		out, encErr = toon.Encode(searchArtistsJSON{Artists: artists}, nil)

	case zmb.SearchTypeAlbum:
		var albums []Album
		for offset := 0; len(albums) < limit && offset < retries; offset++ {
			res, err := c.client.Search(context.Background(), args.Query, searchType, zmb.Limit(limit), zmb.Offset(offset*limit))
			if err != nil {
				logger.Printf("SearchSpotifyTool: album search error (query=%q offset=%d status=%d): %v", args.Query, offset, spotifyErrStatus(err), err)
				return SearchSpotifyToolResult{Success: false}, err
			}
			if res.Albums == nil {
				continue
			}
			for _, a := range res.Albums.Albums {
				if string(a.ID) == "" {
					continue
				}
				var artistNames []string
				for _, artist := range a.Artists {
					artistNames = append(artistNames, artist.Name)
				}
				albums = append(albums, Album{
					URI:                  string(a.URI),
					ID:                   string(a.ID),
					Name:                 a.Name,
					Artists: strings.Join(artistNames, ", "),
					TotalTracks:          int(a.TotalTracks),
					ReleaseDate:          a.ReleaseDate,
				})
			}
		}
		out, encErr = toon.Encode(searchAlbumsJSON{Albums: albums}, nil)

	case zmb.SearchTypeTrack:
		var tracks []Track
		for offset := 0; len(tracks) < limit && offset < retries; offset++ {
			res, err := c.client.Search(context.Background(), args.Query, searchType, zmb.Limit(limit), zmb.Offset(offset*limit))
			if err != nil {
				logger.Printf("SearchSpotifyTool: track search error (query=%q offset=%d status=%d): %v", args.Query, offset, spotifyErrStatus(err), err)
				return SearchSpotifyToolResult{Success: false}, err
			}
			if res.Tracks == nil {
				continue
			}
			for _, t := range res.Tracks.Tracks {
				if string(t.ID) == "" {
					continue
				}
				var artistNames []string
				for _, artist := range t.Artists {
					artistNames = append(artistNames, artist.Name)
				}
				tracks = append(tracks, Track{
					URI:                  string(t.URI),
					ID:                   string(t.ID),
					Name:                 t.Name,
					Artists: strings.Join(artistNames, ", "),
					Album:                t.Album.Name,
					DurationMs:           int(t.Duration),
					TrackNumber:          int(t.TrackNumber),
				})
			}
		}
		out, encErr = toon.Encode(searchTracksJSON{Tracks: tracks}, nil)

	case zmb.SearchTypePlaylist:
		var playlists []Playlist
		for offset := 0; len(playlists) < limit && offset < retries; offset++ {
			res, err := c.client.Search(context.Background(), args.Query, searchType, zmb.Limit(limit), zmb.Offset(offset*limit))
			if err != nil {
				logger.Printf("SearchSpotifyTool: playlist search error (query=%q offset=%d status=%d): %v", args.Query, offset, spotifyErrStatus(err), err)
				return SearchSpotifyToolResult{Success: false}, err
			}
			if res.Playlists == nil {
				continue
			}
			for _, p := range res.Playlists.Playlists {
				if string(p.ID) == "" {
					continue
				}
				playlists = append(playlists, Playlist{
					URI:           string(p.URI),
					ID:            string(p.ID),
					Name:          p.Name,
					Owner:         p.Owner.DisplayName,
					Collaborative: p.Collaborative,
					Description:   p.Description,
				})
			}
		}
		out, encErr = toon.Encode(searchPlaylistsJSON{Playlists: playlists}, nil)
	}

	if encErr != nil {
		logger.Printf("SearchSpotifyTool: encode error: %v", encErr)
		return SearchSpotifyToolResult{Success: false}, encErr
	}
	return SearchSpotifyToolResult{ToonOutput: out, Success: true}, nil
}

type getNowPlayingJSON struct {
	IsPlaying    bool  `json:"is_playing"`
	ProgressMs   int   `json:"progress_ms"`
	ShuffleState bool  `json:"shuffle_state"`
	Track        Track `json:"track"`
}

type GetNowPlayingToolResult struct {
	ToonOutput string `json:"result_as_toon"`
	Success    bool   `json:"success"`
}

// GetNowPlayingTool returns the currently playing track and playback state.
func (c *SpotifyClient) GetNowPlayingTool(ctx tool.Context, _ struct{}) (GetNowPlayingToolResult, error) {
	result, err := c.client.PlayerState(context.Background())
	if err != nil {
		logger.Printf("GetNowPlayingTool: PlayerState error (status=%d): %v", spotifyErrStatus(err), err)
		return GetNowPlayingToolResult{Success: false}, err
	}

	j := getNowPlayingJSON{
		IsPlaying:    result.Playing,
		ProgressMs:   int(result.Progress),
		ShuffleState: result.ShuffleState,
	}

	if result.Item != nil {
		t := result.Item
		var artistNames []string
		for _, a := range t.Artists {
			artistNames = append(artistNames, a.Name)
		}
		j.Track = Track{
			URI:         string(t.URI),
			ID:          string(t.ID),
			Name:        t.Name,
			Artists:     strings.Join(artistNames, ", "),
			Album:       t.Album.Name,
			DurationMs:  int(t.Duration),
			TrackNumber: int(t.TrackNumber),
		}
	}

	out, encErr := toon.Encode(j, nil)
	if encErr != nil {
		logger.Printf("GetNowPlayingTool: encode error: %v", encErr)
		return GetNowPlayingToolResult{Success: false}, encErr
	}
	return GetNowPlayingToolResult{ToonOutput: out, Success: true}, nil
}

type GetUserPlaylistsToolArgs struct {
	Limit  int `json:"limit" jsonschema:"Maximum number of playlists to return. Default value is 10 for single tool calls"`
	Offset int `json:"offset" jsonschema:"Index of the first playlist to return."`
}

type GetUserPlaylistsToolResult struct {
	ToonOutput string `json:"result_as_toon"`
	Success    bool   `json:"success"`
}

type GetUserPlaylistsToolResultJSON struct {
	Playlists []Playlist `json:"playlists"`
	Total     int        `json:"total_user_playlists"`
}

// Get a list of the current user's playlists on Spotify.
//
// It returns an array of playlists packed into SpotifyItems.
func (c *SpotifyClient) GetUserPlaylistsTool(ctx tool.Context, args GetUserPlaylistsToolArgs) (GetUserPlaylistsToolResult, error) {
	logger.Printf("GetUserPlaylistsTool: limit=%d offset=%d", args.Limit, args.Offset)
	page, err := c.client.CurrentUsersPlaylists(context.Background(), zmb.Limit(args.Limit), zmb.Offset(args.Offset))
	if err != nil {
		logger.Printf("GetUserPlaylistsTool: API error (status=%d): %v", spotifyErrStatus(err), err)
		return GetUserPlaylistsToolResult{Success: false}, err
	}

	var playlists []Playlist
	for _, p := range page.Playlists {
		playlists = append(playlists, Playlist{
			URI:           string(p.URI),
			ID:            string(p.ID),
			Name:          p.Name,
			Owner:         p.Owner.DisplayName,
			Collaborative: p.Collaborative,
			Description:   p.Description,
		})
	}
	j := GetUserPlaylistsToolResultJSON{
		Playlists: playlists,
		Total:     int(page.Total),
	}
	out, err := toon.Encode(j, nil)
	if err != nil {
		logger.Printf("GetUserPlaylistsTool: encode error: %v", err)
		return GetUserPlaylistsToolResult{Success: false}, err
	}
	return GetUserPlaylistsToolResult{ToonOutput: out, Success: true}, nil
}

type GetPlaylistTracksToolArgs struct {
	PlaylistID string `json:"playlist_id" jsonschema:"The Spotify ID of the playlist"`
	Limit      int    `json:"limit"       jsonschema:"Maximum tracks to return (1-20). Default 20"`
	Offset     int    `json:"offset"      jsonschema:"Pagination offset. Default 0"`
}

type GetPlaylistTracksToolResult struct {
	ToonOutput string `json:"result_as_toon"`
	Success    bool   `json:"success"`
}

type getPlaylistTracksJSON struct {
	Tracks []Track `json:"tracks"`
	Total  int     `json:"total_tracks"`
}

// GetPlaylistTracksTool returns paginated tracks for a playlist. Max limit 20.
func (c *SpotifyClient) GetPlaylistTracksTool(ctx tool.Context, args GetPlaylistTracksToolArgs) (GetPlaylistTracksToolResult, error) {
	if args.Limit > 20 {
		args.Limit = 20
	}
	logger.Printf("GetPlaylistTracksTool: playlist=%q limit=%d offset=%d", args.PlaylistID, args.Limit, args.Offset)
	page, err := c.client.GetPlaylistItems(context.Background(), zmb.ID(args.PlaylistID), zmb.Limit(args.Limit), zmb.Offset(args.Offset))
	if err != nil {
		logger.Printf("GetPlaylistTracksTool: API error (playlist=%q status=%d): %v", args.PlaylistID, spotifyErrStatus(err), err)
		return GetPlaylistTracksToolResult{Success: false}, err
	}

	var tracks []Track
	for _, item := range page.Items {
		t := item.Track.Track
		if t == nil || string(t.ID) == "" {
			continue
		}
		var artistNames []string
		for _, a := range t.Artists {
			artistNames = append(artistNames, a.Name)
		}
		tracks = append(tracks, Track{
			URI:         string(t.URI),
			ID:          string(t.ID),
			Name:        t.Name,
			Artists:     strings.Join(artistNames, ", "),
			Album:       t.Album.Name,
			DurationMs:  int(t.Duration),
			TrackNumber: int(t.TrackNumber),
		})
	}

	out, encErr := toon.Encode(getPlaylistTracksJSON{Tracks: tracks, Total: int(page.Total)}, nil)
	if encErr != nil {
		logger.Printf("GetPlaylistTracksTool: encode error: %v", encErr)
		return GetPlaylistTracksToolResult{Success: false}, encErr
	}
	return GetPlaylistTracksToolResult{ToonOutput: out, Success: true}, nil
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
	logger.Printf("PlayItemTool: %+v", args)
	if err := c.ExecutePlayback(args); err != nil {
		logger.Printf("PlayItemTool: error (status=%d): %v", spotifyErrStatus(err), err)
		return PlayItemToolResult{Success: false}, err
	}
	return PlayItemToolResult{Success: true}, nil
}

type SkipNextToolResult struct {
	Success bool `json:"success"`
}

func (c *SpotifyClient) SkipNextTool(ctx tool.Context, _ struct{}) (SkipNextToolResult, error) {
	if err := c.skipForward(); err != nil {
		logger.Printf("SkipNextTool: error (status=%d): %v", spotifyErrStatus(err), err)
		return SkipNextToolResult{Success: false}, err
	}
	return SkipNextToolResult{Success: true}, nil
}

type SkipPreviousToolResult struct {
	Success bool `json:"success"`
}

func (c *SpotifyClient) SkipPreviousTool(ctx tool.Context, _ struct{}) (SkipPreviousToolResult, error) {
	if err := c.skipBack(); err != nil {
		logger.Printf("SkipPreviousTool: error (status=%d): %v", spotifyErrStatus(err), err)
		return SkipPreviousToolResult{Success: false}, err
	}
	return SkipPreviousToolResult{Success: true}, nil
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

func (c *SpotifyClient) AddToQueueTool(ctx tool.Context, args AddToQueueToolArgs) (AddToQueueToolResult, error) {
	logger.Printf("AddToQueue: %+v", args)
	err := c.QueueSong(args)
	if err != nil {
		logger.Printf("AddToQueueTool: error (status=%d): %v", spotifyErrStatus(err), err)
		return AddToQueueToolResult{Success: false}, err
	}
	return AddToQueueToolResult{Success: true}, nil
}

// /////// ALBUM TOOLS /////////
type GetAlbumsToolArgs struct {
	AlbumIDs []string `json:"album_ids" jsonschema:"Spotify album IDs to look up. Maximum 20"`
}

type GetAlbumsToolResult struct {
	ToonOutput string `json:"result_as_toon"`
	Success    bool   `json:"success"`
}

type getAlbumsJSON struct {
	Albums []Album `json:"albums"`
}

// GetAlbumsTool returns details for one or more albums by Spotify ID. Max 20.
func (c *SpotifyClient) GetAlbumsTool(ctx tool.Context, args GetAlbumsToolArgs) (GetAlbumsToolResult, error) {
	if len(args.AlbumIDs) > 20 {
		args.AlbumIDs = args.AlbumIDs[:20]
	}
	logger.Printf("GetAlbumsTool: ids=%v", args.AlbumIDs)
	ids := make([]zmb.ID, len(args.AlbumIDs))
	for i, id := range args.AlbumIDs {
		ids[i] = zmb.ID(id)
	}

	results, err := c.client.GetAlbums(context.Background(), ids)
	if err != nil {
		logger.Printf("GetAlbumsTool: API error (status=%d): %v", spotifyErrStatus(err), err)
		return GetAlbumsToolResult{Success: false}, err
	}

	var albums []Album
	for _, a := range results {
		if a == nil || string(a.ID) == "" {
			continue
		}
		var artistNames []string
		for _, artist := range a.Artists {
			artistNames = append(artistNames, artist.Name)
		}
		albums = append(albums, Album{
			URI:         string(a.URI),
			ID:          string(a.ID),
			Name:        a.Name,
			Artists:     strings.Join(artistNames, ", "),
			TotalTracks: int(a.TotalTracks),
			ReleaseDate: a.ReleaseDate,
		})
	}

	out, encErr := toon.Encode(getAlbumsJSON{Albums: albums}, nil)
	if encErr != nil {
		logger.Printf("GetAlbumsTool: encode error: %v", encErr)
		return GetAlbumsToolResult{Success: false}, encErr
	}
	return GetAlbumsToolResult{ToonOutput: out, Success: true}, nil
}

type GetAlbumTracksToolArgs struct {
	AlbumID string `json:"album_id" jsonschema:"The Spotify ID of the album"`
	Limit   int    `json:"limit"    jsonschema:"Maximum tracks to return (1-20). Default 20"`
	Offset  int    `json:"offset"   jsonschema:"Pagination offset. Default 0"`
}

type GetAlbumTracksToolResult struct {
	ToonOutput string `json:"result_as_toon"`
	Success    bool   `json:"success"`
}

type getAlbumTracksJSON struct {
	Tracks []Track `json:"tracks"`
	Total  int     `json:"total_tracks"`
}

// GetAlbumTracksTool returns paginated tracks for an album.
func (c *SpotifyClient) GetAlbumTracksTool(ctx tool.Context, args GetAlbumTracksToolArgs) (GetAlbumTracksToolResult, error) {
	if args.Limit > 20 {
		args.Limit = 20
	}
	logger.Printf("GetAlbumTracksTool: album=%q limit=%d offset=%d", args.AlbumID, args.Limit, args.Offset)
	page, err := c.client.GetAlbumTracks(context.Background(), zmb.ID(args.AlbumID), zmb.Limit(args.Limit), zmb.Offset(args.Offset))
	if err != nil {
		logger.Printf("GetAlbumTracksTool: API error (album=%q status=%d): %v", args.AlbumID, spotifyErrStatus(err), err)
		return GetAlbumTracksToolResult{Success: false}, err
	}

	var tracks []Track
	for _, t := range page.Tracks {
		if string(t.ID) == "" {
			continue
		}
		var artistNames []string
		for _, a := range t.Artists {
			artistNames = append(artistNames, a.Name)
		}
		tracks = append(tracks, Track{
			URI:         string(t.URI),
			ID:          string(t.ID),
			Name:        t.Name,
			Artists:     strings.Join(artistNames, ", "),
			Album:       t.Album.Name,
			DurationMs:  int(t.Duration),
			TrackNumber: int(t.TrackNumber),
		})
	}

	out, encErr := toon.Encode(getAlbumTracksJSON{Tracks: tracks, Total: int(page.Total)}, nil)
	if encErr != nil {
		logger.Printf("GetAlbumTracksTool: encode error: %v", encErr)
		return GetAlbumTracksToolResult{Success: false}, encErr
	}
	return GetAlbumTracksToolResult{ToonOutput: out, Success: true}, nil
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
