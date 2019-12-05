package gold

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/logrusorgru/aurora"
	"github.com/lucasb-eyer/go-colorful"
	bf "gopkg.in/russross/blackfriday.v2"
)

type BaseElement struct {
	Token  string
	Prefix string
	Suffix string
	Style  ElementStyle
}

func color(c *string) (uint8, error) {
	if c == nil || len(*c) == 0 {
		return 0, errors.New("Invalid color")
	}
	if (*c)[0] == '#' {
		i, err := hexToANSIColor(*c)
		return uint8(i), err
	}
	i, err := strconv.Atoi(*c)
	return uint8(i), err
}

func colorSeq(c *string) (string, error) {
	if c == nil || len(*c) == 0 {
		return "", errors.New("Invalid color")
	}

	col, err := colorful.Hex(*c)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d;%d;%dm", uint8(col.R*255), uint8(col.G*255), uint8(col.B*255)), nil
}

func renderText(w io.Writer, rules ElementStyle, s string) {
	if len(s) == 0 {
		return
	}

	// FIXME: ugly true-color ANSI support hack
	if os.Getenv("COLORTERM") == "truecolor" {
		bg, err := colorSeq(rules.BackgroundColor)
		if err == nil {
			s = "\x1b[48;2;" + bg + s
		}
		fg, err := colorSeq(rules.Color)
		if err == nil {
			s = "\x1b[38;2;" + fg + s
		}
	}

	out := aurora.Reset(s)

	if rules.Color != nil {
		i, err := color(rules.Color)
		if err == nil {
			out = out.Index(i)
		}
	}
	if rules.BackgroundColor != nil {
		i, err := color(rules.BackgroundColor)
		if err == nil {
			out = out.BgIndex(i)
		}
	}
	if rules.Underline != nil && *rules.Underline {
		out = out.Underline()
	}
	if rules.Bold != nil && *rules.Bold {
		out = out.Bold()
	}
	if rules.Italic != nil && *rules.Italic {
		out = out.Italic()
	}
	if rules.CrossedOut != nil && *rules.CrossedOut {
		out = out.CrossedOut()
	}
	if rules.Overlined != nil && *rules.Overlined {
		out = out.Overlined()
	}
	if rules.Inverse != nil && *rules.Inverse {
		out = out.Reverse()
	}
	if rules.Blink != nil && *rules.Blink {
		out = out.Blink()
	}

	_, _ = w.Write([]byte(out.String()))
}

func (e *BaseElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	renderText(w, tr.blockStack.Current().Style, e.Prefix)
	defer func() {
		renderText(w, tr.blockStack.Current().Style, e.Suffix)
	}()

	rules := tr.blockStack.With(e.Style)
	renderText(w, rules, rules.Prefix)
	defer func() {
		renderText(w, rules, rules.Suffix)
	}()

	renderText(w, rules, e.Token)
	return nil
}
