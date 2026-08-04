package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// lightDark resolves light/dark color pairs based on terminal background.
// It is initialized with a default dark background and updated when
// tea.BackgroundColorMsg is received.
var lightDark = lipgloss.LightDark(true)

// adaptive returns a color appropriate for the current terminal background.
func adaptive(light, dark string) color.Color {
	return lightDark(lipgloss.Color(light), lipgloss.Color(dark))
}

// initStyles rebuilds all styles that depend on adaptive colors.
// Call this after updating lightDark.
func initStyles() {
	// Colors
	normalDim = adaptive("#A49FA5", "#777777")
	gray = adaptive("#909090", "#626262")
	midGray = adaptive("#B2B2B2", "#4A4A4A")
	darkGray = adaptive("#DDDADA", "#3C3C3C")
	brightGray = adaptive("#847A85", "#979797")
	dimBrightGray = adaptive("#C2B8C2", "#4D4D4D")
	cream = adaptive("#FFFDF5", "#FFFDF5")
	yellowGreen = adaptive("#04B575", "#ECFD65")
	fuchsia = adaptive("#EE6FF8", "#EE6FF8")
	dimFuchsia = adaptive("#F1A8FF", "#99519E")
	dullFuchsia = adaptive("#F793FF", "#AD58B4")
	dimDullFuchsia = adaptive("#F6C9FF", "#7B4380")
	green = lipgloss.Color("#04B575")
	red = adaptive("#FF4672", "#ED567A")
	semiDimGreen = adaptive("#35D79C", "#036B46")
	dimGreen = adaptive("#72D2B0", "#0B5137")

	// Pager colors
	mintGreen = adaptive("#89F0CB", "#89F0CB")
	darkGreenAdaptive = adaptive("#1C8760", "#1C8760")
	lineNumberFg = adaptive("#656565", "#7D7D7D")
	statusBarNoteFg = adaptive("#656565", "#7D7D7D")
	statusBarBg = adaptive("#E6E6E6", "#242424")

	// Render-func styles
	dimNormalFg = lipgloss.NewStyle().Foreground(normalDim).Render
	brightGrayFg = lipgloss.NewStyle().Foreground(brightGray).Render
	dimBrightGrayFg = lipgloss.NewStyle().Foreground(dimBrightGray).Render
	grayFg = lipgloss.NewStyle().Foreground(gray).Render
	midGrayFg = lipgloss.NewStyle().Foreground(midGray).Render
	darkGrayFg = lipgloss.NewStyle().Foreground(darkGray)
	greenFg = lipgloss.NewStyle().Foreground(green).Render
	semiDimGreenFg = lipgloss.NewStyle().Foreground(semiDimGreen).Render
	dimGreenFg = lipgloss.NewStyle().Foreground(dimGreen).Render
	fuchsiaFg = lipgloss.NewStyle().Foreground(fuchsia).Render
	dimFuchsiaFg = lipgloss.NewStyle().Foreground(dimFuchsia).Render
	dullFuchsiaFg = lipgloss.NewStyle().Foreground(dullFuchsia).Render
	dimDullFuchsiaFg = lipgloss.NewStyle().Foreground(dimDullFuchsia).Render
	redFg = lipgloss.NewStyle().Foreground(red).Render

	// Named styles
	tabStyle = lipgloss.NewStyle().Foreground(adaptive("#909090", "#626262"))
	selectedTabStyle = lipgloss.NewStyle().Foreground(adaptive("#333333", "#979797"))
	errorTitleStyle = lipgloss.NewStyle().Foreground(cream).Background(red).Padding(0, 1)
	subtleStyle = lipgloss.NewStyle().Foreground(adaptive("#9B9B9B", "#5C5C5C"))
	paginationStyle = subtleStyle

	// Pager styles
	statusBarScrollPosStyle = lipgloss.NewStyle().
					Foreground(adaptive("#949494", "#5A5A5A")).
					Background(statusBarBg).
					Render

	statusBarNoteStyle = lipgloss.NewStyle().
				Foreground(statusBarNoteFg).
				Background(statusBarBg).
				Render

	statusBarHelpStyle = lipgloss.NewStyle().
				Foreground(statusBarNoteFg).
				Background(adaptive("#DCDCDC", "#323232")).
				Render

	statusBarMessageStyle = lipgloss.NewStyle().
				Foreground(mintGreen).
				Background(darkGreenAdaptive).
				Render

	statusBarMessageScrollPosStyle = lipgloss.NewStyle().
						Foreground(mintGreen).
						Background(darkGreenAdaptive).
						Render

	statusBarMessageHelpStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#B6FFE4")).
					Background(green).
					Render

	helpViewStyle = lipgloss.NewStyle().
			Foreground(statusBarNoteFg).
			Background(adaptive("#f2f2f2", "#1B1B1B")).
			Render

	lineNumberStyle = lipgloss.NewStyle().
			Foreground(lineNumberFg).
			Render

	// Stash styles
	dividerDot = darkGrayFg.SetString(" • ")
	dividerBar = darkGrayFg.SetString(" │ ")

	logoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ECFD65")).
			Background(fuchsia).
			Bold(true)

	stashSpinnerStyle = lipgloss.NewStyle().
				Foreground(gray)
	stashInputPromptStyle = lipgloss.NewStyle().
				Foreground(yellowGreen).
				MarginRight(1)
}

// Colors.
var (
	normalDim      color.Color
	gray           color.Color
	midGray        color.Color
	darkGray       color.Color
	brightGray     color.Color
	dimBrightGray  color.Color
	cream          color.Color
	yellowGreen    color.Color
	fuchsia        color.Color
	dimFuchsia     color.Color
	dullFuchsia    color.Color
	dimDullFuchsia color.Color
	green          color.Color
	red            color.Color
	semiDimGreen   color.Color
	dimGreen       color.Color
)

// Pager colors.
var (
	mintGreen       color.Color
	darkGreenAdaptive color.Color
	lineNumberFg    color.Color
	statusBarNoteFg color.Color
	statusBarBg     color.Color
)

// Render-func styles.
var (
	dimNormalFg      func(...string) string
	brightGrayFg     func(...string) string
	dimBrightGrayFg  func(...string) string
	grayFg           func(...string) string
	midGrayFg        func(...string) string
	darkGrayFg       lipgloss.Style
	greenFg          func(...string) string
	semiDimGreenFg   func(...string) string
	dimGreenFg       func(...string) string
	fuchsiaFg        func(...string) string
	dimFuchsiaFg     func(...string) string
	dullFuchsiaFg    func(...string) string
	dimDullFuchsiaFg func(...string) string
	redFg            func(...string) string
)

// Named styles.
var (
	tabStyle         lipgloss.Style
	selectedTabStyle lipgloss.Style
	errorTitleStyle  lipgloss.Style
	subtleStyle      lipgloss.Style
	paginationStyle  lipgloss.Style
)

// Pager styles.
var (
	statusBarScrollPosStyle        func(...string) string
	statusBarNoteStyle             func(...string) string
	statusBarHelpStyle             func(...string) string
	statusBarMessageStyle          func(...string) string
	statusBarMessageScrollPosStyle func(...string) string
	statusBarMessageHelpStyle      func(...string) string
	helpViewStyle                  func(...string) string
	lineNumberStyle                func(...string) string
)

// Stash styles.
var (
	dividerDot            lipgloss.Style
	dividerBar            lipgloss.Style
	logoStyle             lipgloss.Style
	stashSpinnerStyle     lipgloss.Style
	stashInputPromptStyle lipgloss.Style
)
