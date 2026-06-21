package spotify

import (
	"aleguy02/spotify-tui/internal/toon"
	"context"
	"strings"

	"google.golang.org/adk/tool"
)

type GetNowPlayingToolResult struct {
	ToonOutput string `json:"result_as_toon"`
	Success    bool   `json:"success"`
}

type getNowPlayingJSON struct {
	IsPlaying    bool  `json:"is_playing"`
	ProgressMs   int   `json:"progress_ms"`
	ShuffleState bool  `json:"shuffle_state"`
	Track        Track `json:"track"`
}

// GetNowPlayingTool returns the currently playing track and playback state.
func (c *SpotifyClient) GetNowPlayingTool(ctx tool.Context, _ struct{}) (GetNowPlayingToolResult, error) {
	logger.Print("GetNowPlayingTool: was called")

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

type SkipToolResult struct {
	Success bool `json:"success"`
}

func (c *SpotifyClient) SkipNextTool(ctx tool.Context, _ struct{}) (SkipToolResult, error) {
	if err := c.skipForward(); err != nil {
		logger.Printf("SkipNextTool: error (status=%d): %v", spotifyErrStatus(err), err)
		return SkipToolResult{Success: false}, err
	}
	return SkipToolResult{Success: true}, nil
}

func (c *SpotifyClient) SkipPreviousTool(ctx tool.Context, _ struct{}) (SkipToolResult, error) {
	if err := c.skipBack(); err != nil {
		logger.Printf("SkipPreviousTool: error (status=%d): %v", spotifyErrStatus(err), err)
		return SkipToolResult{Success: false}, err
	}
	return SkipToolResult{Success: true}, nil
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
