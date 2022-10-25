package main

import . "github.com/charmbracelet/lipgloss" //nolint:revive

var (
	keyword = NewStyle().
		Foreground(AdaptiveColor{Light: "#04B575", Dark: "#04B575"}).
		Render

	paragraph = NewStyle().
			Width(78).
			Padding(0, 0, 0, 2).
			Render
)
