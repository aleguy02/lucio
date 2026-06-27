package spotify

import (
	"aleguy02/spotify-tui/internal/toon"
	"context"
	"fmt"
	"net/url"
	"strconv"

	zmb "github.com/zmb3/spotify/v2"
	"google.golang.org/adk/tool"
)

type AddTracksToPlaylistToolArgs struct {
	PlaylistID string   `json:"playlist_id" jsonschema:"Spotify ID of the *playlist* to add items to"`
	TrackIDs   []string `json:"track_ids"        jsonschema:"Spotify IDs of the *tracks* to add. Maximum 30"`
}

type AddTracksToPlaylistToolResult struct {
	SnapshotID string `json:"snapshot_id"`
	Success    bool   `json:"success"`
}

// AddTracksToPlaylistTool appends one or more tracks to a playlist and
// returns the resulting snapshot id. Max 30 tracks.
func (c *SpotifyClient) AddTracksToPlaylistTool(ctx tool.Context, args AddTracksToPlaylistToolArgs) (AddTracksToPlaylistToolResult, error) {
	if len(args.TrackIDs) == 0 {
		err := fmt.Errorf("track_ids is required")
		logger.Printf("AddTracksToPlaylistTool: %v", err)
		return AddTracksToPlaylistToolResult{Success: false}, err
	}
	if len(args.TrackIDs) > 30 {
		logger.Printf("AddTracksToPlaylistTool: truncated track_ids to 30: was %d", len(args.TrackIDs))
		args.TrackIDs = args.TrackIDs[:30]
	}

	uris := make([]string, len(args.TrackIDs))
	for i, id := range args.TrackIDs {
		uris[i] = fmt.Sprintf("spotify:track:%s", id)
	}
	logger.Printf("AddTracksToPlaylistTool: playlist_id=%q track_ids=\"%v\"", args.PlaylistID, args.TrackIDs)

	snap, err := c.addPlaylistItems(context.Background(), args.PlaylistID, uris)
	if err != nil {
		logger.Printf("AddTracksToPlaylistTool: API error (playlist_id=%q status=%d): %v", args.PlaylistID, spotifyErrStatus(err), err)
		return AddTracksToPlaylistToolResult{Success: false}, err
	}
	return AddTracksToPlaylistToolResult{SnapshotID: snap, Success: true}, nil
}

type RemoveTracksFromPlaylistToolArgs struct {
	PlaylistID string   `json:"playlist_id" jsonschema:"Spotify ID of the *playlist* to remove items from"`
	TrackIDs   []string `json:"track_ids"        jsonschema:"Spotify IDs of the *tracks* to remove. Maximum 30"`
}

type RemoveTracksFromPlaylistToolResult struct {
	SnapshotID string `json:"snapshot_id"`
	Success    bool   `json:"success"`
}

// RemoveTracksFromPlaylistTool removes one or more tracks from a playlist and
// returns the resulting snapshot id. All occurrences are removed. Max 30 tracks.
func (c *SpotifyClient) RemoveTracksFromPlaylistTool(ctx tool.Context, args RemoveTracksFromPlaylistToolArgs) (RemoveTracksFromPlaylistToolResult, error) {
	if len(args.TrackIDs) == 0 {
		err := fmt.Errorf("uris is required")
		logger.Printf("RemoveTracksFromPlaylistTool: %v", err)
		return RemoveTracksFromPlaylistToolResult{Success: false}, err
	}
	if len(args.TrackIDs) > 30 {
		logger.Printf("RemoveTracksFromPlaylistTool: truncated track_ids to 30: was %d", len(args.TrackIDs))
		args.TrackIDs = args.TrackIDs[:30]
	}

	uris := make([]string, len(args.TrackIDs))
	for i, id := range args.TrackIDs {
		uris[i] = fmt.Sprintf("spotify:track:%s", id)
	}
	logger.Printf("RemoveTracksFromPlaylistTool: playlist_id=%q track_ids=\"%v\"", args.PlaylistID, args.TrackIDs)

	snap, err := c.removePlaylistItems(context.Background(), args.PlaylistID, uris)
	if err != nil {
		logger.Printf("RemoveTracksFromPlaylistTool: API error (playlist_id=%q status=%d): %v", args.PlaylistID, spotifyErrStatus(err), err)
		return RemoveTracksFromPlaylistToolResult{Success: false}, err
	}
	return RemoveTracksFromPlaylistToolResult{SnapshotID: snap, Success: true}, nil
}

type RemovePlaylistsFromLibraryToolArgs struct {
	PlaylistIDs []string `json:"playlist_ids" jsonschema:"Spotify IDs of the playlists to remove from the user's library. Maximum 10"`
}

