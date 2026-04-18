# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Spotify Hands Client is a multi-process terminal UI app that lets users control Spotify playback via hand gestures. A Python process handles gesture recognition via webcam and sends commands over a Unix domain socket to a Go TUI client, which calls the Spotify Web API.

## Commands

### Go TUI Client
```bash
cd tui-client/
go build           # produces the binary
go run .           # build and run directly
```

### Python Gesture Sender
```bash
# Activate virtual environment first
source .venv/bin/activate

python sender.py --socket /tmp/spotify-tui.sock        # normal mode (shows webcam window)
python sender.py --socket /tmp/spotify-tui.sock -d     # headless mode (no OpenCV window)
```

### Running the Full System
Start the TUI client (`go run .` from `tui-client/`); it spawns `sender.py` as a subprocess automatically.

### Setup
```bash
pip install -r requirements.txt
```
Read `.env` to get Spotify credentials.

## Architecture

**Three main components:**

1. **`sender.py`** — Captures webcam frames, runs MediaPipe `GestureRecognizer` on them, maps detected gestures to action strings (`PLAY`, `PAUSE`, `SKIPF`, `SKIPB`, `SEEKF`, `SEEKB`), and writes them newline-delimited to the Unix socket `/tmp/spotify-tui.sock`. Includes 1.5s debounce. Shuts down on SIGTERM/SIGINT.

2. **`tui-client/`** — Go BubbleTea app. Owns the socket server (`gestureServer.go`), Spotify API integration (`spotify.go`), and all UI views. Key files:
   - `main.go` — entry point, spawns/manages the Python subprocess; global `q`/`ctrl+c` quit (blocked during `terminalMode`)
   - `menu.go` — central state machine owning all views. Two browsable tabs (`tabModalities` ↔ `tabNowPlaying`, cycled with `tab`/`shift+tab`) plus overlay states (`terminalMode`, `searchResultsMode`, `spotifyItemMode`, `helpMode`). `:` enters terminal mode from any tab. `renderLayout()` owns all full-screen placement, reserving rows for the bottom bar.
   - `modalities.go` — hand gesture / voice / agent toggle cards; returns raw card row from `View()` (placement handled by `menu.go`)
   - `nowplaying.go` — static Now Playing screen with dummy data; returns raw content from `View()` (placement handled by `menu.go`)
   - `guide.go` — Help screen with 4 sections (Terminal, Gesture, Voice, Agent) cycled with `tab`/`shift+tab`; `esc` sends `backToMenuMsg`
   - `search.go` — search results list + `SpotifyItemDetails` interface (`TrackDetails`, `AlbumDetails`, `ArtistDetails` subtypes). Factory: `NewSpotifyItemDetails(item)`
   - `spotify.go` — OAuth2 flow, token persistence, routes action strings to Spotify API calls
   - `gestureServer.go` — Unix socket server; injects `SpotifyActionMsg` into the BubbleTea event loop
   - `style.go` — `ColorSpotifyGreen` + `Theme` type (`ThemeDefault`, `ThemeMinimalist`, `ThemeVibes`)
   - `messages.go` — custom BubbleTea message types
   - `logger.go` — service-prefixed logging (prefix with `[gestureServer]`, `[terminalMode]`, etc.)

3. **Spotify Web API** — called directly from Go via `github.com/zmb3/spotify/v2`. Required scopes: `user-modify-playback-state`, `user-read-playback-state`.

**Data flow:**
```
webcam → MediaPipe (Python) → Unix socket → gestureServer.go → BubbleTea msg → spotify.go → Spotify API
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

### Terminal mode commands
`PLAY`, `PAUSE`, `SKIPF`, `SKIPB`, `SEEKF <s>`, `SEEKB <s>`, `SEARCH artist/album/track <query>`, `details`, `theme default|minimalist|vibes`

## Key Technical Details

- **IPC:** Unix domain socket at `/tmp/spotify-tui.sock`; messages are newline-terminated ASCII strings
- **Gesture model:** `gesture_recognizer.task` (MediaPipe, gitignored, must be present in repo root)
- **Named gestures mapped:** `Open_Palm → PLAY`, `Closed_Fist → PAUSE`, `Thumb_Up → SKIPF`, `Thumb_Down → SKIPB`; custom finger-landmark logic handles `SEEKF`/`SEEKB`
- **Layout pattern:** sub-views (`modalities`, `nowplaying`, `guide`) return raw content from `View()`; `menu.go`'s `renderLayout()` does full-screen `lipgloss.Place` after reserving rows for the bottom bar — never do full-screen placement inside a sub-view
- **`SpotifyItemDetails` interface:** `View() string`, `ItemType() string`, `RawItem() SpotifyItem` + dead-code stubs for `relatedItems()` / `userStats()`; `theme` state stored in `menu` struct (dead code until wired)
- **TUI debug log:** `tui-client/debug.log` (BubbleTea `LogToFile`, gitignored)
- **Go module:** `aleguy02/spotify-tui`, requires Go 1.25.6+
- **Python runtime:** 3.13 (`.venv/lib/python3.13`)
