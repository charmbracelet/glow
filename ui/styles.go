package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Styles contains all styles used in the UI. They are resolved from light and
// dark color variants according to the terminal's background color, which
// Bubble Tea reports via tea.BackgroundColorMsg. Until then, we default to
// a dark background.
type Styles struct {
	lightDark lipgloss.LightDarkFunc

	fuchsia color.Color

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

	tabStyle         lipgloss.Style
	selectedTabStyle lipgloss.Style
	errorTitleStyle  lipgloss.Style
	subtleStyle      lipgloss.Style
	paginationStyle  lipgloss.Style

	statusBarScrollPosStyle        func(...string) string
	statusBarNoteStyle             func(...string) string
	statusBarHelpStyle             func(...string) string
	statusBarMessageStyle          func(...string) string
	statusBarMessageScrollPosStyle func(...string) string
	statusBarMessageHelpStyle      func(...string) string
	helpViewStyle                  func(...string) string
	lineNumberStyle                func(...string) string

	dividerDot        lipgloss.Style
	dividerBar        lipgloss.Style
	logoStyle         lipgloss.Style
	stashSpinnerStyle lipgloss.Style
	inputPromptStyle  lipgloss.Style

	searchHighlightStyle         lipgloss.Style
	searchSelectedHighlightStyle lipgloss.Style
}

// newStyles builds all styles for the given terminal background.
func newStyles(isDark bool) Styles {
	s := Styles{lightDark: lipgloss.LightDark(isDark)}

	// Colors
	normalDim := s.adaptive("#A49FA5", "#777777")
	gray := s.adaptive("#909090", "#626262")
	midGray := s.adaptive("#B2B2B2", "#4A4A4A")
	darkGray := s.adaptive("#DDDADA", "#3C3C3C")
	brightGray := s.adaptive("#847A85", "#979797")
	dimBrightGray := s.adaptive("#C2B8C2", "#4D4D4D")
	cream := s.adaptive("#FFFDF5", "#FFFDF5")
	yellowGreen := s.adaptive("#04B575", "#ECFD65")
	s.fuchsia = s.adaptive("#EE6FF8", "#EE6FF8")
	dimFuchsia := s.adaptive("#F1A8FF", "#99519E")
	dullFuchsia := s.adaptive("#F793FF", "#AD58B4")
	dimDullFuchsia := s.adaptive("#F6C9FF", "#7B4380")
	green := lipgloss.Color("#04B575")
	red := s.adaptive("#FF4672", "#ED567A")
	semiDimGreen := s.adaptive("#35D79C", "#036B46")
	dimGreen := s.adaptive("#72D2B0", "#0B5137")

	// Pager colors
	mintGreen := s.adaptive("#89F0CB", "#89F0CB")
	darkGreen := s.adaptive("#1C8760", "#1C8760")
	lineNumberFg := s.adaptive("#656565", "#7D7D7D")
	statusBarNoteFg := s.adaptive("#656565", "#7D7D7D")
	statusBarBg := s.adaptive("#E6E6E6", "#242424")

	// Render-func styles
	s.dimNormalFg = lipgloss.NewStyle().Foreground(normalDim).Render
	s.brightGrayFg = lipgloss.NewStyle().Foreground(brightGray).Render
	s.dimBrightGrayFg = lipgloss.NewStyle().Foreground(dimBrightGray).Render
	s.grayFg = lipgloss.NewStyle().Foreground(gray).Render
	s.midGrayFg = lipgloss.NewStyle().Foreground(midGray).Render
	s.darkGrayFg = lipgloss.NewStyle().Foreground(darkGray)
	s.greenFg = lipgloss.NewStyle().Foreground(green).Render
	s.semiDimGreenFg = lipgloss.NewStyle().Foreground(semiDimGreen).Render
	s.dimGreenFg = lipgloss.NewStyle().Foreground(dimGreen).Render
	s.fuchsiaFg = lipgloss.NewStyle().Foreground(s.fuchsia).Render
	s.dimFuchsiaFg = lipgloss.NewStyle().Foreground(dimFuchsia).Render
	s.dullFuchsiaFg = lipgloss.NewStyle().Foreground(dullFuchsia).Render
	s.dimDullFuchsiaFg = lipgloss.NewStyle().Foreground(dimDullFuchsia).Render
	s.redFg = lipgloss.NewStyle().Foreground(red).Render

	// Named styles
	s.tabStyle = lipgloss.NewStyle().Foreground(s.adaptive("#909090", "#626262"))
	s.selectedTabStyle = lipgloss.NewStyle().Foreground(s.adaptive("#333333", "#979797"))
	s.errorTitleStyle = lipgloss.NewStyle().Foreground(cream).Background(red).Padding(0, 1)
	s.subtleStyle = lipgloss.NewStyle().Foreground(s.adaptive("#9B9B9B", "#5C5C5C"))
	s.paginationStyle = s.subtleStyle

	// Pager styles
	s.statusBarScrollPosStyle = lipgloss.NewStyle().
		Foreground(s.adaptive("#949494", "#5A5A5A")).
		Background(statusBarBg).
		Render

	s.statusBarNoteStyle = lipgloss.NewStyle().
		Foreground(statusBarNoteFg).
		Background(statusBarBg).
		Render

	s.statusBarHelpStyle = lipgloss.NewStyle().
		Foreground(statusBarNoteFg).
		Background(s.adaptive("#DCDCDC", "#323232")).
		Render

	s.statusBarMessageStyle = lipgloss.NewStyle().
		Foreground(mintGreen).
		Background(darkGreen).
		Render

	s.statusBarMessageScrollPosStyle = lipgloss.NewStyle().
		Foreground(mintGreen).
		Background(darkGreen).
		Render

	s.statusBarMessageHelpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#B6FFE4")).
		Background(green).
		Render

	s.helpViewStyle = lipgloss.NewStyle().
		Foreground(statusBarNoteFg).
		Background(s.adaptive("#f2f2f2", "#1B1B1B")).
		Render

	s.lineNumberStyle = lipgloss.NewStyle().
		Foreground(lineNumberFg).
		Render

	// Stash styles
	s.dividerDot = s.darkGrayFg.SetString(" • ")
	s.dividerBar = s.darkGrayFg.SetString(" │ ")

	s.logoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ECFD65")).
		Background(s.fuchsia).
		Bold(true)

	s.stashSpinnerStyle = lipgloss.NewStyle().
		Foreground(gray)
	s.inputPromptStyle = lipgloss.NewStyle().
		Foreground(yellowGreen).
		MarginRight(1)

	s.searchHighlightStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(yellowGreen)
	s.searchSelectedHighlightStyle = lipgloss.NewStyle().
		Foreground(cream).
		Background(s.fuchsia)

	return s
}

// adaptive returns a color appropriate for the current terminal background.
func (s Styles) adaptive(light, dark string) color.Color {
	return s.lightDark(lipgloss.Color(light), lipgloss.Color(dark))
}
