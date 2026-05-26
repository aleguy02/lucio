package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	myagent "aleguy02/spotify-tui/internal/agent"
	"aleguy02/spotify-tui/internal/spotify"

	"google.golang.org/adk/agent"
	"google.golang.org/genai"
)

const (
	userID    = "test-user"
	sessionID = "test-session"
	modelName = "gemma4:e2b"
	ollamaURL = ""
)

func main() {
	client, err := spotify.NewSpotifyClient()
	if err != nil {
		log.Fatal("Spotify setup failed: ", err)
	}

	_, runner, err := myagent.NewRunner(client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create runner: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Agent ready. Type a message and press Enter. Ctrl+C or 'quit' to exit.")
	fmt.Println(strings.Repeat("-", 50))

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "quit" || input == "exit" {
			break
		}

		msg := genai.NewContentFromText(input, "user")
		cfg := agent.RunConfig{StreamingMode: agent.StreamingModeSSE}

		fmt.Print("Lucio: ")
		for event, err := range runner.Run(context.Background(), userID, sessionID, msg, cfg) {
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
				break
			}
			if event == nil || event.Content == nil {
				continue
			}
			for _, part := range event.Content.Parts {
				switch {
				case part.FunctionCall != nil:
					fmt.Printf("\n[tool call] %s(%v)\n", part.FunctionCall.Name, part.FunctionCall.Args)
				case part.FunctionResponse != nil:
					fmt.Printf("[tool result] %s → %v\n", part.FunctionResponse.Name, part.FunctionResponse.Response)
				case part.Text != "" && !part.Thought:
					fmt.Print(part.Text)
				}
			}
		}
		fmt.Println()
	}
}
