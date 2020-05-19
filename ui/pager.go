package ui

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/charmbracelet/boba"
	"github.com/charmbracelet/boba/viewport"
	"github.com/charmbracelet/charm"
	"github.com/charmbracelet/glamour"
	te "github.com/muesli/termenv"
)

// MSG

type contentRenderedMsg string

// MODEL

type pagerState int

const (
	pagerStateNormal pagerState = iota
	pagerStateSetNote
)

type pagerModel struct {
	err          error
	viewport     viewport.Model
	glamourStyle string
	width        int
	height       int

	// Current document being rendered, sans-glamour rendering. We cache
	// this here so we can re-render it on resize.
	currentDocument *charm.Markdown
}

func (m *pagerModel) setSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
}

func (m *pagerModel) setContent(s string) {
	m.viewport.SetContent(s)
}

func (m *pagerModel) unload() {
	m.viewport.SetContent("")
	m.viewport.Y = 0
}

// UPDATE

func pagerUpdate(msg boba.Msg, m pagerModel) (pagerModel, boba.Cmd) {
	switch msg := msg.(type) {

	// Glow has rendered the content
	case contentRenderedMsg:
		m.setContent(string(msg))
		return m, nil

	// We've reveived terminal dimensions, either for the first time or
	// after a resize
	case terminalSizeMsg:
		if msg.Error() != nil {
			m.err = msg.Error()
			return m, nil
		}

		var cmd boba.Cmd
		if m.currentDocument != nil {
			cmd = renderWithGlamour(m, m.currentDocument.Body)
		}
		return m, cmd
	}

	var cmd boba.Cmd
	m.viewport, cmd = viewport.Update(msg, m.viewport)

	return m, cmd
}

// VIEW

func pagerView(m pagerModel) string {
	return fmt.Sprintf(
		"\n%s\n%s",
		viewport.View(m.viewport),
		pagerStatusBarView(m),
	)
}

func pagerStatusBarView(m pagerModel) string {
	// Logo
	logoText := " Glow "
	logo := glowLogoView(logoText)

	// Scroll percent
	scrollPercent := math.Max(0.0, math.Min(1.0, m.viewport.ScrollPercent()))
	percentText := fmt.Sprintf(" %3.f%% ", scrollPercent*100)
	percent := te.String(percentText).
		Foreground(statusBarFg.Color()).
		Background(statusBarBg.Color()).
		String()

	// Note
	noteText := m.currentDocument.Note
	if len(noteText) == 0 {
		noteText = "(No title)"
	}
	noteText = truncate(" "+noteText+" ", max(0, m.width-len(logoText)-len(percentText)))
	note := te.String(noteText).
		Foreground(statusBarFg.Color()).
		Background(statusBarBg.Color()).String()

	// Empty space
	emptyCell := te.String(" ").Background(statusBarBg.Color()).String()
	padding := max(0, m.width-len(logoText)-len(noteText)-len(percentText))
	emptySpace := strings.Repeat(emptyCell, padding)

	return logo + note + emptySpace + percent
}

// CMD

func renderWithGlamour(m pagerModel, md string) boba.Cmd {
	return func() boba.Msg {
		s, err := glamourRender(m, md)
		if err != nil {
			return errMsg(err)
		}
		return contentRenderedMsg(s)
	}
}

// This is where the magic happens
func glamourRender(m pagerModel, markdown string) (string, error) {

	if os.Getenv("GLOW_DISABLE_GLAMOUR") != "" {
		return markdown, nil
	}

	// initialize glamour
	var gs glamour.TermRendererOption
	if m.glamourStyle == "auto" {
		gs = glamour.WithAutoStyle()
	} else {
		gs = glamour.WithStylePath(m.glamourStyle)
	}

	r, err := glamour.NewTermRenderer(
		gs,
		glamour.WithWordWrap(min(120, m.viewport.Width)),
	)
	if err != nil {
		return "", err
	}

	out, err := r.Render(markdown)
	if err != nil {
		return "", err
	}

	// trim lines
	lines := strings.Split(string(out), "\n")

	var content string
	for i, s := range lines {
		content += strings.TrimSpace(s)

		// don't add an artificial newline after the last split
		if i+1 < len(lines) {
			content += "\n"
		}
	}

	return content, nil
}
