package ui

import (
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/compat"
)

// Colors.
var (
	normalDim      = compat.AdaptiveColor{Light: lipgloss.Color("#A49FA5"), Dark: lipgloss.Color("#777777")}
	gray           = compat.AdaptiveColor{Light: lipgloss.Color("#909090"), Dark: lipgloss.Color("#626262")}
	midGray        = compat.AdaptiveColor{Light: lipgloss.Color("#B2B2B2"), Dark: lipgloss.Color("#4A4A4A")}
	darkGray       = compat.AdaptiveColor{Light: lipgloss.Color("#DDDADA"), Dark: lipgloss.Color("#3C3C3C")}
	brightGray     = compat.AdaptiveColor{Light: lipgloss.Color("#847A85"), Dark: lipgloss.Color("#979797")}
	dimBrightGray  = compat.AdaptiveColor{Light: lipgloss.Color("#C2B8C2"), Dark: lipgloss.Color("#4D4D4D")}
	cream          = compat.AdaptiveColor{Light: lipgloss.Color("#FFFDF5"), Dark: lipgloss.Color("#FFFDF5")}
	yellowGreen    = compat.AdaptiveColor{Light: lipgloss.Color("#04B575"), Dark: lipgloss.Color("#ECFD65")}
	fuchsia        = compat.AdaptiveColor{Light: lipgloss.Color("#EE6FF8"), Dark: lipgloss.Color("#EE6FF8")}
	dimFuchsia     = compat.AdaptiveColor{Light: lipgloss.Color("#F1A8FF"), Dark: lipgloss.Color("#99519E")}
	dullFuchsia    = compat.AdaptiveColor{Dark: lipgloss.Color("#AD58B4"), Light: lipgloss.Color("#F793FF")}
	dimDullFuchsia = compat.AdaptiveColor{Light: lipgloss.Color("#F6C9FF"), Dark: lipgloss.Color("#7B4380")}
	green          = lipgloss.Color("#04B575")
	red            = compat.AdaptiveColor{Light: lipgloss.Color("#FF4672"), Dark: lipgloss.Color("#ED567A")}
	semiDimGreen   = compat.AdaptiveColor{Light: lipgloss.Color("#35D79C"), Dark: lipgloss.Color("#036B46")}
	dimGreen       = compat.AdaptiveColor{Light: lipgloss.Color("#72D2B0"), Dark: lipgloss.Color("#0B5137")}
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
	tabStyle         = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#909090"), Dark: lipgloss.Color("#626262")})
	selectedTabStyle = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#333333"), Dark: lipgloss.Color("#979797")})
	errorTitleStyle  = lipgloss.NewStyle().Foreground(cream).Background(red).Padding(0, 1)
	subtleStyle      = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#9B9B9B"), Dark: lipgloss.Color("#5C5C5C")})
	paginationStyle  = subtleStyle
)
