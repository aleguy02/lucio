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
type AgentChunkMsg struct {
	Text string
	// ToolName is the bare function name; non-empty when the agent invoked a tool.
	ToolName string
	// ToolArgs holds the invoked tool's arguments.
	ToolArgs map[string]any
	ToolID   string
	// ConfirmRequired=true marks an unwrapped tool-confirmation request: ToolName/ToolArgs
	// hold the original tool's name/args and ToolID holds the adk_request_confirmation id to
	// echo back in the FunctionResponse.
	ConfirmRequired bool
	// Done=true means the current r.Run stream ended; it is transport-level only and does NOT
	// imply the agent has nothing left to do (e.g. a run can end while paused awaiting a
	// confirmation response).
	Done bool
	// Err carries any transport error.
	Err error
}

// ToolConfirmationMsg carries the user's confirmation status and function ID
type ToolConfirmationMsg struct {
	ID        string
	Confirmed bool
}
