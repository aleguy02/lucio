package main

import (
	"context"
	"log"
	"net"
	"os"

	tea "charm.land/bubbletea/v2"
)

const SOCKET_PATH = "/tmp/spotify-tui.sock"

func startGestureServer(ctx context.Context, ch chan tea.Msg) error {
	os.RemoveAll(SOCKET_PATH)
	sock, err := net.Listen("unix", SOCKET_PATH)
	if err != nil {
		return err
	}
	log.Printf("Listening at %s\n", SOCKET_PATH)
	
	go func() {
		go func() {
			<-ctx.Done()  // note to self: receiving from a channel is blocking
			_ = sock.Close()
		}()

		for {
			conn, err := sock.Accept()
			if err != nil {
				log.Println(err)
				return
			}

			go func() {
				defer func() { _ = conn.Close() }()
				
				cmdStr := SpotifyCommand("PLAY")
				arg := ""
				ch <- SpotifyActionMsg{Command: cmdStr, Arg: arg}
				log.Println("marker inside gesture server code")

				for {
					// validate spotify messages here and emit SpotifyActionMsg
					buf := make([]byte, 5)
					_, err := conn.Read(buf)
					if err != nil {
						log.Println("client disconnected:", err)
						return
					}
				}
			}()
		}
	}()

	// sock.Close()
	return nil
}