package spotify

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	zmb "github.com/zmb3/spotify/v2"
)

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
	err := c.client.QueueSong(context.Background(), zmb.ID(msg.Id))
	if err != nil {
		logger.Printf("QueueSong %q: %v", msg.Id, err)
	}
	return err
}

// func (c *SpotifyClient) LikeTrack(id string) error {
// 	err := c.client.AddTracksToLibrary(context.Background(), zmb.ID(id))
// 	if err != nil {
// 		logger.Printf("LikeTrack %q: %v", id, err)
// 	}
// 	return err
// }
