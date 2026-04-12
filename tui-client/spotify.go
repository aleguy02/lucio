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

// SpotifyClient wraps the Spotify Web API.
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
