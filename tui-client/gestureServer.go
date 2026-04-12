package main

import (
	"context"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"syscall"

	tea "charm.land/bubbletea/v2"
)

const SOCKET_PATH = "/tmp/spotify-tui.sock"

// macOS LOCAL_PEERPID: returns the PID of the process on the other end of a
// Unix-domain socket.  SOL_LOCAL=0, LOCAL_PEERPID=5 (from <sys/un.h>).
const (
	solLocal     = 0
	localPeerPID = 5
)

func peerPID(conn *net.UnixConn) (int, error) {
	raw, err := conn.SyscallConn()
	if err != nil {
		return 0, err
	}
	var pid int
	var innerErr error
	_ = raw.Control(func(fd uintptr) {
		pid, innerErr = syscall.GetsockoptInt(int(fd), solLocal, localPeerPID)
	})
	return pid, innerErr
}

func startGestureServer(ctx context.Context, ch chan tea.Msg) error {
	os.RemoveAll(SOCKET_PATH)
	sock, err := net.Listen("unix", SOCKET_PATH)
	if err != nil {
		return err
	}
	GestureLog.Printf("Listening at %s\n", SOCKET_PATH)

	var clientPID atomic.Int32

	go func() {
		// Shutdown watcher: when the context is cancelled, send SIGTERM to the
		// connected client so it can clean up, then close the listener.
		go func() {
			<-ctx.Done()
			if pid := int(clientPID.Load()); pid != 0 {
				proc, err := os.FindProcess(pid)
				if err == nil {
					if err := proc.Signal(syscall.SIGTERM); err == nil {
						GestureLog.Printf("sent SIGTERM to client PID %d\n", pid)
					}
				}
			}
			_ = sock.Close()
		}()

		for {
			conn, err := sock.Accept()
			if err != nil {
				GestureLog.Println(err)
				return
			}

			unixConn := conn.(*net.UnixConn)
			pid, err := peerPID(unixConn)
			if err != nil {
				GestureLog.Printf("could not read peer PID: %v\n", err)
			} else {
				clientPID.Store(int32(pid))
				GestureLog.Printf("client connected, PID %d\n", pid)
			}

			go func(conn net.Conn, pid int) {
				defer func() {
					_ = conn.Close()
					clientPID.Store(0)
				}()

				buf := make([]byte, 256)
				for {
					n, err := conn.Read(buf)
					if err != nil {
						GestureLog.Println("client disconnected:", err)
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
						GestureLog.Printf("unknown command %q\n", cmd)
						continue
					}

					GestureLog.Printf("command: %q arg: %q\n", cmd, arg)
					ch <- SpotifyActionMsg{Command: cmd, Arg: arg}
				}
			}(conn, pid)
		}
	}()

	return nil
}