package spotify

import (
	"context"
	"maps"

	zmb "github.com/zmb3/spotify/v2"
)

// GetPlaylistName resolves a playlist ID to its display name, for use in tool-confirmation
// prompts. Returns a cached name when available; otherwise fetches and caches it.
func (c *SpotifyClient) GetPlaylistName(id string) (string, error) {
	if name, ok := c.getCached(id); ok {
		return name, nil
	}
	pl, err := c.client.GetPlaylist(context.Background(), zmb.ID(id))
	if err != nil {
		logger.Printf("GetPlaylistName %q: %v", id, err)
		return "", err
	}
	c.setCache(id, pl.Name)
	return pl.Name, nil
}

// GetTrackNames resolves multiple track IDs to their display names, preserving order, for
// use in tool-confirmation prompts. Cached names are reused; only the remaining (uncached)
// IDs are fetched, in a single batched call.
func (c *SpotifyClient) GetTrackNames(ids []string) ([]string, error) {
	names := make([]string, len(ids))
	var missingIdx []int
	var missingIDs []zmb.ID
	for i, id := range ids {
		if name, ok := c.getCached(id); ok {
			names[i] = name
			continue
		}
		missingIdx = append(missingIdx, i)
		missingIDs = append(missingIDs, zmb.ID(id))
	}
	if len(missingIDs) == 0 {
		return names, nil
	}

	tracks, err := c.client.GetTracks(context.Background(), missingIDs)
	if err != nil {
		logger.Printf("GetTrackNames %v: %v", missingIDs, err)
		return nil, err
	}
	for j, t := range tracks {
		if t == nil {
			continue
		}
		idx := missingIdx[j]
		names[idx] = t.Name
		c.setCache(string(t.ID), t.Name)
	}
	return names, nil
}

// toStringSlice coerces a tool argument value into a []string. Args arrive as
// map[string]any decoded from JSON, so a []string-typed argument may surface as []any of
// strings; both shapes are accepted. Returns nil if v isn't a recognizable string slice.
func toStringSlice(v any) []string {
	switch vv := v.(type) {
	case []string:
		return vv
	case []any:
		out := make([]string, 0, len(vv))
		for _, e := range vv {
			s, ok := e.(string)
			if !ok {
				return nil
			}
			out = append(out, s)
		}
		return out
	default:
		return nil
	}
}

// ResolveConfirmationArgs returns a display-friendly copy of a tool's arguments for the
// human confirmation prompt, substituting playlist/track IDs with their Spotify names where
// this repo knows how to resolve them. Unknown tools, or IDs that fail to resolve, are left
// unchanged so the prompt still shows something actionable.
func (c *SpotifyClient) ResolveConfirmationArgs(toolName string, args map[string]any) map[string]any {
	display := make(map[string]any, len(args))
	maps.Copy(display, args)

	switch toolName {
	case "spotifyAddTracksToPlaylist", "spotifyRemoveTracksFromPlaylist":
		if id, ok := args["playlist_id"].(string); ok {
			if name, err := c.GetPlaylistName(id); err == nil {
				delete(display, "playlist_id")
				display["playlist"] = name
			}
		}
		if ids := toStringSlice(args["track_ids"]); len(ids) > 0 {
			if names, err := c.GetTrackNames(ids); err == nil {
				delete(display, "track_ids")
				display["tracks"] = names
			}
		}
	case "spotifyRemovePlaylistsFromLibrary":
		if ids := toStringSlice(args["playlist_ids"]); len(ids) > 0 {
			names := make([]string, len(ids))
			for i, id := range ids {
				if name, err := c.GetPlaylistName(id); err == nil {
					names[i] = name
				} else {
					names[i] = id
				}
			}
			delete(display, "playlist_ids")
			display["playlists"] = names
		}
	}

	return display
}
