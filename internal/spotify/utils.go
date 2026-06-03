package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	zmb "github.com/zmb3/spotify/v2"
)

func parseSeconds(arg string) (int, error) {
	if strings.TrimSpace(arg) == "" {
		return 0, fmt.Errorf("seconds argument is required")
	}
	s, err := strconv.Atoi(strings.TrimSpace(arg))
	if err != nil {
		return 0, fmt.Errorf("seconds argument must be a whole integer, got %q", arg)
	}
	if s <= 0 {
		return 0, fmt.Errorf("seconds argument must be positive, got %d", s)
	}
	return s, nil
}

func (c *SpotifyClient) play() error {
	err := c.client.Play(context.Background())
	if err != nil {
		logger.Printf("play: %v", err)
	}
	return err
}

func (c *SpotifyClient) pause() error {
	err := c.client.Pause(context.Background())
	if err != nil {
		logger.Printf("pause: %v", err)
	}
	return err
}

func (c *SpotifyClient) skipForward() error {
	err := c.client.Next(context.Background())
	if err != nil {
		logger.Printf("skipForward: %v", err)
	}
	return err
}

func (c *SpotifyClient) skipBack() error {
	err := c.client.Previous(context.Background())
	if err != nil {
		logger.Printf("skipBack: %v", err)
	}
	return err
}

func (c *SpotifyClient) seekForward(s int) error {
	state, err := c.client.PlayerState(context.Background())
	if err != nil {
		return fmt.Errorf("could not get player state: %w", err)
	}
	newPos := int(state.Progress) + s*1000
	if state.Item != nil && newPos > int(state.Item.Duration) {
		newPos = int(state.Item.Duration)
	}
	err = c.client.Seek(context.Background(), newPos)
	if err != nil {
		logger.Printf("seekForward: %v", err)
	}
	return err
}

func (c *SpotifyClient) toggleShuffle(curShuffleState bool) error {
	err := c.client.Shuffle(context.Background(), curShuffleState)
	if err != nil {
		logger.Printf("toggleShuffle: %v", err)
	}
	return err
}

func (c *SpotifyClient) seekBack(s int) error {
	state, err := c.client.PlayerState(context.Background())
	if err != nil {
		return fmt.Errorf("could not get player state: %w", err)
	}
	newPos := int(state.Progress) - s*1000
	if newPos < 0 {
		newPos = 0
	}
	err = c.client.Seek(context.Background(), newPos)
	if err != nil {
		logger.Printf("seekBack: %v", err)
	}
	return err
}

func (c *SpotifyClient) search(query string, t zmb.SearchType, offset int) (*zmb.SearchResult, error) {
	result, err := c.client.Search(context.Background(), query, t, zmb.Limit(7), zmb.Offset(offset))
	if err != nil {
		return nil, fmt.Errorf("could not perform search: %w", err)
	}
	return result, nil
}

func (c *SpotifyClient) getDevices() ([]zmb.PlayerDevice, error) {
	devices, err := c.client.PlayerDevices(context.Background())
	if err != nil {
		return nil, fmt.Errorf("could not get devices: %w", err)
	}
	return devices, nil
}

func (c *SpotifyClient) selectDevice(id string) error {
	if err := c.client.TransferPlayback(context.Background(), zmb.ID(id), true); err != nil {
		return fmt.Errorf("could not transfer playback to device %q: %w", id, err)
	}
	return nil
}

// playTrack queues then skips to the song — PlayOpt alone stops after one track.
func (c *SpotifyClient) playTrack(id zmb.ID) error {
	if err := c.client.QueueSong(context.Background(), id); err != nil {
		return fmt.Errorf("could not queue song: %w", err)
	}
	if err := c.client.Next(context.Background()); err != nil {
		return fmt.Errorf("could not play song: %w", err)
	}
	return nil
}

func (c *SpotifyClient) playFromContext(uri zmb.URI) error {
	err := c.client.PlayOpt(context.Background(), &zmb.PlayOptions{
		PlaybackContext: &uri,
	})
	if err != nil {
		logger.Printf("playFromContext: %v", err)
	}
	return err
}

/* IMPORTANT
 * reimplementation of zmb's GetPlaylistItems since that function is hitting a deprecated endpoint and uses some deprecated return values
 */
var baseURL string = "https://api.spotify.com/v1/"

type PlaylistItemPage struct {
	basePage
	Items []PlaylistItem `json:"items"`
}
type basePage struct {
	Endpoint string      `json:"href"`
	Limit    zmb.Numeric `json:"limit"`
	Offset   zmb.Numeric `json:"offset"`
	Total    zmb.Numeric `json:"total"`
	Next     string      `json:"next"`
	Previous string      `json:"previous"`
}
type PlaylistItem struct {
	AddedAt string                `json:"added_at"`
	AddedBy zmb.User              `json:"added_by"`
	IsLocal bool                  `json:"is_local"`
	Track   zmb.PlaylistItemTrack `json:"item"`
}

func (c *SpotifyClient) fetchPlaylistItems(ctx context.Context, playlistID string, params url.Values) ([]Track, int, error) {
	spotifyURL := fmt.Sprintf("%splaylists/%s/items", baseURL, playlistID)

	if params == nil {
		params = url.Values{}
	}
	if queryString := params.Encode(); queryString != "" {
		spotifyURL += "?" + queryString
	}

	var result PlaylistItemPage

	// GET
	req, err := http.NewRequestWithContext(ctx, "GET", spotifyURL, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}

	defer func() { _ = resp.Body.Close() }()

	// rate limited. TODO
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, 0, fmt.Errorf("spotify: too many requests (status=%d)", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil, 0, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("spotify: status not OK (status=%d)", resp.StatusCode)
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, 0, err
	}

	var tracks []Track
	for _, item := range result.Items {
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
	return tracks, int(result.Total), nil
}

/* END
 *
 */
