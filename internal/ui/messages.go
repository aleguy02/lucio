package ui

import tea "charm.land/bubbletea/v2"

type SwitchViewMsg int

func SwitchViewCmd(view int) tea.Cmd {
	return func() tea.Msg { return SwitchViewMsg(view) }
}

// backToMenuMsg is sent by sub-views (e.g. guide) to return to the menu browsing state.
type backToMenuMsg struct{}
