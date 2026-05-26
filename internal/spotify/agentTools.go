package spotify

import (
	"aleguy02/spotify-tui/internal/toon"
	"context"
	"fmt"
	"strings"

	zmb "github.com/zmb3/spotify/v2"
	"google.golang.org/adk/tool"
)

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
		return SearchSpotifyToolResult{Success: false}, fmt.Errorf("unknown search type %d", args.SearchType)
	}

	var out string
	var encErr error

	switch searchType {
	case zmb.SearchTypeArtist:
		var artists []Artist
		for offset := 0; len(artists) < limit && offset < retries; offset++ {
			res, err := c.client.Search(context.Background(), args.Query, searchType, zmb.Limit(limit), zmb.Offset(offset*limit))
			if err != nil {
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
		return SearchSpotifyToolResult{Success: false}, encErr
	}
	return SearchSpotifyToolResult{ToonOutput: out, Success: true}, nil
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
	page, err := c.client.CurrentUsersPlaylists(context.Background(), zmb.Limit(args.Limit), zmb.Offset(args.Offset))
	if err != nil {
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
		return GetUserPlaylistsToolResult{Success: false}, err
	}
	return GetUserPlaylistsToolResult{ToonOutput: out, Success: true}, nil
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
