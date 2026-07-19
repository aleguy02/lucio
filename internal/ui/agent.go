package ui

import (
	"fmt"
	"math/rand/v2"
	"sort"
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

type ToolCall struct {
	Name string
	Args map[string]any
}

// Message is the base chat unit.
type Message struct {
	Sender    MessageSender
	Content   string
	ToolCalls []ToolCall
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
	flameOrangeStyle   = lipgloss.NewStyle().Foreground(ColorFlameOrange)
	flameYellowStyle   = lipgloss.NewStyle().Foreground(ColorFlameYellow)
	flameRedStyle      = lipgloss.NewStyle().Foreground(ColorFlameRed)
	flameWhiteStyle    = lipgloss.NewStyle().Foreground(ColorFlameWhite)
	flameEyesStyle     = lipgloss.NewStyle().Foreground(lipgloss.Black)
)

type agentChatModel struct {
	messages              []Message
	streamAccumulator     string
	streamToolCalls       []ToolCall
	pendingConfirmationID string
	pendingToolName       string
	pendingToolArgs       map[string]any
	isResponding          bool
	viewport              viewport.Model
	input                 textinput.Model
	keys                  agentChatKeyMap
	width                 int
	height                int
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
			content = "thinking..."
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
		for _, tc := range msg.ToolCalls {
			out += "\n" + toolCallStyle.Render("+ "+formatToolCall(tc.Name, tc.Args))
		}
		out += "\n" + agentMsgTextStyle.Render(msg.Content)
		return out
	default:
		label := userMsgLabelStyle.Render("you")
		return label + "\n" + userMsgTextStyle.Render(msg.Content)
	}
}

// formatToolCall renders a tool invocation as "name(key: val, ...)", or just "name" when
// it takes no arguments. Arguments are sorted for stable output.
func formatToolCall(name string, args map[string]any) string {
	if len(args) == 0 {
		return name
	}
	keys := make([]string, 0, len(args))
	for k := range args {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %v", k, args[k]))
	}
	return name + "(" + strings.Join(parts, ", ") + ")"
}

var (
	confirmBlurbStyle = lipgloss.NewStyle().Foreground(ColorLightYellow).Bold(true)
	confirmHintStyle  = lipgloss.NewStyle().Foreground(ColorMidGray).Faint(true)
)

