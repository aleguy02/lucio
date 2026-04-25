package main

import (
	"fmt"
	"log"
	"os"

	myagent "aleguy02/spotify-tui/internal/agent"
	"aleguy02/spotify-tui/internal/spotify"

	tea "charm.land/bubbletea/v2"
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

	agentRunner, err := myagent.NewRunner("gemma4:e2b", "")
	if err != nil {
		log.Fatal("Agent setup failed: ", err)
	}

	p := tea.NewProgram(newModel(client, agentRunner))
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
