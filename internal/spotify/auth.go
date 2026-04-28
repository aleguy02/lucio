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
			// spotifyauth.ScopeUserLibraryModify,
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
