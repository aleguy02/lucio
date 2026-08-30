package spotify

import (
	"aleguy02/spotify-tui/internal/toon"
	"context"
	"fmt"
	"strings"

	zmb "github.com/zmb3/spotify/v2"
	"google.golang.org/adk/tool"
)

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
				artists = append(artists, Artist{ID: string(a.ID), Name: a.Name})
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
					ID:          string(a.ID),
					Name:        a.Name,
					Artists:     strings.Join(artistNames, ", "),
					TotalTracks: int(a.TotalTracks),
					ReleaseDate: a.ReleaseDate,
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
				c.setCache(string(t.ID), t.Name)
				tracks = append(tracks, Track{
					ID:          string(t.ID),
					Name:        t.Name,
					Artists:     strings.Join(artistNames, ", "),
					Album:       t.Album.Name,
					DurationMs:  int(t.Duration),
					TrackNumber: int(t.TrackNumber),
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
				c.setCache(string(p.ID), p.Name)
				playlists = append(playlists, Playlist{
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
