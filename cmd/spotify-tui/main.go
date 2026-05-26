package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	myagent "aleguy02/spotify-tui/internal/agent"
	"aleguy02/spotify-tui/internal/spotify"

	tea "charm.land/bubbletea/v2"
)

func main() {
	f, err := tea.LogToFile(filepath.Join("logs", "tui.log"), "")
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

	sessionService, agentRunner, err := myagent.NewRunner(client)
	if err != nil {
		log.Fatal("Agent setup failed: ", err)
	}

	p := tea.NewProgram(newModel(client, sessionService, agentRunner))
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
