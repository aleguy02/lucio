package ui

import "log"

// ServiceLogger tags every log line with a service name prefix, making
// log output grep-able by service (e.g. grep '\[terminalMode\]' debug.log).
type ServiceLogger struct{ prefix string }

func (l ServiceLogger) Printf(format string, v ...any) {
	log.Printf("["+l.prefix+"] "+format, v...)
}

func (l ServiceLogger) Println(v ...any) {
	log.Println(append([]any{"[" + l.prefix + "]"}, v...)...)
}

var (
	TerminalLog   = ServiceLogger{prefix: "terminalMode"}
	ModalitiesLog = ServiceLogger{prefix: "modalitiesMenu"}
	NowPlayingLog = ServiceLogger{prefix: "nowPlaying"}
)
