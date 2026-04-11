package main

import (
	"fmt"
	"strconv"
	"strings"
)

var commandsWithArg = map[SpotifyCommand]bool{
	CmdSeekF:  true,
	CmdSeekB:  true,
	CmdSearch: true,
}

// SpotifyClient wraps the Spotify Web API.
type SpotifyClient struct{}

func NewSpotifyClient() *SpotifyClient {
	return &SpotifyClient{}
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

	case CmdSearch:
		if strings.TrimSpace(msg.Arg) == "" {
			return fmt.Errorf("SEARCH requires a non-empty search term")
		}
		return c.search(msg.Arg)

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

// --- stub handlers (Spotify Web API calls go here) ---

func (c *SpotifyClient) play() error {
	fmt.Println("[spotify] PLAY")
	return nil
}

func (c *SpotifyClient) pause() error {
	fmt.Println("[spotify] PAUSE")
	return nil
}

func (c *SpotifyClient) skipForward() error {
	fmt.Println("[spotify] SKIPF")
	return nil
}

func (c *SpotifyClient) skipBack() error {
	fmt.Println("[spotify] SKIPB")
	return nil
}

func (c *SpotifyClient) seekForward(s int) error {
	fmt.Printf("[spotify] SEEKF %ds\n", s)
	return nil
}

func (c *SpotifyClient) seekBack(s int) error {
	fmt.Printf("[spotify] SEEKB %ds\n", s)
	return nil
}

func (c *SpotifyClient) search(term string) error {
	fmt.Printf("[spotify] SEARCH %q\n", term)
	return nil
}
