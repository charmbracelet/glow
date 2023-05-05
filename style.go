package main

import . "github.com/charmbracelet/lipgloss" //nolint:revive

var (
	keyword = NewStyle().
		Foreground(Color("#04B575")).
		Render

	paragraph = NewStyle().
			Width(78).
			Padding(0, 0, 0, 2).
			Render
)
