package gestures

import tea "charm.land/bubbletea/v2"

type ToggleGesturesMsg bool

func ToggleGesturesCmd(toggle bool) tea.Cmd {
	return func() tea.Msg { return ToggleGesturesMsg(toggle) }
}

// WaitForGestureCmd blocks until the next message arrives on ch.
// Returns nil when ch is nil (gesture modality is off), a no-op for BubbleTea.
func WaitForGestureCmd(ch chan tea.Msg) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg { return <-ch }
}

// GestureClientExitedMsg is delivered when the gesture subprocess exits.
// Ch identifies the session so stale notifications from a previous session can be ignored.
type GestureClientExitedMsg struct {
	Ch  chan tea.Msg
	Err error
}
