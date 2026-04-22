package main

import (
	"fmt"
	"log"
	"os"

	tea "charm.land/bubbletea/v2"
	"aleguy02/spotify-tui/internal/spotify"
)

func main() {
	f, err := tea.LogToFile("debug.log", "")
	if err != nil {
		fmt.Println("fatal:", err)
		os.Exit(1)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("failed to close file: %v", err)
		}
	}()

	client, err := spotify.NewSpotifyClient()
	if err != nil {
		log.Fatal("Spotify setup failed: ", err)
	}

	p := tea.NewProgram(newModel(client))
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
