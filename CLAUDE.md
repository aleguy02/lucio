# CLAUDE.md

## Overview

Spotify TUI Client is a terminal UI app that lets users control Spotify playback. It includes a Go TUI client, which calls the Spotify Web API, and integrates an agent runner for agentic control.

## Commands

### Go TUI Client
```bash
go build ./cmd/spotify-tui/   # produces the binary
go run ./cmd/spotify-tui/     # build and run directly
```

## Architecture

**Key components:**

1. **Go module** (`go.mod` at repo root, `module aleguy02/spotify-tui`) — structured as:
   - `cmd/spotify-tui/main.go` — thin entry point: setup logging, auth, run BubbleTea program
   - `cmd/spotify-tui/app.go` — root `Model`; routes Spotify/agent messages; global `q`/`ctrl+c` quit (blocked during `terminalMode`)
   - `internal/ui/menu.go` — `Menu` struct, central state machine. Browsable tabs plus overlay states (`terminalMode`, `searchResultsMode`, `spotifyItemMode`, `helpMode`). `renderLayout()` owns all full-screen placement.
   - `internal/ui/modalities.go` — agent toggle card
   - `internal/ui/nowplaying.go` — Now Playing screen; receives `SpotifyPlaybackStateMsg` from the tick loop
   - `internal/ui/guide.go` — Help screen with sections; `esc` sends `backToMenuMsg`
   - `internal/ui/search.go` — search results list + `SpotifyItemDetails` interface (`TrackDetails`, `AlbumDetails`, `ArtistDetails` subtypes)
   - `internal/ui/style.go` — `ColorSpotifyGreen` + `Theme` type
   - `internal/ui/logger.go` — `ServiceLogger` + `TerminalLog`, `ModalitiesLog`, `NowPlayingLog`
   - `internal/spotify/spotify.go` — `SpotifyClient`: OAuth2 flow, token persistence, all API calls; shared types (`SpotifyItem`, `Details`, `PlaybackState`)
   - `internal/spotify/messages.go` — BubbleTea message types and `SpotifyCommand` constants

2. **Spotify Web API** — called directly from Go via `github.com/zmb3/spotify/v2`. Required scopes: `user-modify-playback-state`, `user-read-playback-state`.

## TUI Navigation

| View | How to reach | How to leave |
|---|---|---|
| Modalities | default on launch; `tab`/`shift+tab` from Now Playing | `tab`/`shift+tab` |
| Now Playing | `tab`/`shift+tab` from Modalities | `tab`/`shift+tab` |
| Help | type `h` in terminal mode | `esc` |
| Search Results | type `search artist/album/track <query>` in terminal mode | `esc` |
| Item Details | `tab` on a search result, or `details` in terminal mode | `esc` |
| Terminal mode | `:` from any tab | `esc` or `enter` |

## Key Technical Details

- **Layout pattern:** sub-views (`modalities`, `nowplaying`, `guide`) return raw content from `View()`; `Menu.renderLayout()` does full-screen `lipgloss.Place` after reserving rows for the bottom bar — never do full-screen placement inside a sub-view
- **`SpotifyItemDetails` interface:** `View() string`, `ItemType() string`, `RawItem() SpotifyItem` + dead-code stubs for `relatedItems()` / `userStats()`; `theme` state stored in `Menu` struct (dead code until wired)
- **TUI debug log:** `debug.log` at CWD (BubbleTea `LogToFile`, gitignored)
- **Go module:** `aleguy02/spotify-tui`, requires Go 1.25.6+; run commands from repo root
