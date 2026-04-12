package main

import (
	"context"
	"log"
	"net"
	"os"
	"strings"

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
				
				buf := make([]byte, 256)
				for {
					n, err := conn.Read(buf)
					if err != nil {
						log.Println("client disconnected:", err)
						return
					}
					raw := strings.TrimSpace(string(buf[:n]))
					if raw == "" {
						continue
					}

					parts := strings.Fields(strings.ToUpper(raw))
					cmd := SpotifyCommand(parts[0])
					arg := ""
					if len(parts) > 1 {
						arg = strings.Join(parts[1:], " ")
					}

					if (cmd == CmdSeekF || cmd == CmdSeekB) && arg == "" {
						arg = "10"
					}

					if !IsValidSpotifyCommand(cmd) {
						log.Printf("gesture server: unknown command %q\n", cmd)
						continue
					}

					ch <- SpotifyActionMsg{Command: cmd, Arg: arg}
				}
			}()
		}
	}()

	// sock.Close()
	return nil
}