// renderConfirmationPrompt is the human-in-the-loop prompt shown in the dataFrame when the
// agent requests permission to run a tool. width is the available content width for wrapping.
func renderConfirmationPrompt(name string, args map[string]any, width int) string {
	wrap := lipgloss.NewStyle().Width(max(1, width)).Align(lipgloss.Center)
	blurb := wrap.Inherit(confirmBlurbStyle).Render("Lucio is requesting your permission to run the following tool. This tool may be destructive and irreversible.")
	tool := wrap.Inherit(toolCallStyle).Render(formatToolCall(name, args))
	hint := wrap.Inherit(confirmHintStyle).Render("[y] confirm  ·  [any other key] reject")
	return lipgloss.JoinVertical(lipgloss.Center, blurb, "", tool, "", hint)
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
			if msg.ConfirmRequired {
				// ADK forwards the to-be-confirmed call as a normal event immediately before its
				// confirmation wrapper, so it's the tail of streamToolCalls. Drop it (matched by
				// name) so it isn't committed now: it's shown in the dataFrame prompt and re-added
				// to the chat only on accept. Earlier tools from this same run are left intact.
				if n := len(c.streamToolCalls); n > 0 && c.streamToolCalls[n-1].Name == msg.ToolName {
					c.streamToolCalls = c.streamToolCalls[:n-1]
				}
				// Hold the request; the user resolves it via keypress once the stream's
				// Done lands (the !isResponding guard in the keypress handler enforces this).
				c.pendingConfirmationID = msg.ToolID
				c.pendingToolName = msg.ToolName
				c.pendingToolArgs = msg.ToolArgs
			} else if msg.ToolName != "" {
				c.streamToolCalls = append(c.streamToolCalls, ToolCall{Name: msg.ToolName, Args: msg.ToolArgs})
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
		// Resolve a pending tool confirmation: y approves, any other key rejects.
		// The !isResponding guard ensures the request's stream has finished (Done) before
		// we fire a resume, so we never start a second r.Run mid-turn.
		if c.pendingConfirmationID != "" && !c.isResponding {
			id := c.pendingConfirmationID
			confirmed := strings.EqualFold(msg.String(), "y")
			c.pendingConfirmationID = ""
			if confirmed {
				c.streamToolCalls = append(c.streamToolCalls, ToolCall{Name: c.pendingToolName, Args: c.pendingToolArgs})
			}
			c.pendingToolName, c.pendingToolArgs = "", nil
			c.isResponding = true // resume stream is starting
			c.viewport.SetContent(c.renderMessages())
			c.viewport.GotoBottom()
			return c, func() tea.Msg { return ToolConfirmationMsg{ID: id, Confirmed: confirmed} }
		}

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
			if c.isResponding {
				return c, nil
			}
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

	// Center the content inside the data frame's inner area (border subtracts 2 each axis).
	// While a tool confirmation is pending, the frame shows the prompt instead of the sprite.
	innerW := max(0, rightW-2)
	innerH := max(0, t.height-2)
	content := dataFrameSprite()
	if t.chat.pendingConfirmationID != "" {
		content = renderConfirmationPrompt(t.chat.pendingToolName, t.chat.pendingToolArgs, innerW)
	}
	placed := lipgloss.Place(innerW, innerH, lipgloss.Center, lipgloss.Center, content)
	dataFrame := dataFrameStyle.Width(rightW).Height(t.height).Render(placed)

	return lipgloss.JoinHorizontal(lipgloss.Top, chatFrame, dataFrame)
}

// dataFrameSprite renders one frame of a self-contained flame "simulation". There is no
// external tick, so a frame only advances when the view re-renders; motion is therefore
// coarse and frames are independent (no temporal coherence) — an accepted tradeoff.
//
// The model is a static heat field: a teardrop that is hottest at the base and cools toward
// a narrow, flickering tip. Each frame we perturb every cell with random noise (stronger
// higher up, where real flames dance most), then map the resulting heat through a hot→cool
// gradient of glyphs and colors.
func dataFrameSprite() string {
	// heat is each cell's resting intensity (0..1): hottest low-center, tapering up and out.
	heat := [][]float64{
		{0.00, 0.00, 0.00, 0.18, 0.28, 0.18, 0.00, 0.00, 0.00},
		{0.00, 0.10, 0.32, 0.68, 0.72, 0.68, 0.32, 0.10, 0.00},
		{0.40, 0.42, 0.82, 0.92, 0.96, 0.92, 0.82, 0.26, 0.40},
		{0.50, 0.80, 1.00, 0.96, 0.99, 0.96, 1.00, 0.80, 0.50},
		{0.40, 0.65, 0.82, 0.82, 0.92, 0.82, 0.72, 0.49, 0.40},
	}

	n := len(heat)
	var sb strings.Builder
	for i, row := range heat {
		// Cells flicker more near the tip (top) and barely at the stable base (bottom).
		flicker := 0.45 * (1 - float64(i)/float64(n-1))
		for _, base := range row {
			if base <= 0 {
				sb.WriteByte(' ')
				continue
			}
			if base == 1.00 {
				sb.WriteString(flameEyesStyle.Render("█"))
				continue
			}

			h := base + (rand.Float64()*2-1)*flicker
			switch {
			case h >= 0.88:
				sb.WriteString(flameWhiteStyle.Render("█"))
			case h >= 0.66:
				sb.WriteString(flameYellowStyle.Render("█"))
			case h >= 0.44:
				sb.WriteString(flameOrangeStyle.Render("█"))
			case h >= 0.26:
				sb.WriteString(flameOrangeStyle.Render("▓"))
			case h >= 0.12:
				sb.WriteString(flameRedStyle.Render("▒"))
			case h > 0.02:
				sb.WriteString(flameRedStyle.Render("░"))
			default:
				sb.WriteByte(' ')
			}
		}
		if i < n-1 {
			sb.WriteByte('\n')
		}
	}
	sb.WriteString("\n")
	sb.WriteString(flameOrangeStyle.Render(" ▀▀▀▀▀▀▀ "))
	return sb.String()
}
