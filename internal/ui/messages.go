package ui

import tea "charm.land/bubbletea/v2"

type SwitchViewMsg int

func SwitchViewCmd(view int) tea.Cmd {
	return func() tea.Msg { return SwitchViewMsg(view) }
}

// backToMenuMsg is sent by sub-views (e.g. guide) to return to the menu browsing state.
type backToMenuMsg struct{}

// AgentQueryMsg is emitted by the agent tab when the user submits a chat message.
type AgentQueryMsg struct {
	Text string
}

// AgentChunkMsg carries one streamed token (or the final done signal) from the agent runner.
// Done=true means the turn is complete; Err carries any transport error.
// ToolName is the bare function name (non-empty when the agent invoked a tool); ToolArgs holds its arguments.
type AgentChunkMsg struct {
	Text     string
	ToolName string
	ToolArgs map[string]any
	Done     bool
	Err      error
}

// ToolConfirmationMsg carries the user's confirmation status and function ID
type ToolConfirmationMsg struct {
	ID	string
	Confirmed bool
}