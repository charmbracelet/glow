package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type TextPosition int

const (
	TextStart TextPosition = iota
	TextMiddle
	TextEnd
)

func executeWithFlow(cmd *cobra.Command, src *source, w io.Writer) error {
	var line string
	var nextLine string
	for {
		b := make([]byte, 1024)
		n, err := src.reader.Read(b)
		if err != nil {
			if err == io.EOF {
				if n == 0 {
					if len(line) > 0 {
						out := renderString(line, "NOT USED NOW", TextStart)
						writeRendered(w, &line, &out)
						line = ""
					}
					if nextLine == "" {
						return nil
					}
				}
			} else {
				return fmt.Errorf("unable to read from reader: %w", err)
			}
		}

		if n == 0 && nextLine == "" {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// if buffered nextlin exists, update current line
		if nextLine != "" {
			line = strings.TrimLeft(nextLine, "\n")
			if strings.HasPrefix(nextLine, "\n") {
				fmt.Fprint(w, "\n")
			}
			nextLine = ""
		}

		chunk := strings.ReplaceAll(string(b[:n]), "\r", "")
		fullLine := line + chunk

		if i := strings.IndexAny(fullLine, "\n"); i != -1 {
			line = fullLine[:i+1]
			if len(fullLine) > i+1 {
				nextLine = fullLine[i+1:]
			}
		} else {
			line = fullLine
		}

		out := renderString(line, "NOT USED NOW", TextStart)

		if strings.HasSuffix(line, "\n") {

			writeRendered(w, &line, &out)

			line = ""
			continue
		}
		if len(line) > int(width) {
			continue
		}
		writeRendered(w, &line, &out)
	}
}

func renderString(in string, style string, position TextPosition) string {

	var styleConfig ansi.StyleConfig

	// Get style
	if lipgloss.HasDarkBackground() {
		styleConfig = styles.DarkStyleConfig
	} else {
		styleConfig = styles.LightStyleConfig
	}

	m := uint(2)
	styleConfig.Document.BlockPrefix = ""
	styleConfig.Document.BlockSuffix = ""
	styleConfig.Document.Margin = &m

	r, err := glamour.NewTermRenderer(
		glamour.WithColorProfile(lipgloss.ColorProfile()),
		glamour.WithStyles(styleConfig),
		glamour.WithWordWrap(int(0)), //nolint:gosec
	)

	if err != nil {
		panic(err)
	}

	out, _ := r.Render(in)
	out = strings.ReplaceAll(out, "\n", "")
	return out
}

func writeRendered(w io.Writer, line, out *string) {
	if strings.HasPrefix(*line, "\n") {
		fmt.Fprint(w, "\n")
	}
	fmt.Fprint(w, "\r", strings.Repeat(" ", int(width)))
	fmt.Fprint(w, "\r"+(*out))
	if strings.HasSuffix(*line, "\n") {
		fmt.Fprint(w, "\n")
	}
}
