package main

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

	spotify "github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

var commandsWithArg = map[SpotifyCommand]bool{
	CmdSeekF:  true,
	CmdSeekB:  true,
	CmdSearch: true,
}

type SpotifyClient struct {
	client *spotify.Client
}

func NewSpotifyClient() (*SpotifyClient, error) {
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	redirectURI := os.Getenv("SPOTIFY_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = "http://127.0.0.1:3000/callback"
	}

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set\n")
	}

	auth := spotifyauth.New(
		spotifyauth.WithClientID(clientID),
		spotifyauth.WithClientSecret(clientSecret),
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(
			spotifyauth.ScopeUserModifyPlaybackState,
			spotifyauth.ScopeUserReadPlaybackState,
		),
	)

	if tok, err := loadToken(); err == nil {
		httpClient := auth.Client(context.Background(), tok)
		return &SpotifyClient{client: spotify.New(httpClient)}, nil
	}

	client, err := runOAuthFlow(auth, redirectURI)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	return &SpotifyClient{client: client}, nil
}

// runOAuthFlow starts a temporary local HTTP server on the redirect URI port,
// opens the Spotify auth page in the browser, and waits for the OAuth callback.
func runOAuthFlow(auth *spotifyauth.Authenticator, redirectURI string) (*spotify.Client, error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return nil, fmt.Errorf("invalid redirect URI %q: %w", redirectURI, err)
	}

	addr := ":" + u.Port()
	callbackPath := u.Path

	const state = "spotify-hands-auth"
	clientCh := make(chan *spotify.Client, 1)
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
			log.Printf("warning: could not cache token: %v", err)
		}
		fmt.Fprintln(w, "Authentication successful! You can close this tab.")
		clientCh <- spotify.New(auth.Client(r.Context(), tok))
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
 * Route validates the incoming action and dispatches it to the appropriate
 * handler. It returns an error if the command or its argument is invalid.
 */
func (c *SpotifyClient) Route(msg SpotifyActionMsg) error {
	cmd := SpotifyCommand(strings.ToUpper(string(msg.Command)))

	switch cmd {
	case CmdPlay:
		if msg.Arg != "" {
			return fmt.Errorf("PLAY takes no argument, got %q", msg.Arg)
		}
		return c.play()

	case CmdPause:
		if msg.Arg != "" {
			return fmt.Errorf("PAUSE takes no argument, got %q", msg.Arg)
		}
		return c.pause()

	case CmdSkipF:
		if msg.Arg != "" {
			return fmt.Errorf("SKIPF takes no argument, got %q", msg.Arg)
		}
		return c.skipForward()

	case CmdSkipB:
		if msg.Arg != "" {
			return fmt.Errorf("SKIPB takes no argument, got %q", msg.Arg)
		}
		return c.skipBack()

	case CmdSeekF:
		s, err := parseSeconds(msg.Arg)
		if err != nil {
			return fmt.Errorf("SEEKF: %w", err)
		}
		return c.seekForward(s)

	case CmdSeekB:
		s, err := parseSeconds(msg.Arg)
		if err != nil {
			return fmt.Errorf("SEEKB: %w", err)
		}
		return c.seekBack(s)

	default:
		return fmt.Errorf("unknown command %q", msg.Command)
	}
}

