package ui

import (
	lib "github.com/charmbracelet/charm/ui/common"
	te "github.com/muesli/termenv"
)

type styleFunc func(string) string

const (
	darkGray = "#333333"
)

var (
	normalFg    = newFgStyle(lib.NewColorPair("#dddddd", "#1a1a1a"))
	dimNormalFg = newFgStyle(lib.NewColorPair("#777777", "#A49FA5"))

	brightGrayFg    = newFgStyle(lib.NewColorPair("#979797", "#847A85"))
	dimBrightGrayFg = newFgStyle(lib.NewColorPair("#4D4D4D", "#C2B8C2"))

	grayFg     = newFgStyle(lib.NewColorPair("#626262", "#909090"))
	midGrayFg  = newFgStyle(lib.NewColorPair("#4A4A4A", "#B2B2B2"))
	darkGrayFg = newFgStyle(lib.NewColorPair("#3C3C3C", "#DDDADA"))

	greenFg        = newFgStyle(lib.NewColorPair("#04B575", "#04B575"))
	semiDimGreenFg = newFgStyle(lib.NewColorPair("#036B46", "#35D79C"))
	dimGreenFg     = newFgStyle(lib.NewColorPair("#0B5137", "#72D2B0"))

	fuchsiaFg    = newFgStyle(lib.Fuschia)
	dimFuchsiaFg = newFgStyle(lib.NewColorPair("#99519E", "#F1A8FF"))

	dullFuchsiaFg    = newFgStyle(lib.NewColorPair("#AD58B4", "#F793FF"))
	dimDullFuchsiaFg = newFgStyle(lib.NewColorPair("#6B3A6F", "#F6C9FF"))

	indigoFg    = newFgStyle(lib.Indigo)
	dimIndigoFg = newFgStyle(lib.NewColorPair("#494690", "#9498FF"))

	subtleIndigoFg    = newFgStyle(lib.NewColorPair("#514DC1", "#7D79F6"))
	dimSubtleIndigoFg = newFgStyle(lib.NewColorPair("#383584", "#BBBDFF"))

	yellowFg     = newFgStyle(lib.YellowGreen)                        // renders light green on light backgrounds
	dullYellowFg = newFgStyle(lib.NewColorPair("#9BA92F", "#6BCB94")) // renders light green on light backgrounds
	redFg        = newFgStyle(lib.Red)
	faintRedFg   = newFgStyle(lib.FaintRed)

	// Ultimately, we should transition to named styles
	tabColor         = newFgStyle(lib.NewColorPair("#626262", "#909090"))
	selectedTabColor = newFgStyle(lib.NewColorPair("#979797", "#332F33"))
)

// Returns a termenv style with foreground and background options.
func newStyle(fg, bg lib.ColorPair, bold bool) func(string) string {
	s := te.Style{}.Foreground(fg.Color()).Background(bg.Color())
	if bold {
		s = s.Bold()
	}
	return s.Styled
}

// Returns a new termenv style with background options only.
func newFgStyle(c lib.ColorPair) styleFunc {
	return te.Style{}.Foreground(c.Color()).Styled
}
