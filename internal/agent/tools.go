package agent

import (
	sp "aleguy02/spotify-tui/internal/spotify"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// Very simple tool "factory" function
func tools(client *sp.SpotifyClient) ([]tool.Tool, error) {
	skipfTool, err := functiontool.New(
		functiontool.Config{
			Name:        "spotifySkipTrack",
			Description: "Skip the current song/track playing in Spotify",
		}, client.SkipNextTool)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	skipbTool, err := functiontool.New(
		functiontool.Config{
			Name:        "spotifyPreviousTrack",
			Description: "Skip to the previous song/track playing in Spotify",
		}, client.SkipPreviousTool)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	getUserPlaylistsTool, err := functiontool.New(
		functiontool.Config{
			Name:        "spotifyGetUserPlaylists",
			Description: "Get a list of the user's playlists on Spotify",
		}, client.GetUserPlaylistsTool)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	playItemTool, err := functiontool.New(
		functiontool.Config{
			Name:        "spotifyPlayItem",
			Description: "Play an item (track, album, artist, or playlist) on Spotify",
		}, client.PlayItemTool)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	searchSpotifyTool, err := functiontool.New(
		functiontool.Config{
			Name:        "spotifySearch",
			Description: "Search for a track, album, artist, or playlist on Spotify",
		}, client.SearchSpotifyTool)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	addToQueueTool, err := functiontool.New(
		functiontool.Config{
			Name:        "spotifyAddToQueue",
			Description: "Add a track to the queue on Spotify. *Only* supports tracks.",
		}, client.AddToQueueTool)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	getNowPlayingTool, err := functiontool.New(
		functiontool.Config{
			Name:        "spotifyGetNowPlaying",
			Description: "Get the song that is currently playing on Spotify",
		}, client.GetNowPlayingTool)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	getPlaylistTracksTool, err := functiontool.New(
		functiontool.Config{
			Name:        "spotifyGetPlaylistTracks",
			Description: "Get a list of tracks from a playlist owned by the user or where the user is a collaborator on Spotify. Attempting to get tracks of a non-user owned/collaborated playlist will surface FORBIDDEN errors.",
		}, client.GetPlaylistTracksTool)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	getAlbumsTool, err := functiontool.New(
		functiontool.Config{
			Name:        "spotifyGetAlbums",
			Description: "Get the details of one or more albums on Spotify",
		}, client.GetAlbumsTool)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	getAlbumTracksTool, err := functiontool.New(
		functiontool.Config{
			Name:        "spotifyGetAlbumTracks",
			Description: "Get a list of tracks from an album on Spotify",
		}, client.GetAlbumTracksTool)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	getLikedTracksTool, err := functiontool.New(
		functiontool.Config{
			Name:        "spotifyGetLikedTracks",
			Description: "Get a list of the user's liked songs (also called saved tracks) on Spotify",
		}, client.GetLikedTracksTool)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	addTracksToPlaylistTool, err := functiontool.New(
		functiontool.Config{
			Name:                "spotifyAddTracksToPlaylist",
			Description:         "Add one or more tracks (by Spotify ID) to a playlist the user owns or collaborates on. Non-owned playlists surface FORBIDDEN errors.",
			RequireConfirmation: true,
		}, client.AddTracksToPlaylistTool)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	removeTracksFromPlaylistTool, err := functiontool.New(
		functiontool.Config{
			Name:                "spotifyRemoveTracksFromPlaylist",
			Description:         "Remove one or more tracks (by Spotify ID) from a playlist the user owns or collaborates on. Non-owned playlists surface FORBIDDEN errors.",
			RequireConfirmation: true,
		}, client.RemoveTracksFromPlaylistTool)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	removePlaylistFromLibraryTool, err := functiontool.New(
		functiontool.Config{
			Name:                "spotifyRemovePlaylistsFromLibrary",
			Description:         "Removes one or more playlists from the user's library",
			RequireConfirmation: true,
		}, client.RemovePlaylistsFromLibraryTool)
	if err != nil {
		logger.Printf("failed to create function tool: %s", err)
		return nil, fmt.Errorf("failed to create function tool: %w", err)
	}

	tools := []tool.Tool{
		skipfTool,
		skipbTool,
		getUserPlaylistsTool,
		playItemTool,
		searchSpotifyTool,
		addToQueueTool,
		getNowPlayingTool,
		getPlaylistTracksTool,
		getAlbumsTool,
		getAlbumTracksTool,
		getLikedTracksTool,
		addTracksToPlaylistTool,
		removeTracksFromPlaylistTool,
		removePlaylistFromLibraryTool,
	}

	if appConf.TavilyAPIKey != "" {
		tavilyWebSearchTool, err := functiontool.New(
			functiontool.Config{
				Name:        "webSearch",
				Description: "Use natural language to query the web",
			}, TavilyWebSearchTool)
		if err != nil {
			logger.Printf("failed to create function tool: %s", err)
			return nil, fmt.Errorf("failed to create function tool: %w", err)
		}

		tools = append(tools, tavilyWebSearchTool)
	}

	return tools, nil
}

var TAVILY_SEARCH_URL string = "https://api.tavily.com/search"

type TavilyWebSearchToolArgs struct {
	Query string `json:"query" jsonschema:"Natural language search query"`
}

type TavilyWebSearchToolResult struct {
	Result  string `json:"result"`
	Success bool   `json:"success"`
}

func TavilyWebSearchTool(ctx tool.Context, args TavilyWebSearchToolArgs) (TavilyWebSearchToolResult, error) {
	payload := strings.NewReader(fmt.Sprintf("{\n  \"query\": %q,\n  \"search_depth\": \"advanced\",\n  \"chunks_per_source\": 3,\n  \"max_results\": 5,\n  \"topic\": \"general\",\n  \"time_range\": null,\n  \"include_answer\": true,\n  \"include_raw_content\": false,\n  \"include_images\": false,\n  \"include_image_descriptions\": false,\n  \"include_favicon\": false,\n  \"include_domains\": [],\n  \"exclude_domains\": [\"spotify.com\"],\n  \"country\": null,\n  \"auto_parameters\": false,\n  \"exact_match\": false,\n  \"include_usage\": true,\n  \"safe_search\": false\n}", args.Query))

	req, err := http.NewRequest("POST", TAVILY_SEARCH_URL, payload)
	if err != nil {
		return TavilyWebSearchToolResult{Success: false}, err
	}

	authStr := fmt.Sprintf("Bearer %s", appConf.TavilyAPIKey)
	req.Header.Add("Authorization", authStr)
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return TavilyWebSearchToolResult{Success: false}, err
	}

	defer func() { _ = res.Body.Close() }()
	body, _ := io.ReadAll(res.Body)

	var responseMap map[string]interface{}
	if err := json.Unmarshal(body, &responseMap); err != nil {
		return TavilyWebSearchToolResult{Success: false}, err
	}

	queryVal, _ := responseMap["answer"].(string)
	return TavilyWebSearchToolResult{Result: queryVal, Success: true}, nil
}
