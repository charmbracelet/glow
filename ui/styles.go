package ui

import (
	"github.com/charmbracelet/charm/ui/common"
	te "github.com/muesli/termenv"
)

type styleFunc func(string) string

const (
	darkGray = "#333333"
)

var (
	normalFg    = newFgStyle(common.NewColorPair("#dddddd", "#1a1a1a"))
	dimNormalFg = newFgStyle(common.NewColorPair("#777777", "#A49FA5"))

	brightGrayFg    = newFgStyle(common.NewColorPair("#979797", "#847A85"))
	dimBrightGrayFg = newFgStyle(common.NewColorPair("#4D4D4D", "#C2B8C2"))

	grayFg     = newFgStyle(common.NewColorPair("#626262", "#909090"))
	midGrayFg  = newFgStyle(common.NewColorPair("#4A4A4A", "#B2B2B2"))
	darkGrayFg = newFgStyle(common.NewColorPair("#3C3C3C", "#DDDADA"))

	greenFg    = newFgStyle(common.NewColorPair("#04B575", "#04B575"))
	dimGreenFg = newFgStyle(common.NewColorPair("#0B5137", "#72D2B0"))

	fuchsiaFg    = newFgStyle(common.Fuschia)
	dimFuchsiaFg = newFgStyle(common.NewColorPair("#99519E", "#F1A8FF"))

	dullFuchsiaFg    = newFgStyle(common.NewColorPair("#AD58B4", "#F793FF"))
	dimDullFuchsiaFg = newFgStyle(common.NewColorPair("#6B3A6F", "#F6C9FF"))

	indigoFg    = newFgStyle(common.Indigo)
	dimIndigoFg = newFgStyle(common.NewColorPair("#494690", "#9498FF"))

	subtleIndigoFg    = newFgStyle(common.NewColorPair("#514DC1", "#7D79F6"))
	dimSubtleIndigoFg = newFgStyle(common.NewColorPair("#383584", "#BBBDFF"))

	yellowFg     = newFgStyle(common.YellowGreen)                        // renders light green on light backgrounds
	dullYellowFg = newFgStyle(common.NewColorPair("#9BA92F", "#6BCB94")) // renders light green on light backgrounds
	redFg        = newFgStyle(common.Red)
	faintRedFg   = newFgStyle(common.FaintRed)
)

// Returns a termenv style with foreground and background options.
func newStyle(fg, bg common.ColorPair, bold bool) func(string) string {
	s := te.Style{}.Foreground(fg.Color()).Background(bg.Color())
	if bold {
		s = s.Bold()
	}
	return s.Styled
}

// Returns a new termenv style with background options only.
func newFgStyle(c common.ColorPair) styleFunc {
	return te.Style{}.Foreground(c.Color()).Styled
}
