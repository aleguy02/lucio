package main

import "log"

// ServiceLogger tags every log line with a service name prefix, making
// log output grep-able by service (e.g. grep '\[gestureServer\]' debug.log).
//
// It delegates to the default log package functions so it always writes to
// whatever output tea.LogToFile (or log.SetOutput) has configured.
type ServiceLogger struct{ prefix string }

func (l ServiceLogger) Printf(format string, v ...any) {
	log.Printf("["+l.prefix+"] "+format, v...)
}

func (l ServiceLogger) Println(v ...any) {
	log.Println(append([]any{"[" + l.prefix + "]"}, v...)...)
}

var (
	TerminalLog = ServiceLogger{prefix: "terminalMode"}
	GestureLog  = ServiceLogger{prefix: "gestureServer"}
)
