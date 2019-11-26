package gold

import (
	"fmt"
	"io"
	"strconv"

	"github.com/logrusorgru/aurora"
	bf "gopkg.in/russross/blackfriday.v2"
)

type BaseElement struct {
	Token string
	Pre   string
	Post  string
	Style StyleType
}

func (e *BaseElement) Render(w io.Writer, node *bf.Node, tr *TermRenderer) error {
	if e.Pre != "" {
		fmt.Fprintf(w, "%s", e.Pre)
	}
	defer func() {
		if e.Post != "" {
			fmt.Fprintf(w, "%s", e.Post)
		}
	}()

	rules := tr.style[e.Style]
	if rules == nil {
		fmt.Fprintf(w, "%s", e.Token)
		return nil
	}

	out := aurora.Reset(e.Token)
	if rules.Color != "" {
		i, err := strconv.Atoi(rules.Color)
		if err == nil && i >= 0 && i <= 255 {
			out = out.Index(uint8(i))
		}
	}
	if rules.BackgroundColor != "" {
		i, err := strconv.Atoi(rules.BackgroundColor)
		if err == nil && i >= 0 && i <= 255 {
			out = out.Index(uint8(i))
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

	w.Write([]byte(aurora.Sprintf("%s", out)))
	return nil
}
