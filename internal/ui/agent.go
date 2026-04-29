package ui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// MessageSender identifies the author of a chat message.
type MessageSender int

const (
	SenderUser MessageSender = iota
	SenderAgent
)

// Message is the base chat unit.
type Message struct {
	Sender    MessageSender
	Content   string
	ToolCalls []string
}

const maxChatMessages = 15

type agentChatKeyMap struct {
	ScrollUp   key.Binding
	ScrollDown key.Binding
	Send       key.Binding
}

func (k agentChatKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Send, k.ScrollUp, k.ScrollDown}
}

func (k agentChatKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Send, k.ScrollUp, k.ScrollDown}}
}

func defaultAgentChatKeyMap() agentChatKeyMap {
	return agentChatKeyMap{
		ScrollUp: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("up", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("down", "scroll down"),
		),
		Send: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send"),
		),
	}
}

var (
	agentFrameStyle = lipgloss.NewStyle().Border(lipgloss.HiddenBorder())
	dataFrameStyle  = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(ColorWhite)

	chatInputBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(ColorDarkGray).
				Padding(0, 1)

	userMsgLabelStyle  = lipgloss.NewStyle().Foreground(ColorMidGray).Faint(true)
	userMsgTextStyle   = lipgloss.NewStyle().Foreground(ColorLightGrey)
	agentMsgLabelStyle = lipgloss.NewStyle().Foreground(ColorSpotifyGreen).Bold(true)
	agentMsgTextStyle  = lipgloss.NewStyle().Foreground(ColorWhite)
	toolCallStyle      = lipgloss.NewStyle().Foreground(ColorLightYellow)
)

type agentChatModel struct {
	messages          []Message
	streamAccumulator string
	streamToolCalls   []string
	isResponding      bool
	viewport          viewport.Model
	input             textinput.Model
	keys              agentChatKeyMap
	width             int
	height            int
}

func newAgentChatModel() agentChatModel {
	ti := textinput.New()
	// TODO(polish): add bank of random placeholders like this
	ti.Placeholder = "hey lucio, play my favorite song..."
	ti.SetStyles(ti.Styles())
	ti.Focus()

	vp := viewport.New()
	vp.SoftWrap = true

	return agentChatModel{
		input:    ti,
		viewport: vp,
		keys:     defaultAgentChatKeyMap(),
	}
}

// chatInputBoxHeight = 1 text line + 2 border lines (top + bottom).
const chatInputBoxHeight = 3

func (c *agentChatModel) resize() {
	vpH := max(1, c.height-chatInputBoxHeight)
	c.viewport.SetHeight(vpH)
	c.viewport.SetWidth(c.width)
	// outer width is c.width; border(-2) and padding(-2) give content width of c.width-4
	promptW := lipgloss.Width(c.input.Prompt)
	c.input.SetWidth(max(1, c.width-5-promptW))
	c.viewport.SetContent(c.renderMessages())
}

