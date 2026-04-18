package main

import "charm.land/lipgloss/v2"

var (
	ColorSpotifyGreen = lipgloss.Color("#1DB954")
	ColorWhite        = lipgloss.Color("#FFFFFF")
	ColorMidGray      = lipgloss.Color("#B3B3B3")
	ColorDarkGray     = lipgloss.Color("#535353")
)

type Theme int

const (
	ThemeDefault    Theme = iota
	ThemeMinimalist
	ThemeVibes
)
