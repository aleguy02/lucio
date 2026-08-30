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
	"sync"
	"time"

	zmb "github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

var confPath = "conf.yaml"

type spotifyConf struct {
	ClientID     string `yaml:"spotify_client_id"`
	ClientSecret string `yaml:"spotify_client_secret"`
	RedirectURI  string `yaml:"spotify_redirect_uri"`
}

var logger *log.Logger

func init() {
	f, err := os.OpenFile(filepath.Join("logs", "spotify.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}
	logger = log.New(f, "", log.LstdFlags)
}

type SpotifyClient struct {
	client     *zmb.Client
	httpClient *http.Client
	nameCache  sync.Map
}

func (c *SpotifyClient) setCache(id, name string) {
	if id == "" || name == "" {
		return
	}
	c.nameCache.Store(id, name)
}

func (c *SpotifyClient) getCached(id string) (string, bool) {
	v, ok := c.nameCache.Load(id)
	if !ok {
		return "", false
	}
	return v.(string), true
}

func NewSpotifyClient() (*SpotifyClient, error) {
	data, err := os.ReadFile(confPath)
	if err != nil {
		logger.Printf("error reading file: %v", err)
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	c := spotifyConf{}
	if err := yaml.Unmarshal(data, &c); err != nil {
		logger.Printf("could not unmarshal yaml: %v", err)
		return nil, fmt.Errorf("could not unmarshal yaml: %w", err)
	}

	if c.ClientID == "" || c.ClientSecret == "" || c.RedirectURI == "" {
		logger.Printf("spotify_client_id, spotify_client_secret, and spotify_redirect_uri must be set")
		return nil, fmt.Errorf("spotify_client_id, spotify_client_secret, and spotify_redirect_uri must be set")
	}

	auth := spotifyauth.New(
		spotifyauth.WithClientID(c.ClientID),
		spotifyauth.WithClientSecret(c.ClientSecret),
		spotifyauth.WithRedirectURL(c.RedirectURI),
		spotifyauth.WithScopes(
			spotifyauth.ScopeUserModifyPlaybackState,
			spotifyauth.ScopeUserReadPlaybackState,
			spotifyauth.ScopePlaylistReadPrivate,
			spotifyauth.ScopePlaylistReadCollaborative,
			spotifyauth.ScopePlaylistModifyPublic,
			spotifyauth.ScopePlaylistModifyPrivate,
			spotifyauth.ScopeUserLibraryModify,
			spotifyauth.ScopeUserFollowModify,
		),
	)

	if tok, err := loadToken(); err == nil {
		httpClient := auth.Client(context.Background(), tok)
		return &SpotifyClient{client: zmb.New(httpClient), httpClient: httpClient}, nil
	}

	client, httpClient, err := runOAuthFlow(auth, c.RedirectURI)
	if err != nil {
		logger.Printf("authentication failed: %v", err)
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	return &SpotifyClient{client: client, httpClient: httpClient}, nil
}

/*
 * Auth Flow
 */
func runOAuthFlow(auth *spotifyauth.Authenticator, redirectURI string) (*zmb.Client, *http.Client, error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid redirect URI %q: %w", redirectURI, err)
	}

	addr := ":" + u.Port()
	callbackPath := u.Path

	const state = "spotify-hands-auth"
	httpClientCh := make(chan *http.Client, 1)
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
		httpClientCh <- auth.Client(r.Context(), tok)
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
	case httpClient := <-httpClientCh:
		client := zmb.New(httpClient)
		fmt.Println("Authenticated successfully.")
		return client, httpClient, nil
	case err := <-errCh:
		return nil, nil, err
	case <-time.After(5 * time.Minute):
		return nil, nil, fmt.Errorf("authentication timed out after 5 minutes")
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
