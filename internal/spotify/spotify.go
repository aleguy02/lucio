package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	zmb "github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

var logger *log.Logger

func init() {
	f, err := os.OpenFile(filepath.Join("logs", "spotify.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}
	logger = log.New(f, "", log.LstdFlags)
}

type SpotifyClient struct {
	client *zmb.Client
}

func NewSpotifyClient() (*SpotifyClient, error) {
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	redirectURI := os.Getenv("SPOTIFY_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = "http://127.0.0.1:3000/callback"
	}

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set")
	}

	auth := spotifyauth.New(
		spotifyauth.WithClientID(clientID),
		spotifyauth.WithClientSecret(clientSecret),
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(
			spotifyauth.ScopeUserModifyPlaybackState,
			spotifyauth.ScopeUserReadPlaybackState,
			spotifyauth.ScopePlaylistReadPrivate,
			spotifyauth.ScopePlaylistReadCollaborative,
		),
	)

	if tok, err := loadToken(); err == nil {
		httpClient := auth.Client(context.Background(), tok)
		return &SpotifyClient{client: zmb.New(httpClient)}, nil
	}

	client, err := runOAuthFlow(auth, redirectURI)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	return &SpotifyClient{client: client}, nil
}

/*
 * Auth Flow
 */
func runOAuthFlow(auth *spotifyauth.Authenticator, redirectURI string) (*zmb.Client, error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return nil, fmt.Errorf("invalid redirect URI %q: %w", redirectURI, err)
	}

	addr := ":" + u.Port()
	callbackPath := u.Path

	const state = "spotify-hands-auth"
	clientCh := make(chan *zmb.Client, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	server := &http.Server{Addr: addr, Handler: mux}

	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		if st := r.FormValue("state"); st != state {
			http.Error(w, "state mismatch", http.StatusForbidden)
			errCh <- fmt.Errorf("OAuth state mismatch (got %q)", st)
			return
		}
		tok, err := auth.Token(r.Context(), state, r)
		if err != nil {
			http.Error(w, "could not get token", http.StatusForbidden)
			errCh <- fmt.Errorf("could not exchange code for token: %w", err)
			return
		}
		if err := saveToken(tok); err != nil {
			logger.Printf("warning: could not cache token: %v", err)
		}
		if _, err := fmt.Fprintln(w, "Authentication successful! You can close this tab."); err != nil {
			logger.Printf("failed to write authentication success message: %v", err)
		}
		clientCh <- zmb.New(auth.Client(r.Context(), tok))
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server: %w", err)
		}
	}()

	authURL := auth.AuthURL(state)
	fmt.Printf(
		"\nOpen the following URL in your browser to authenticate with Spotify:\n\n  %s\n\nWaiting for authentication...\n",
		authURL,
	)
	openBrowser(authURL)

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	select {
	case client := <-clientCh:
		fmt.Println("Authenticated successfully.")
		return client, nil
	case err := <-errCh:
		return nil, err
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("authentication timed out after 5 minutes")
	}
}

func openBrowser(rawURL string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "linux":
		cmd = exec.Command("xdg-open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		return
	}
	_ = cmd.Start()
}

func tokenPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "spotify-hands-client", "token.json"), nil
}

