# CLAUDE.md

## Overview

Spotify Hands Client is a multi-process terminal UI app that lets users control Spotify playback via hand gestures. A Python process handles gesture recognition via webcam and sends commands over a Unix domain socket to a Go TUI client, which calls the Spotify Web API.

## Commands

### Go TUI Client
```bash
go build ./cmd/spotify-tui/   # produces the binary
go run ./cmd/spotify-tui/     # build and run directly
```

### Python Gesture Sender
```bash
# Activate virtual environment first
source .venv/bin/activate

python scripts/sender.py --socket /tmp/spotify-tui.sock        # normal mode (shows webcam window)
python scripts/sender.py --socket /tmp/spotify-tui.sock -d     # headless mode (no OpenCV window)
```

### Running the Full System
Start the TUI client (`go run ./cmd/spotify-tui/` from repo root); it spawns `scripts/sender.py` as a subprocess automatically.

### Setup
```bash
pip install -r requirements.txt
```

## Architecture

**Three main components:**

1. **`sender.py`** — Captures webcam frames, runs MediaPipe `GestureRecognizer` on them, maps detected gestures to action strings (`PLAY`, `PAUSE`, `SKIPF`, `SKIPB`, `SEEKF`, `SEEKB`), and writes them newline-delimited to the Unix socket `/tmp/spotify-tui.sock`. Includes 1.5s debounce. Shuts down on SIGTERM/SIGINT.

2. **Go module** (`go.mod` at repo root, `module aleguy02/spotify-tui`) — structured as:
   - `cmd/spotify-tui/main.go` — thin entry point: setup logging, auth, run BubbleTea program
   - `cmd/spotify-tui/app.go` — root `Model`; routes Spotify/gesture messages, manages gesture subprocess lifecycle; global `q`/`ctrl+c` quit (blocked during `terminalMode`)
   - `internal/ui/menu.go` — `Menu` struct, central state machine. Two browsable tabs (`tabModalities` ↔ `tabNowPlaying`) plus overlay states (`terminalMode`, `searchResultsMode`, `spotifyItemMode`, `helpMode`). `renderLayout()` owns all full-screen placement.
   - `internal/ui/modalities.go` — hand gesture / voice / agent toggle cards
   - `internal/ui/nowplaying.go` — Now Playing screen; receives `SpotifyPlaybackStateMsg` from the tick loop
   - `internal/ui/guide.go` — Help screen with 4 sections; `esc` sends `backToMenuMsg`
   - `internal/ui/search.go` — search results list + `SpotifyItemDetails` interface (`TrackDetails`, `AlbumDetails`, `ArtistDetails` subtypes)
   - `internal/ui/style.go` — `ColorSpotifyGreen` + `Theme` type
   - `internal/ui/logger.go` — `ServiceLogger` + `TerminalLog`, `ModalitiesLog`, `NowPlayingLog`
   - `internal/spotify/spotify.go` — `SpotifyClient`: OAuth2 flow, token persistence, all API calls; shared types (`SpotifyItem`, `Details`, `PlaybackState`)
   - `internal/spotify/messages.go` — BubbleTea message types and `SpotifyCommand` constants
   - `internal/gestures/server.go` — Unix socket server (`StartGestureServer`); injects `SpotifyActionMsg` into the BubbleTea event loop
   - `internal/gestures/messages.go` — `ToggleGesturesMsg`, `GestureClientExitedMsg`, `WaitForGestureCmd`

3. **Spotify Web API** — called directly from Go via `github.com/zmb3/spotify/v2`. Required scopes: `user-modify-playback-state`, `user-read-playback-state`.

**Data flow:**
```
webcam → MediaPipe (Python) → Unix socket → gestures/server.go → BubbleTea msg → spotify/spotify.go → Spotify API
```

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

- **IPC:** Unix domain socket at `/tmp/spotify-tui.sock`; messages are newline-terminated ASCII strings
- **Gesture model:** `gesture_recognizer.task` (MediaPipe, gitignored, must be present in repo root)
- **Named gestures mapped:** `Open_Palm → PLAY`, `Closed_Fist → PAUSE`, `Thumb_Up → SKIPF`, `Thumb_Down → SKIPB`; custom finger-landmark logic handles `SEEKF`/`SEEKB`
- **Layout pattern:** sub-views (`modalities`, `nowplaying`, `guide`) return raw content from `View()`; `Menu.renderLayout()` does full-screen `lipgloss.Place` after reserving rows for the bottom bar — never do full-screen placement inside a sub-view
- **`SpotifyItemDetails` interface:** `View() string`, `ItemType() string`, `RawItem() SpotifyItem` + dead-code stubs for `relatedItems()` / `userStats()`; `theme` state stored in `Menu` struct (dead code until wired)
- **TUI debug log:** `debug.log` at CWD (BubbleTea `LogToFile`, gitignored)
- **Go module:** `aleguy02/spotify-tui`, requires Go 1.25.6+; run commands from repo root
- **Python runtime:** 3.13 (`.venv/lib/python3.13`); gesture sender at `scripts/sender.py`
