package gestures

import (
	"context"
	"log"
	"net"
	"os"
	"strings"

	sp "aleguy02/spotify-tui/internal/spotify"
	tea "charm.land/bubbletea/v2"
)

const SocketPath = "/tmp/spotify-tui.sock"

func StartGestureServer(ctx context.Context, ch chan tea.Msg) error {
	if err := os.RemoveAll(SocketPath); err != nil {
		return err
	}
	sock, err := net.Listen("unix", SocketPath)
	if err != nil {
		return err
	}
	log.Printf("[gestureServer] listening at %s\n", SocketPath)

	go func() {
		go func() {
			<-ctx.Done()
			_ = sock.Close()
		}()

		for {
			conn, err := sock.Accept()
			if err != nil {
				log.Println("[gestureServer]", err)
				return
			}
			log.Println("[gestureServer] client connected")
			go handleConn(conn, ch)
		}
	}()

	return nil
}

func handleConn(conn net.Conn, ch chan tea.Msg) {
	defer func() { _ = conn.Close() }()

	buf := make([]byte, 256)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Println("[gestureServer] client disconnected:", err)
			return
		}
		raw := strings.TrimSpace(string(buf[:n]))
		if raw == "" {
			continue
		}

		parts := strings.Fields(strings.ToUpper(raw))
		cmd := sp.SpotifyCommand(parts[0])
		arg := ""
		if len(parts) > 1 {
			arg = strings.Join(parts[1:], " ")
		}

		if (cmd == sp.CmdSeekF || cmd == sp.CmdSeekB) && arg == "" {
			arg = "10"
		}

		if !sp.IsValidSpotifyCommand(cmd) {
			log.Printf("[gestureServer] unknown command %q\n", cmd)
			continue
		}

		log.Printf("[gestureServer] command: %q arg: %q\n", cmd, arg)
		ch <- sp.SpotifyActionMsg{Command: cmd, Arg: arg}
	}
}
