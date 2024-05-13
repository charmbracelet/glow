package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	. "github.com/charmbracelet/lipgloss"
) //nolint: revive

// Colors.
var (
	normal          = AdaptiveColor{Light: "#1A1A1A", Dark: "#dddddd"}
	normalDim       = AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}
	gray            = AdaptiveColor{Light: "#909090", Dark: "#626262"}
	midGray         = AdaptiveColor{Light: "#B2B2B2", Dark: "#4A4A4A"}
	darkGray        = AdaptiveColor{Light: "#DDDADA", Dark: "#3C3C3C"}
	brightGray      = AdaptiveColor{Light: "#847A85", Dark: "#979797"}
	dimBrightGray   = AdaptiveColor{Light: "#C2B8C2", Dark: "#4D4D4D"}
	indigo          = AdaptiveColor{Light: "#5A56E0", Dark: "#7571F9"}
	dimIndigo       = AdaptiveColor{Light: "#9498FF", Dark: "#494690"}
	subtleIndigo    = AdaptiveColor{Light: "#7D79F6", Dark: "#514DC1"}
	dimSubtleIndigo = AdaptiveColor{Light: "#BBBDFF", Dark: "#383584"}
	cream           = AdaptiveColor{Light: "#FFFDF5", Dark: "#FFFDF5"}
	yellowGreen     = AdaptiveColor{Light: "#04B575", Dark: "#ECFD65"}
	dullYellowGreen = AdaptiveColor{Light: "#6BCB94", Dark: "#9BA92F"}
	fuchsia         = AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"}
	dimFuchsia      = AdaptiveColor{Light: "#F1A8FF", Dark: "#99519E"}
	dullFuchsia     = AdaptiveColor{Dark: "#AD58B4", Light: "#F793FF"}
	dimDullFuchsia  = AdaptiveColor{Light: "#F6C9FF", Dark: "#6B3A6F"}
	green           = Color("#04B575")
	red             = AdaptiveColor{Light: "#FF4672", Dark: "#ED567A"}
	faintRed        = AdaptiveColor{Light: "#FF6F91", Dark: "#C74665"}

	semiDimGreen = AdaptiveColor{Light: "#35D79C", Dark: "#036B46"}
	dimGreen     = AdaptiveColor{Light: "#72D2B0", Dark: "#0B5137"}
)

type styler func(...string) string

// Ulimately, we'll transition to named styles.
// nolint:deadcode,unused,varcheck
type styles struct {
	NormalFg    styler
	DimNormalFg styler

	BrightGrayFg    styler
	DimBrightGrayFg styler

	GrayFg     styler
	MidGrayFg  styler
	DarkGrayFg styler

	GreenFg        styler
	SemiDimGreenFg styler
	DimGreenFg     styler

	FuchsiaFg    styler
	DimFuchsiaFg styler

	DullFuchsiaFg    styler
	DimDullFuchsiaFg styler

	IndigoFg    styler
	DimIndigoFg styler

	SubtleIndigoFg    styler
	DimSubtleIndigoFg styler

	YellowFg     styler
	DullYellowFg styler
	RedFg        styler
	FaintRedFg   styler

	TabStyle         Style
	SelectedTabStyle Style
	ErrorTitleStyle  Style
	SubtleStyle      Style
	PaginationStyle  Style

	LogoStyle             Style
	StashSpinnerStyle     Style
	StashInputPromptStyle Style
	StashInputCursorStyle Style

	DividerDot        string
	DividerBar        string
	OfflineHeaderNote string
}

func defaultStyles(ctx tea.Context) styles {
	darkGrayFg := ctx.NewStyle().Foreground(darkGray).Render
	subtleStyle := ctx.NewStyle().
		Foreground(AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"})

	return styles{
		NormalFg:    ctx.NewStyle().Foreground(normal).Render,
		DimNormalFg: ctx.NewStyle().Foreground(normalDim).Render,

		BrightGrayFg:    ctx.NewStyle().Foreground(brightGray).Render,
		DimBrightGrayFg: ctx.NewStyle().Foreground(dimBrightGray).Render,

		GrayFg:     ctx.NewStyle().Foreground(gray).Render,
		MidGrayFg:  ctx.NewStyle().Foreground(midGray).Render,
		DarkGrayFg: darkGrayFg,

		GreenFg:        ctx.NewStyle().Foreground(green).Render,
		SemiDimGreenFg: ctx.NewStyle().Foreground(semiDimGreen).Render,
		DimGreenFg:     ctx.NewStyle().Foreground(dimGreen).Render,

		FuchsiaFg:    ctx.NewStyle().Foreground(fuchsia).Render,
		DimFuchsiaFg: ctx.NewStyle().Foreground(dimFuchsia).Render,

		DullFuchsiaFg:    ctx.NewStyle().Foreground(dullFuchsia).Render,
		DimDullFuchsiaFg: ctx.NewStyle().Foreground(dimDullFuchsia).Render,

		IndigoFg:    ctx.NewStyle().Foreground(fuchsia).Render,
		DimIndigoFg: ctx.NewStyle().Foreground(dimIndigo).Render,

		SubtleIndigoFg:    ctx.NewStyle().Foreground(subtleIndigo).Render,
		DimSubtleIndigoFg: ctx.NewStyle().Foreground(dimSubtleIndigo).Render,

		YellowFg:     ctx.NewStyle().Foreground(yellowGreen).Render,     // renders light green on light backgrounds
		DullYellowFg: ctx.NewStyle().Foreground(dullYellowGreen).Render, // renders light green on light backgrounds
		RedFg:        ctx.NewStyle().Foreground(red).Render,
		FaintRedFg:   ctx.NewStyle().Foreground(faintRed).Render,
		TabStyle: ctx.NewStyle().
			Foreground(AdaptiveColor{Light: "#909090", Dark: "#626262"}),

		SelectedTabStyle: ctx.NewStyle().
			Foreground(AdaptiveColor{Light: "#333333", Dark: "#979797"}),

		ErrorTitleStyle: ctx.NewStyle().
			Foreground(cream).
			Background(red).
			Padding(0, 1),

		SubtleStyle: subtleStyle,

		PaginationStyle: subtleStyle.Copy(),

		LogoStyle: ctx.NewStyle().
			Foreground(lipgloss.Color("#ECFD65")).
			Background(fuchsia).
			Bold(true),

		StashSpinnerStyle: ctx.NewStyle().
			Foreground(gray),
		StashInputPromptStyle: ctx.NewStyle().
			Foreground(yellowGreen).
			MarginRight(1),
		StashInputCursorStyle: ctx.NewStyle().
			Foreground(fuchsia).
			MarginRight(1),

		DividerDot:        darkGrayFg(" • "),
		DividerBar:        darkGrayFg(" │ "),
		OfflineHeaderNote: darkGrayFg("(Offline)"),
	}
}