func (c *SpotifyClient) HandleSearch(msg SpotifyActionMsg) ([]SpotifyItem, error) {
	if msg.Arg == "" {
		return nil, fmt.Errorf("SEARCH requires an argument")
	}

	subcommand, term, _ := strings.Cut(msg.Arg, " ")

	var searchType spotify.SearchType
	switch subcommand {
	case "artist":
		searchType = spotify.SearchTypeArtist
	case "album":
		searchType = spotify.SearchTypeAlbum
	case "track":
		searchType = spotify.SearchTypeTrack
	default:
		return nil, fmt.Errorf("unknown search subcommand %q: expected artist, album, or track", subcommand)
	}

	if strings.TrimSpace(term) == "" {
		return nil, fmt.Errorf("SEARCH %s requires a non-empty search term", subcommand)
	}

	searchResult, err := c.search(term, searchType)
	if err != nil {
		return nil, err
	}

	// metadata will be displayed in insertion order in TUI
	var results []SpotifyItem
	if searchResult.Artists != nil {
		for _, a := range searchResult.Artists.Artists {
			l := Details{
				Name: a.SimpleArtist.Name,
				Metadata: []MetaItem{{
					Label: "followers", Value: strconv.Itoa(int(a.Followers.Count)),
				}},
			}
			results = append(results, SpotifyItem{
				Type:           searchType,
				URI:            a.URI,
				ID:             a.ID,
				ShortViewItems: []string{a.SimpleArtist.Name},
				LongView:       l,
			})
		}
	}
	if searchResult.Albums != nil {
		for _, a := range searchResult.Albums.Albums {
			var artistNames []string
			for _, artist := range a.Artists {
				artistNames = append(artistNames, artist.Name)
			}
			artists := strings.Join(artistNames, ", ")

			l := Details{
				Name: a.Name,
				Metadata: []MetaItem{
					{Label: "artists", Value: artists},
					{Label: "# tracks", Value: strconv.Itoa(int(a.TotalTracks))},
					{Label: "released", Value: a.ReleaseDate},
				},
			}
			results = append(results, SpotifyItem{
				Type:           searchType,
				URI:            a.URI,
				ID:             a.ID,
				ShortViewItems: []string{a.Name, artists},
				LongView:       l,
			})
		}
	}
	if searchResult.Tracks != nil {
		for _, t := range searchResult.Tracks.Tracks {
			var artistNames []string
			for _, artist := range t.Artists {
				artistNames = append(artistNames, artist.Name)
			}
			artists := strings.Join(artistNames, ", ")

			l := Details{
				Name: t.SimpleTrack.Name,
				Metadata: []MetaItem{
					{Label: "artists", Value: artists},
					{Label: "album", Value: t.Album.Name},
					{Label: "duration", Value: strconv.Itoa(int(t.SimpleTrack.Duration))},
				},
			}
			results = append(results, SpotifyItem{
				Type:           searchType,
				URI:            t.URI,
				ID:             t.ID,
				ShortViewItems: []string{t.SimpleTrack.Name, artists},
				LongView:       l,
			})
		}
	}

	return results, nil
}

// parseSeconds validates and converts a seconds string argument.
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

// --- Spotify Web API handlers ---

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

func (c *SpotifyClient) search(query string, t spotify.SearchType) (*spotify.SearchResult, error) {
	result, err := c.client.Search(context.Background(), query, t, spotify.Limit(7))
	if err != nil {
		return nil, fmt.Errorf("could not not perform search: %w", err)
	}
	return result, nil
}

func (c *SpotifyClient) getPlaybackState() (PlaybackState, error) {
	result, err := c.client.PlayerCurrentlyPlaying(context.Background())
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
		Progress:  int(result.Progress), // ms
		IsPlaying: result.Playing,
		Context:   1,
		Track: SpotifyItem{
			Type:           spotify.SearchTypeTrack,
			URI:            t.URI,
			ID:             t.ID,
			ShortViewItems: []string{t.SimpleTrack.Name, artists},
			LongView: Details{
				Name: t.SimpleTrack.Name,
				Metadata: []MetaItem{
					{Label: "artists", Value: artists},
					{Label: "album", Value: t.Album.Name},
					{Label: "duration", Value: strconv.Itoa(int(t.SimpleTrack.Duration))},
				},
			},
		},
	}, nil
}

// PlayTrack starts immediate playback of the given track URI. This is a hack because the PlayOpt function only plays the song then stops playback
func (c *SpotifyClient) playTrack(id spotify.ID) error {
	c.client.QueueSong(context.Background(), id)
	return c.client.Next(context.Background())
}

func (c *SpotifyClient) playFromContext(uri spotify.URI) error {
	return c.client.PlayOpt(context.Background(), &spotify.PlayOptions{
		PlaybackContext: &uri,
	})
}

/*
 * Structs to pass around in frontend 
 */
type SpotifyItem struct {
	Type           spotify.SearchType
	URI            spotify.URI
	ID             spotify.ID
	ShortViewItems []string // will be displayed as * separated string
	LongView       Details
}

type Details struct {
	Name     string
	Metadata []MetaItem // slice will maintain insertion order
}

type MetaItem struct {
	Label string
	Value string
}

// TODO: this could be extended with device, repeat state, shuffle state
type PlaybackState struct {
	Progress	int
	IsPlaying	bool
	Context		int // TODO
	Track		SpotifyItem
}

// ExecutePlayback dispatches a PlaybackMsg to the appropriate Spotify playback endpoint
// based on the item type. Add new cases here to support album, artist, and playlist playback.
func (c *SpotifyClient) ExecutePlayback(msg PlaybackMsg) error {
	switch msg.Item.Type {
	case spotify.SearchTypeTrack:
		return c.playTrack(msg.Item.ID)
	case spotify.SearchTypeAlbum, spotify.SearchTypeArtist:
		return c.playFromContext(msg.Item.URI)
	default:
		return fmt.Errorf("playback not yet supported for type %v", msg.Item.Type)
	}
}

func (c *SpotifyClient) QueueSong(msg QueueMsg) error {
	return c.client.QueueSong(context.Background(), spotify.ID(msg.Id))
}