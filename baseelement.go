package gold

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"text/template"

	"github.com/logrusorgru/aurora"
	"github.com/lucasb-eyer/go-colorful"
)

type BaseElement struct {
	Token  string
	Prefix string
	Suffix string
	Style  ElementStyle
}

func color(c *string) (uint8, error) {
	if c == nil || len(*c) == 0 {
		return 0, errors.New("invalid color")
	}
	if (*c)[0] == '#' {
		i, err := hexToANSIColor(*c)
		return uint8(i), err
	}
	i, err := strconv.Atoi(*c)
	return uint8(i), err
}

func colorSeq(fg *string, bg *string) (string, error) {
	fc := ""
	bc := ""
	if fg != nil {
		fc = *fg
	}
	if bg != nil {
		bc = *bg
	}

	fs := ""
	bs := ""
	f, err := colorful.Hex(fc)
	if err == nil {
		fs = fmt.Sprintf("38;2;%d;%d;%d", uint8(f.R*255), uint8(f.G*255), uint8(f.B*255))
	}
	b, err := colorful.Hex(bc)
	if err == nil {
		bs = fmt.Sprintf("48;2;%d;%d;%d", uint8(b.R*255), uint8(b.G*255), uint8(b.B*255))
	}

	if len(fs) > 0 || len(bs) > 0 {
		seq := "\x1b[" + fs
		if len(fs) > 0 {
			seq += ";"
		}
		return seq + bs + "m", nil
	}

	return "", errors.New("invalid color")
}

func formatToken(format string, token string) (string, error) {
	var b bytes.Buffer

	v := make(map[string]interface{})
	v["text"] = token

	tmpl, err := template.New(format).Funcs(TemplateFuncMap).Parse(format)
	if err != nil {
		return "", err
	}

	err = tmpl.Execute(&b, v)
	return b.String(), err
}

func renderText(w io.Writer, rules ElementStyle, s string) {
	if len(s) == 0 {
		return
	}

	truecolor := os.Getenv("COLORTERM") == "truecolor"
	// FIXME: ugly true-color ANSI support hack
	if truecolor {
		seq, err := colorSeq(rules.Color, rules.BackgroundColor)
		if err == nil {
			s = seq + s
		} else {
			truecolor = false
		}
	}

	out := aurora.Reset(s)

	if !truecolor {
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

func (e *BaseElement) Render(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack

	renderText(w, bs.Current().Style, e.Prefix)
	defer func() {
		renderText(w, bs.Current().Style, e.Suffix)
	}()

	rules := bs.With(e.Style)
	renderText(w, bs.Current().Style, rules.Prefix)
	defer func() {
		renderText(w, bs.Current().Style, rules.Suffix)
	}()

	s := e.Token
	if len(rules.Format) > 0 {
		var err error
		s, err = formatToken(rules.Format, s)
		if err != nil {
			return err
		}
	}
	renderText(w, rules, s)
	return nil
}
