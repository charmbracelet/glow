package gold

import (
	"errors"
	"io"
	"strconv"

	"github.com/logrusorgru/aurora"
	bf "gopkg.in/russross/blackfriday.v2"
)

type BaseElement struct {
	Token  string
	Prefix string
	Suffix string
	Style  StyleType
}

func color(c string) (uint8, error) {
	if len(c) == 0 {
		return 0, errors.New("Invalid color")
	}
	if c[0] == '#' {
		i, err := hexToANSIColor(c)
		return uint8(i), err
	}
	i, err := strconv.Atoi(c)
	return uint8(i), err
}

func renderText(w io.Writer, rules *ElementStyle, s string) {
	if len(s) == 0 {
		return
	}

	out := aurora.Reset(s)

	if rules != nil {
		i, err := color(rules.Color)
		if err == nil {
			out = out.Index(i)
		}
		i, err = color(rules.BackgroundColor)
		if err == nil {
			out = out.BgIndex(i)
		}

		if rules.Underline {
			out = out.Underline()
		}
		if rules.Bold {
			out = out.Bold()
		}
		if rules.Italic {
			out = out.Italic()
		}
		if rules.CrossedOut {
			out = out.CrossedOut()
		}
		if rules.Overlined {
			out = out.Overlined()
		}
		if rules.Inverse {
			out = out.Reverse()
		}
		if rules.Blink {
			out = out.Blink()
		}
	}

	_, _ = w.Write([]byte(out.String()))
}

func (e *BaseElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	renderText(w, tr.blockStack.Current().Style, e.Prefix)
	defer func() {
		renderText(w, tr.blockStack.Current().Style, e.Suffix)
	}()

	rules := tr.blockStack.With(tr.style[e.Style])
	if rules != nil {
		renderText(w, rules, rules.Prefix)
		defer func() {
			renderText(w, rules, rules.Suffix)
		}()
	}

	renderText(w, rules, e.Token)
	return nil
}