func (c agentChatModel) renderMessages() string {
	lines := make([]string, 0, len(c.messages)+1)
	for _, msg := range c.messages {
		lines = append(lines, renderMessage(msg))
	}
	if c.isResponding {
		content := c.streamAccumulator
		if content == "" && len(c.streamToolCalls) == 0 {
			content = "..."
		}
		lines = append(lines, renderMessage(Message{Sender: SenderAgent, Content: content, ToolCalls: c.streamToolCalls}))
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func renderMessage(msg Message) string {
	switch msg.Sender {
	case SenderAgent:
		label := agentMsgLabelStyle.Render("lucio")
		out := label
		for _, name := range msg.ToolCalls {
			out += "\n" + toolCallStyle.Render("+ "+name)
		}
		out += "\n" + agentMsgTextStyle.Render(msg.Content)
		return out
	default:
		label := userMsgLabelStyle.Render("you")
		return label + "\n" + userMsgTextStyle.Render(msg.Content)
	}
}

func (c agentChatModel) Update(msg tea.Msg) (agentChatModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height
		c.resize()
		return c, nil

	case AgentChunkMsg:
		if msg.Err != nil {
			c.messages = append(c.messages, Message{Sender: SenderAgent, Content: "error: " + msg.Err.Error()})
			if len(c.messages) > maxChatMessages {
				c.messages = c.messages[len(c.messages)-maxChatMessages:]
			}
			c.streamAccumulator = ""
			c.streamToolCalls = nil
			c.isResponding = false
		} else {
			if msg.ToolName != "" {
				c.streamToolCalls = append(c.streamToolCalls, msg.ToolName)
			}
			if !msg.Done {
				c.streamAccumulator += msg.Text
			}
			if msg.Done {
				if c.streamAccumulator != "" || len(c.streamToolCalls) > 0 {
					c.messages = append(c.messages, Message{
						Sender:    SenderAgent,
						Content:   c.streamAccumulator,
						ToolCalls: c.streamToolCalls,
					})
					if len(c.messages) > maxChatMessages {
						c.messages = c.messages[len(c.messages)-maxChatMessages:]
					}
				}
				c.streamAccumulator = ""
				c.streamToolCalls = nil
				c.isResponding = false
			}
		}
		c.viewport.SetContent(c.renderMessages())
		c.viewport.GotoBottom()
		return c, nil

	case tea.PasteMsg:
		c.input, cmd = c.input.Update(msg)
		return c, cmd

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, c.keys.Send):
			text := strings.TrimSpace(c.input.Value())
			if text == "" || c.isResponding {
				return c, nil
			}
			c.messages = append(c.messages, Message{Sender: SenderUser, Content: text})
			if len(c.messages) > maxChatMessages {
				c.messages = c.messages[len(c.messages)-maxChatMessages:]
			}
			c.input.SetValue("")
			c.isResponding = true
			c.viewport.SetContent(c.renderMessages())
			c.viewport.GotoBottom()
			return c, func() tea.Msg { return AgentQueryMsg{Text: text} }

		case key.Matches(msg, c.keys.ScrollUp), key.Matches(msg, c.keys.ScrollDown):
			c.viewport, cmd = c.viewport.Update(msg)
			return c, cmd

		default:
			c.input, cmd = c.input.Update(msg)
			return c, cmd
		}
	}

	return c, nil
}

func (c agentChatModel) View() string {
	// outer width = c.width; input box border+padding account for the -4
	inputBox := chatInputBoxStyle.Width(c.width).Render(c.input.View())
	return lipgloss.JoinVertical(lipgloss.Left,
		c.viewport.View(),
		inputBox,
	)
}

// TODO(feat): implement data visualization frame on the right
type agentTabModel struct {
	chat   agentChatModel
	width  int
	height int
}

func newAgentTabModel() agentTabModel {
	return agentTabModel{chat: newAgentChatModel()}
}

func (t agentTabModel) Update(msg tea.Msg) (agentTabModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		halfW := msg.Width / 2

		chatMsg := tea.WindowSizeMsg{
			Width:  max(1, halfW-2), // frame borders are 1 each side
			Height: max(1, msg.Height-2),
		}
		var cmd tea.Cmd
		t.chat, cmd = t.chat.Update(chatMsg)
		return t, cmd

	case tea.KeyPressMsg, tea.PasteMsg:
		var cmd tea.Cmd
		t.chat, cmd = t.chat.Update(msg)
		return t, cmd

	case AgentChunkMsg:
		var cmd tea.Cmd
		t.chat, cmd = t.chat.Update(msg)
		return t, cmd
	}

	return t, nil
}

func (t agentTabModel) View() string {
	halfW := t.width / 2
	rightW := t.width - halfW

	// Width/Height set the outer dimensions (including border) in lipgloss v2
	chatFrame := agentFrameStyle.Width(halfW).Height(t.height).Render(t.chat.View())

	// Center the sprite inside the data frame's inner content area (border subtracts 2 each axis).
	innerW := max(0, rightW-2)
	innerH := max(0, t.height-2)
	placed := lipgloss.Place(innerW, innerH, lipgloss.Center, lipgloss.Center, dataFrameSprite())
	dataFrame := dataFrameStyle.Width(rightW).Height(t.height).Render(placed)

	return lipgloss.JoinHorizontal(lipgloss.Top, chatFrame, dataFrame)
}

func dataFrameSprite() string {
	row1 := "   ▄▄░▄▄▒"
	row2 := " ██████▌ "
	row3 := "▐██████▌ "
	row4 := " ▀▀▀▀▀▀ "

	// TODO(polish): add colors
	return strings.Join([]string{row1, row2, row3, row4}, "\n")
}
