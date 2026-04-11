package main

import (
	"context"
	"log"
	"net"
)

const SOCKET_PATH = "/tmp/spotify-tui.sock"

func startGestureServer(ctx context.Context) error {
	log.Printf("Listening at %s. Press Ctrl+C to exit...\n", SOCKET_PATH)
	sock, err := net.Listen("unix", SOCKET_PATH)
	if err != nil {
		return err
	}

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

				for {
					buf := make([]byte, 5)
					_, err := conn.Read(buf)
					if err != nil {
						log.Fatal(err)
					}
					log.Printf("from gesture server: ")
					log.Println(string(buf))
				}
			}()
		}
	}()

	// sock.Close()
	return nil
}