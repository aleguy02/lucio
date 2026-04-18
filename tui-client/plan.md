Your task is to do a MAJOR frontend refactor. The frontend is the bubbletea interface which currently supports a menu page and a help page, and some other components. You must overhaul the frontend pages and navigation. You won't be wiring up a lot of code so it should mostly be static changes.

# TASK

You will update 4 main tabs:  

## Modalities Screen
This is already implemented in modalities.go. You should center the list of modalities.

## Currently Playing Screen
Displays (from top to bottom) context (i.e. radio, songs like, shuffling playlist, playin album, etc.), genre, BOX WITH VISUALS, song name, album name, artist name(s), playback time. Use static values for now, DO NOT wire up the spotify client. Vertically center the elements. BOX WITH VISUALS should be a spinner for now. This is the main visual of the app so make the vibes immaculate. Looking at it should be like looking at a lavalamp or being at a music festival.

You must also add boilerplate code to dynamically render visuals depending on the user's "theme", which they should be able to set with a "theme THEME_NAME" command from terminal mode. Maintain this theme in the app model's state. It should default to "default" and support two other options "minimalist" and "vibes". Don't implement the other themes just allow the user to toggle them. This should be dead code that we will wire up later.

## Help Screen
V1 is implemented in guide.go, we're going to modify it. From inside the help screen, the user should be able to press tab to toggle "which" help screen. There should be a help screen for each of the modalities: terminalMode, gesture mode, voice mode, and ai agent mode.

- Terminal Mode help screen should document the Spotify Client commands. You can find them in the Route and HandleSearch functions in spotify.go.
- Gesture Mode should document the hand gestures and corresponding functions: GESTURE_MAP = {"Open_Palm": "PLAY", "Closed_Fist": "PAUSE", "Thumb_Up": "SKIPF", "Thumb_Down":  "SKIPB"}
- Voice Mode and Agent Mode should be blank for now

## Details Tab

V1 is implemented in search.go SpotifyItem model. Overhaul it to be a generic interface that subtypes can satisfy because we will eventually make an artist, album, and track details screen. You must update search mode to direct to this screen when we hit tab on a search result. Make boilerplate for the subtypes that just displays which type of subtype it is (to prove that it is working). We will wire up this details tab later.

Every subtype will be able to display details, navigate to related items (like a graph but don't worry about it yet), and show user's stats related to this item (don't wire up yet).

###
All the main tab views should have v.AltScreen = true. terminalMode should be accessible from anywhere with the ":" keyboard shortcut

RECAP/SUMMARY: You must overhaul the frontend to make 4 main tabs. Your new frontend will be mostly static or with dummy data for now. Modalities Screen is implemented we are changing its appearance. Currently Playing screen should be set up with dummy values and feel like a vibe station. Help Screen should be overhauled to document the different modalities. Details Tab should be overhauled to be a composable interface in a full screen.
###

# CODING GUIDELINES
- the code should have high signal to noise ratio
- Add logging to areas will implement business logic and are "higher risk". Please leverage logger.go and make new loggers *if necessary*