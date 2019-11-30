package gold

import (
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

func renderText(w io.Writer, rules *ElementStyle, s string) {
	out := aurora.Reset(s)
	if rules.Color != "" {
		i, err := strconv.Atoi(rules.Color)
		if err == nil && i >= 0 && i <= 255 {
			out = out.Index(uint8(i))
		}
	}
	if rules.BackgroundColor != "" {
		i, err := strconv.Atoi(rules.BackgroundColor)
		if err == nil && i >= 0 && i <= 255 {
			out = out.BgIndex(uint8(i))
		}
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

	_, _ = w.Write([]byte(out.String()))
}

func (e *BaseElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	if e.Prefix != "" {
		_, _ = w.Write([]byte(e.Prefix))
	}
	defer func() {
		if e.Suffix != "" {
			_, _ = w.Write([]byte(e.Suffix))
		}
	}()

	rules := tr.style[e.Style]
	if rules == nil {
		_, _ = w.Write([]byte(e.Token))
		return nil
	}

	if rules.Prefix != "" {
		renderText(w, rules, rules.Prefix)
	}
	defer func() {
		if rules.Suffix != "" {
			renderText(w, rules, rules.Suffix)
		}
	}()

	renderText(w, rules, e.Token)
	return nil
}
