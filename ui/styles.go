package ui

import "github.com/charmbracelet/lipgloss"

// Colors.
var (
	normalDim      = lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}
	gray           = lipgloss.AdaptiveColor{Light: "#909090", Dark: "#626262"}
	midGray        = lipgloss.AdaptiveColor{Light: "#B2B2B2", Dark: "#4A4A4A"}
	darkGray       = lipgloss.AdaptiveColor{Light: "#DDDADA", Dark: "#3C3C3C"}
	brightGray     = lipgloss.AdaptiveColor{Light: "#847A85", Dark: "#979797"}
	dimBrightGray  = lipgloss.AdaptiveColor{Light: "#C2B8C2", Dark: "#4D4D4D"}
	cream          = lipgloss.AdaptiveColor{Light: "#FFFDF5", Dark: "#FFFDF5"}
	yellowGreen    = lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#ECFD65"}
	fuchsia        = lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"}
	dimFuchsia     = lipgloss.AdaptiveColor{Light: "#F1A8FF", Dark: "#99519E"}
	dullFuchsia    = lipgloss.AdaptiveColor{Dark: "#AD58B4", Light: "#F793FF"}
	dimDullFuchsia = lipgloss.AdaptiveColor{Light: "#F6C9FF", Dark: "#7B4380"}
	green          = lipgloss.Color("#04B575")
	red            = lipgloss.AdaptiveColor{Light: "#FF4672", Dark: "#ED567A"}
	semiDimGreen   = lipgloss.AdaptiveColor{Light: "#35D79C", Dark: "#036B46"}
	dimGreen       = lipgloss.AdaptiveColor{Light: "#72D2B0", Dark: "#0B5137"}
)

// Ulimately, we'll transition to named styles.
var (
	dimNormalFg      = lipgloss.NewStyle().Foreground(normalDim).Render
	brightGrayFg     = lipgloss.NewStyle().Foreground(brightGray).Render
	dimBrightGrayFg  = lipgloss.NewStyle().Foreground(dimBrightGray).Render
	grayFg           = lipgloss.NewStyle().Foreground(gray).Render
	midGrayFg        = lipgloss.NewStyle().Foreground(midGray).Render
	darkGrayFg       = lipgloss.NewStyle().Foreground(darkGray)
	greenFg          = lipgloss.NewStyle().Foreground(green).Render
	semiDimGreenFg   = lipgloss.NewStyle().Foreground(semiDimGreen).Render
	dimGreenFg       = lipgloss.NewStyle().Foreground(dimGreen).Render
	fuchsiaFg        = lipgloss.NewStyle().Foreground(fuchsia).Render
	dimFuchsiaFg     = lipgloss.NewStyle().Foreground(dimFuchsia).Render
	dullFuchsiaFg    = lipgloss.NewStyle().Foreground(dullFuchsia).Render
	dimDullFuchsiaFg = lipgloss.NewStyle().Foreground(dimDullFuchsia).Render
	redFg            = lipgloss.NewStyle().Foreground(red).Render
	tabStyle         = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#909090", Dark: "#626262"})
	selectedTabStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#333333", Dark: "#979797"})
	errorTitleStyle  = lipgloss.NewStyle().Foreground(cream).Background(red).Padding(0, 1)
	subtleStyle      = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"})
	paginationStyle  = subtleStyle
)