type RemovePlaylistsFromLibraryToolResult struct {
	Success bool `json:"success"`
}

// RemovePlaylistsFromLibraryTool removes one or more playlists from the user's library by Spotify playlist ID.
func (c *SpotifyClient) RemovePlaylistsFromLibraryTool(ctx tool.Context, args RemovePlaylistsFromLibraryToolArgs) (RemovePlaylistsFromLibraryToolResult, error) {
	if len(args.PlaylistIDs) == 0 {
		err := fmt.Errorf("playlist_ids is required")
		logger.Printf("RemovePlaylistsFromLibraryTool: %v", err)
		return RemovePlaylistsFromLibraryToolResult{Success: false}, err
	}
	if len(args.PlaylistIDs) > 10 {
		logger.Printf("RemoveTracksFromPlaylistTool: truncated playlist_ids to 10: was %d", len(args.PlaylistIDs))
		args.PlaylistIDs = args.PlaylistIDs[:10]
	}
	logger.Printf("RemovePlaylistsFromLibraryTool: ids=%v", args.PlaylistIDs)

	uris := make([]string, len(args.PlaylistIDs))
	for i, id := range args.PlaylistIDs {
		uris[i] = fmt.Sprintf("spotify:playlist:%s", id)
	}

	if err := c.removePlaylistsFromLibrary(context.Background(), uris); err != nil {
		logger.Printf("RemovePlaylistsFromLibraryTool: API error (status=%d): %v", spotifyErrStatus(err), err)
		return RemovePlaylistsFromLibraryToolResult{Success: false}, err
	}
	return RemovePlaylistsFromLibraryToolResult{Success: true}, nil
}

type GetUserPlaylistsToolArgs struct {
	Limit  int `json:"limit" jsonschema:"Maximum number of playlists to return. Maximum 10."`
	Offset int `json:"offset" jsonschema:"Offset index of the first playlist to return."`
}

type GetUserPlaylistsToolResult struct {
	ToonOutput string `json:"result_as_toon"`
	Success    bool   `json:"success"`
}

type GetUserPlaylistsToolResultJSON struct {
	Playlists []Playlist `json:"playlists"`
	Total     int        `json:"num_playlists_available"`
}

// Get a list of the current user's playlists on Spotify. Returns an array of playlists packed into SpotifyItems.
func (c *SpotifyClient) GetUserPlaylistsTool(ctx tool.Context, args GetUserPlaylistsToolArgs) (GetUserPlaylistsToolResult, error) {
	if args.Limit > 10 {
		logger.Printf("GetUserPlaylistsTool: clamped limit to 10: was %d", args.Limit)
		args.Limit = 10
	}

	logger.Printf("GetUserPlaylistsTool: limit=%d offset=%d", args.Limit, args.Offset)
	page, err := c.client.CurrentUsersPlaylists(context.Background(), zmb.Limit(args.Limit), zmb.Offset(args.Offset))
	if err != nil {
		logger.Printf("GetUserPlaylistsTool: API error (status=%d): %v", spotifyErrStatus(err), err)
		return GetUserPlaylistsToolResult{Success: false}, err
	}

	var playlists []Playlist
	for _, p := range page.Playlists {
		playlists = append(playlists, Playlist{
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
	PlaylistID string `json:"playlist_id" jsonschema:"Playlist's Spotify ID"`
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
		logger.Printf("GetPlaylistTracksTool: clamped limit to 20: was %d", args.Limit)
		args.Limit = 20
	}
	logger.Printf("GetPlaylistTracksTool: playlist_id=%q limit=%d offset=%d", args.PlaylistID, args.Limit, args.Offset)
	tracks, total, err := c.fetchPlaylistItems(context.Background(), args.PlaylistID, url.Values{
		"limit":  []string{strconv.Itoa(args.Limit)},
		"offset": []string{strconv.Itoa(args.Offset)},
	})
	if err != nil {
		logger.Printf("GetPlaylistTracksTool: API error (playlist_id=%q status=%d): %v", args.PlaylistID, spotifyErrStatus(err), err)
		return GetPlaylistTracksToolResult{Success: false}, err
	}

	out, encErr := toon.Encode(getPlaylistTracksJSON{Tracks: tracks, Total: total}, nil)
	if encErr != nil {
		logger.Printf("GetPlaylistTracksTool: encode error: %v", encErr)
		return GetPlaylistTracksToolResult{Success: false}, encErr
	}
	return GetPlaylistTracksToolResult{ToonOutput: out, Success: true}, nil
}
