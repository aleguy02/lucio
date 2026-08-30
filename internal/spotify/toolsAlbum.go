package spotify

import (
	"aleguy02/spotify-tui/internal/toon"
	"context"
	"strings"

	zmb "github.com/zmb3/spotify/v2"
	"google.golang.org/adk/tool"
)

type GetAlbumsToolArgs struct {
	AlbumIDs []string `json:"album_ids" jsonschema:"Spotify IDs of the albums to look up. Maximum 20"`
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
	AlbumID string `json:"album_id" jsonschema:"Album's Spotify ID"`
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

	out, encErr := toon.Encode(getAlbumTracksJSON{Tracks: tracks, Total: int(page.Total)}, nil)
	if encErr != nil {
		logger.Printf("GetAlbumTracksTool: encode error: %v", encErr)
		return GetAlbumTracksToolResult{Success: false}, encErr
	}
	return GetAlbumTracksToolResult{ToonOutput: out, Success: true}, nil
}
