package ui

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// adaptive is a shorthand for building a light/dark color pair from hex values.
func adaptive(light, dark string) compat.AdaptiveColor {
	return compat.AdaptiveColor{Light: lipgloss.Color(light), Dark: lipgloss.Color(dark)}
}

// Colors.
var (
	normalDim      = adaptive("#A49FA5", "#777777")
	gray           = adaptive("#909090", "#626262")
	midGray        = adaptive("#B2B2B2", "#4A4A4A")
	darkGray       = adaptive("#DDDADA", "#3C3C3C")
	brightGray     = adaptive("#847A85", "#979797")
	dimBrightGray  = adaptive("#C2B8C2", "#4D4D4D")
	cream          = adaptive("#FFFDF5", "#FFFDF5")
	yellowGreen    = adaptive("#04B575", "#ECFD65")
	fuchsia        = adaptive("#EE6FF8", "#EE6FF8")
	dimFuchsia     = adaptive("#F1A8FF", "#99519E")
	dullFuchsia    = adaptive("#F793FF", "#AD58B4")
	dimDullFuchsia = adaptive("#F6C9FF", "#7B4380")
	green          = lipgloss.Color("#04B575")
	red            = adaptive("#FF4672", "#ED567A")
	semiDimGreen   = adaptive("#35D79C", "#036B46")
	dimGreen       = adaptive("#72D2B0", "#0B5137")
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
	tabStyle         = lipgloss.NewStyle().Foreground(adaptive("#909090", "#626262"))
	selectedTabStyle = lipgloss.NewStyle().Foreground(adaptive("#333333", "#979797"))
	errorTitleStyle  = lipgloss.NewStyle().Foreground(cream).Background(red).Padding(0, 1)
	subtleStyle      = lipgloss.NewStyle().Foreground(adaptive("#9B9B9B", "#5C5C5C"))
	paginationStyle  = subtleStyle
)
