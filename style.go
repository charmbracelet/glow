package main

import "charm.land/lipgloss/v2"

var (
	keyword = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Render

	paragraph = lipgloss.NewStyle().
			Width(78).
			Padding(0, 0, 0, 2).
			Render
)
