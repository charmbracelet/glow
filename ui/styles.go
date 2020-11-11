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

	warmGrayFg    = newFgStyle(common.NewColorPair("#979797", "#847A85"))
	dimWarmGrayFg = newFgStyle(common.NewColorPair("#4D4D4D", "#C2B8C2"))

	grayFg    = newFgStyle(common.NewColorPair("#626262", "#000"))
	dimGrayFg = newFgStyle(common.NewColorPair("#3F3F3F", "#000"))

	greenFg    = newFgStyle(common.NewColorPair("#04B575", "#04B575"))
	dimGreenFg = newFgStyle(common.NewColorPair("#0B5137", "#82E1BF"))

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
func newStyle(fg, bg common.ColorPair) func(string) string {
	return te.Style{}.Foreground(fg.Color()).Background(bg.Color()).Styled
}

// Returns a new termenv style with background options only.
func newFgStyle(c common.ColorPair) styleFunc {
	return te.Style{}.Foreground(c.Color()).Styled
}
