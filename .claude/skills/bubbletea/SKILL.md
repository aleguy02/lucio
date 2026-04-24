---
name: bubbletea
description: Browse Bubbletea TUI framework documentation and examples. Use when working with Bubbletea components, models, commands, or building terminal user interfaces in Go.
---

# Bubbletea Documentation

Bubbletea is a Go framework for building terminal user interfaces based on The Elm Architecture.

## Key Resources

When you need to understand Bubbletea patterns or find examples:

1. **Examples README** - Overview of all available examples:
   https://github.com/charmbracelet/bubbletea/blob/main/examples/README.md

2. **Examples Directory** - Full source code for all examples:
   https://github.com/charmbracelet/bubbletea/tree/main/examples

## How to Use

Two tools are available for web access:

- **WebFetch** — fetches a specific URL and processes it with a prompt. Requires both a `url` and a `prompt` describing what to extract. Results may be summarized for large pages. Has a 15-minute cache. **For GitHub URLs, prefer the `gh` CLI via Bash instead** (e.g. `gh api`, `gh browse`).
- **WebSearch** — searches the web by keyword query. Returns links and snippets. Use when you don't have a direct URL or want to discover relevant pages.

1. To browse available Bubbletea examples, use WebFetch on the examples README:
   - URL: `https://raw.githubusercontent.com/charmbracelet/bubbletea/main/examples/README.md`
   - Prompt: "List all available examples with their descriptions"

2. To read a specific example's source, use WebFetch on its raw URL:
   - URL pattern: `https://raw.githubusercontent.com/charmbracelet/bubbletea/main/examples/<name>/main.go`
   - Prompt: "Show me the full source code"

3. To find documentation or patterns not reachable by direct URL, use WebSearch with a keyword query.

## Common Examples to Reference

- `list` - List component with filtering
- `table` - Table component
- `textinput` - Text input handling
- `textarea` - Multi-line text input
- `viewport` - Scrollable content
- `paginator` - Pagination
- `spinner` - Loading spinners
- `progress` - Progress bars
- `tabs` - Tab navigation
- `help` - Help text/keybindings display

## Core Concepts

- **Model**: Application state
- **Update**: Handles messages and returns updated model + commands
- **View**: Renders the model to a string
- **Cmd**: Side effects that produce messages
- **Msg**: Events that trigger updates

## Related Charm Libraries

- **Bubbles**: Pre-built components (github.com/charmbracelet/bubbles)
- **Lipgloss**: Styling and layout (github.com/charmbracelet/lipgloss)
- **Glamour**: Markdown rendering (github.com/charmbracelet/glamour)