func loadToken() (*oauth2.Token, error) {
	path, err := tokenPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

func saveToken(tok *oauth2.Token) error {
	path, err := tokenPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(tok)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

/*
 * Internal Logic
 */
// Route dispatches a SpotifyActionMsg to the appropriate client method.
// It returns an optional result string (non-empty for commands that surface data,
// e.g. "devices list") and an error.
// TODO(refactor): I think this should be refactored to HandlePlaybackAction and make it specific to playback actions
// and extract devices command to loop
func (c *SpotifyClient) Route(msg SpotifyActionMsg) (string, error) {
	cmd := SpotifyCommand(strings.ToUpper(string(msg.Command)))

	switch cmd {
	case CmdPlay:
		if msg.Arg != "" {
			return "", fmt.Errorf("PLAY takes no argument, got %q", msg.Arg)
		}
		return "", c.play()

	case CmdPause:
		if msg.Arg != "" {
			return "", fmt.Errorf("PAUSE takes no argument, got %q", msg.Arg)
		}
		return "", c.pause()

	case CmdSkipF:
		if msg.Arg != "" {
			return "", fmt.Errorf("SKIPF takes no argument, got %q", msg.Arg)
		}
		return "", c.skipForward()

	case CmdSkipB:
		if msg.Arg != "" {
			return "", fmt.Errorf("SKIPB takes no argument, got %q", msg.Arg)
		}
		return "", c.skipBack()

	case CmdSeekF:
		s, err := parseSeconds(msg.Arg)
		if err != nil {
			return "", fmt.Errorf("SEEKF: %w", err)
		}
		return "", c.seekForward(s)

	case CmdSeekB:
		s, err := parseSeconds(msg.Arg)
		if err != nil {
			return "", fmt.Errorf("SEEKB: %w", err)
		}
		return "", c.seekBack(s)

	case CmdShuffle:
		sub := strings.TrimSpace(msg.Arg)
		switch sub {
		case "on":
			return "", c.toggleShuffle(true)
		case "off":
			return "", c.toggleShuffle(false)
		case "":
			return "", fmt.Errorf("SHUFFLE requires on or off")
		default:
			return "", fmt.Errorf("unknown SHUFFLE argument %q: expected on or off", sub)
		}

	case CmdDevices:
		sub, id, _ := strings.Cut(strings.TrimSpace(msg.Arg), " ")
		switch sub {
		case "list":
			names, err := c.GetDeviceNames()
			return names, err
		case "select":
			id = strings.TrimSpace(id)
			if id == "" {
				return "", fmt.Errorf("DEVICES SELECT requires a device ID")
			}
			return "", c.selectDevice(id)
		default:
			return "", fmt.Errorf("unknown DEVICES subcommand %q: expected list or select", sub)
		}

	default:
		return "", fmt.Errorf("unknown command %q", msg.Command)
	}
}

func (c *SpotifyClient) HandleSearch(msg SpotifyActionMsg) ([]SpotifyItem, error) {
	if msg.Arg == "" {
		return nil, fmt.Errorf("SEARCH requires an argument")
	}

	subcommand, term, _ := strings.Cut(msg.Arg, " ")

	var searchType zmb.SearchType
	switch subcommand {
	case "artist":
		searchType = zmb.SearchTypeArtist
	case "album":
		searchType = zmb.SearchTypeAlbum
	case "track":
		searchType = zmb.SearchTypeTrack
	case "playlist":
		searchType = zmb.SearchTypePlaylist
	default:
		return nil, fmt.Errorf("unknown search subcommand %q: expected artist, album, track, or playlist", subcommand)
	}

	if strings.TrimSpace(term) == "" {
		return nil, fmt.Errorf("SEARCH %s requires a non-empty search term", subcommand)
	}

	// searching playlists sometimes returns lots of empty/null results, so we do a bounded loop until we find 7 results
	const limit = 7
	const retries = 3
	var results []SpotifyItem
	offset := 0
	found := 0

	for found < limit && offset < retries {
		tmpResult, err := c.search(term, searchType, offset*limit)
		if err != nil {
			return nil, err
		}

		iterFound := 0
		switch searchType {
		case zmb.SearchTypeArtist:
			if tmpResult.Artists != nil {
				for _, a := range tmpResult.Artists.Artists {
					if string(a.ID) == "" {
						continue
					}
					results = append(results, SpotifyItem{
						Type:           searchType,
						URI:            a.URI,
						ID:             a.ID,
						ShortViewItems: []string{a.Name},
						LongView: Details{
							Name: a.Name,
							Metadata: []MetaItem{{
								Label: "followers", Value: strconv.Itoa(int(a.Followers.Count)),
							}},
						},
					})
					iterFound++
				}
			}
		case zmb.SearchTypeAlbum:
			if tmpResult.Albums != nil {
				for _, a := range tmpResult.Albums.Albums {
					if string(a.ID) == "" {
						continue
					}
					var artistNames []string
					for _, artist := range a.Artists {
						artistNames = append(artistNames, artist.Name)
					}
					artists := strings.Join(artistNames, ", ")
					results = append(results, SpotifyItem{
						Type:           searchType,
						URI:            a.URI,
						ID:             a.ID,
						ShortViewItems: []string{a.Name, artists},
						LongView: Details{
							Name: a.Name,
							Metadata: []MetaItem{
								{Label: "artists", Value: artists},
								{Label: "# tracks", Value: strconv.Itoa(int(a.TotalTracks))},
								{Label: "released", Value: a.ReleaseDate},
							},
						},
					})
					iterFound++
				}
			}
		case zmb.SearchTypeTrack:
			if tmpResult.Tracks != nil {
				for _, t := range tmpResult.Tracks.Tracks {
					if string(t.ID) == "" {
						continue
					}
					var artistNames []string
					for _, artist := range t.Artists {
						artistNames = append(artistNames, artist.Name)
					}
					artists := strings.Join(artistNames, ", ")
					results = append(results, SpotifyItem{
						Type:           searchType,
						URI:            t.URI,
						ID:             t.ID,
						ShortViewItems: []string{t.Name, artists},
						LongView: Details{
							Name: t.Name,
							Metadata: []MetaItem{
								{Label: "artists", Value: artists},
								{Label: "album", Value: t.Album.Name},
								{Label: "duration", Value: strconv.Itoa(int(t.Duration))},
							},
						},
					})
					iterFound++
				}
			}
		case zmb.SearchTypePlaylist:
			if tmpResult.Playlists != nil {
				for _, p := range tmpResult.Playlists.Playlists {
					if string(p.ID) == "" {
						continue
					}
					var collaborative string
					if p.Collaborative {
						collaborative = "yes"
					} else {
						collaborative = "no"
					}
					metadata := []MetaItem{
						{Label: "owner", Value: p.Owner.DisplayName},
						// {Label: "# tracks", Value: strconv.Itoa(int(p.Tracks.Total))},   // the spotify API doesn't track this anymore so it was always 0
						{Label: "collaborative", Value: collaborative},
					}
					if p.Description != "" {
						metadata = append(metadata, MetaItem{Label: "description", Value: p.Description})
					}
					results = append(results, SpotifyItem{
						Type:           searchType,
						URI:            p.URI,
						ID:             p.ID,
						ShortViewItems: []string{p.Name, p.Owner.DisplayName},
						LongView: Details{
							Name:     p.Name,
							Metadata: metadata,
						},
					})
					iterFound++
				}
			}
		}

		found += iterFound
		logger.Printf("search [%s %q] attempt %d: found %d this iteration, %d total", subcommand, term, offset, iterFound, found)
		offset++
	}

	return results, nil
}

func (c *SpotifyClient) HandlePlaylists() ([]SpotifyItem, error) {
	page, err := c.client.CurrentUsersPlaylists(context.Background(), zmb.Limit(7))
	if err != nil {
		return nil, fmt.Errorf("could not get playlists: %w", err)
	}

	var results []SpotifyItem
	for _, p := range page.Playlists {
		var collaborative string
		if p.Collaborative {
			collaborative = "yes"
		} else {
			collaborative = "no"
		}

		metadata := []MetaItem{
			{Label: "owner", Value: p.Owner.DisplayName},
			// {Label: "# tracks", Value: strconv.Itoa(int(p.Tracks.Total))},   // the spotify API doesn't track this anymore so it was always 0
			{Label: "collaborative", Value: collaborative},
		}

		if p.Description != "" {
			metadata = append(metadata, MetaItem{Label: "description", Value: p.Description})
		}

		results = append(results, SpotifyItem{
			Type:           zmb.SearchTypePlaylist,
			URI:            p.URI,
			ID:             p.ID,
			ShortViewItems: []string{p.Name, p.Owner.DisplayName},
			LongView: Details{
				Name:     p.Name,
				Metadata: metadata,
			},
		})
	}
	return results, nil
}

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
	return c.client.Play(context.Background())
}

func (c *SpotifyClient) pause() error {
	return c.client.Pause(context.Background())
}

func (c *SpotifyClient) skipForward() error {
	return c.client.Next(context.Background())
}

func (c *SpotifyClient) skipBack() error {
	return c.client.Previous(context.Background())
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
	return c.client.Seek(context.Background(), newPos)
}

func (c *SpotifyClient) toggleShuffle(curShuffleState bool) error {
	return c.client.Shuffle(context.Background(), curShuffleState)
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
	return c.client.Seek(context.Background(), newPos)
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

// GetDeviceNames returns a comma-separated list of available Spotify devices,
// marking the currently active one with "(active)".
func (c *SpotifyClient) GetDeviceNames() (string, error) {
	devices, err := c.getDevices()
	if err != nil {
		return "", err
	}
	if len(devices) == 0 {
		return "no devices available", nil
	}
	names := make([]string, 0, len(devices))
	for _, d := range devices {
		name := d.Name + " [" + string(d.ID) + "]"
		if d.Active {
			name += " (active)"
		}
		names = append(names, name)
	}
	return strings.Join(names, ", "), nil
}

func (c *SpotifyClient) GetPlaybackState() (PlaybackState, error) {
	result, err := c.client.PlayerState(context.Background())
	if err != nil {
		return PlaybackState{}, fmt.Errorf("could not get playback state: %w", err)
	}
	if result.Item == nil {
		return PlaybackState{IsPlaying: false}, nil
	}
	t := result.Item
	var artistNames []string
	for _, artist := range t.Artists {
		artistNames = append(artistNames, artist.Name)
	}
	artists := strings.Join(artistNames, ", ")

	return PlaybackState{
		Progress:     int(result.Progress),
		IsPlaying:    result.Playing,
		ShuffleState: result.ShuffleState,
		Context:      1,
		Track: SpotifyItem{
			Type:           zmb.SearchTypeTrack,
			URI:            t.URI,
			ID:             t.ID,
			ShortViewItems: []string{t.Name, artists},
			LongView: Details{
				Name: t.Name,
				Metadata: []MetaItem{
					{Label: "artists", Value: artists},
					{Label: "album", Value: t.Album.Name},
					{Label: "duration", Value: strconv.Itoa(int(t.Duration))},
				},
			},
		},
	}, nil
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
	return c.client.PlayOpt(context.Background(), &zmb.PlayOptions{
		PlaybackContext: &uri,
	})
}

func (c *SpotifyClient) ExecutePlayback(msg PlaybackMsg) error {
	switch msg.Item.Type {
	case zmb.SearchTypeTrack:
		return c.playTrack(msg.Item.ID)
	case zmb.SearchTypeAlbum, zmb.SearchTypeArtist, zmb.SearchTypePlaylist:
		return c.playFromContext(msg.Item.URI)
	default:
		return fmt.Errorf("playback not yet supported for type %v", msg.Item.Type)
	}
}

func (c *SpotifyClient) QueueSong(msg QueueMsg) error {
	return c.client.QueueSong(context.Background(), zmb.ID(msg.Id))
}

// SpotifyItem is the shared representation of a Spotify entity used across the UI.
type SpotifyItem struct {
	Type zmb.SearchType
	URI  zmb.URI
	ID   zmb.ID

	// Items to show to user in small models. These should be important information.
	// For example, the name and creator of a track/playlist/album.
	ShortViewItems []string
	LongView       Details
}

type Details struct {
	Name     string
	Metadata []MetaItem
	// TODO(feat): should we add a little "extra metadata" field? It's what ADK does for some types
	// and we could use it to display, say, the if a playlist is collaborative or a song is explicit
	// things people don't care about that much. Or we could put important navigation data (IDs or something)
}

type MetaItem struct {
	Label string
	Value string
}

// TODO(feat): could be extended with device, repeat state
type PlaybackState struct {
	Progress     int
	IsPlaying    bool
	ShuffleState bool
	Context      int // TODO
	Track        SpotifyItem
}
