---
name: spotify_web_api
description: Browse Spotify Web API documentation and patterns. Use when working with the Spotify Web API in Go.
---

# Spotify Web API Skill

## Package

Always import as:
```go
zmb "github.com/zmb3/spotify/v2"
spotifyauth "github.com/zmb3/spotify/v2/auth"
```

**Local source** (read this before going online): `/Users/avillate/go/pkg/mod/github.com/zmb3/spotify/v2@v2.4.3/`

Each API domain has its own file — grep the relevant one rather than fetching docs:
| Domain | Local file |
|---|---|
| Search | `search.go` |
| Playback / player state | `player.go` |
| Playlists | `playlist.go` |
| Albums | `album.go` |
| Artists | `artist.go` |
| Tracks | `track.go` |
| User library (saved) | `library.go` |
| User profile | `user.go` |
| Recommendations | `recommendation.go` |
| Audio features / analysis | `audio_features.go`, `audio_analysis.go` |
| Auth scopes | `auth/` subdirectory |

**Official docs** (use only when local source isn't enough): https://developer.spotify.com/documentation/web-api

---

## Project patterns

### The wrapper type

All Spotify calls go through `SpotifyClient` in `internal/spotify/spotify.go`, which holds a `*zmb.Client`. Always pass `context.Background()`.

```go
func (c *SpotifyClient) myNewMethod(...) error {
    return c.client.SomeZmbMethod(context.Background(), ...)
}
```

### Key zmb types used in this codebase

| zmb type | Used for |
|---|---|
| `zmb.SearchType` | Type tag on `SpotifyItem.Type` |
| `zmb.SearchTypeTrack/Artist/Album/Playlist` | Constants for the above |
| `zmb.URI` | Stored on `SpotifyItem.URI`; used for context playback |
| `zmb.ID` | Stored on `SpotifyItem.ID`; used for track queueing |
| `zmb.PlayerState` | Returned by `c.client.PlayerState(...)` |
| `zmb.PlayerDevice` | Returned by `c.client.PlayerDevices(...)` |
| `zmb.SearchResult` | Returned by `c.client.Search(...)` |
| `zmb.PlayOptions` | Used with `PlayOpt` for context/offset playback |

### Shared UI types (internal/spotify/spotify.go)

New API data gets mapped into these before being handed to the UI:

```go
type SpotifyItem struct {
    Type           zmb.SearchType
    URI            zmb.URI
    ID             zmb.ID
    ShortViewItems []string  // name + key secondary info, shown in list view
    LongView       Details
}

type Details struct {
    Name     string
    Metadata []MetaItem
}

type MetaItem struct{ Label, Value string }
```

### Adding a new search type

1. Add a `case` to the `switch subcommand` block in `HandleSearch` (`spotify.go`)
2. Add a result-mapping block below the existing `if searchResult.X != nil` blocks
3. Add a `XxxDetails` struct in `internal/ui/search.go` implementing `SpotifyItemDetails`
4. Add a `case zmb.SearchTypeXxx` to `NewSpotifyItemDetails`

### Adding a new playback action

- Track: use `c.client.QueueSong` + `c.client.Next` (not `PlayOpt` alone — it stops after one track)
- Album / Artist / Playlist context: use `c.client.PlayOpt` with `PlaybackContext: &uri`
- See `ExecutePlayback` and `playTrack`/`playFromContext` for the existing pattern

### Auth scopes

Scopes are set in `NewSpotifyClient`. When a new API call needs a new scope, add it to the `spotifyauth.WithScopes(...)` call. Available scope constants live in `spotifyauth` (`ScopeUserModifyPlaybackState`, `ScopeUserReadPlaybackState`, etc.). Check `auth/scopes.go` in the local package for the full list.

### BubbleTea messages

New result types go in `internal/spotify/messages.go`. Follow the existing naming (`XxxMsg`, `XxxResultMsg`). Commands are `SpotifyCommand` string constants in `messages.go`.
