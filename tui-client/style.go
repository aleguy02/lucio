package main

import "charm.land/lipgloss/v2"

var (
	ColorSpotifyGreen = lipgloss.Color("#1DB954")
)

type Theme int

const (
	ThemeDefault    Theme = iota
	ThemeMinimalist
	ThemeVibes
)
