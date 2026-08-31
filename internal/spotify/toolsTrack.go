package spotify

import (
	"aleguy02/spotify-tui/internal/toon"
	"context"
	"strings"

	zmb "github.com/zmb3/spotify/v2"
	"google.golang.org/adk/tool"
)

type GetLikedTracksToolArgs struct {
	Limit  int `json:"limit"    jsonschema:"Maximum tracks to return (1-20). Default 20"`
	Offset int `json:"offset"   jsonschema:"Pagination offset. Default 0"`
}

type GetLikedTracksToolResult struct {
	ToonOutput string `json:"result_as_toon"`
	Success    bool   `json:"success"`
}

type getLikedTracksJSON struct {
	Tracks []Track `json:"tracks"`
	Total  int     `json:"total_tracks"`
}

// GetLikedTracksTool returns paginated tracks from the user's liked songs (saved tracks).
func (c *SpotifyClient) GetLikedTracksTool(ctx tool.Context, args GetLikedTracksToolArgs) (GetLikedTracksToolResult, error) {
	if args.Limit <= 0 {
		args.Limit = 20
	}
	if args.Limit > 20 {
		args.Limit = 20
	}
	logger.Printf("GetLikedTracksTool: limit=%d offset=%d", args.Limit, args.Offset)
	page, err := c.client.CurrentUsersTracks(context.Background(), zmb.Limit(args.Limit), zmb.Offset(args.Offset))
	if err != nil {
		logger.Printf("GetLikedTracksTool: API error (status=%d): %v", spotifyErrStatus(err), err)
		return GetLikedTracksToolResult{Success: false}, err
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

	out, encErr := toon.Encode(getLikedTracksJSON{Tracks: tracks, Total: int(page.Total)}, nil)
	if encErr != nil {
		logger.Printf("GetLikedTracksTool: encode error: %v", encErr)
		return GetLikedTracksToolResult{Success: false}, encErr
	}
	return GetLikedTracksToolResult{ToonOutput: out, Success: true}, nil
}
