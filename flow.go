package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

func executeWithFlow(cmd *cobra.Command, src *source, w io.Writer) error {
	var line string
	var nextLine string
	for {
		b := make([]byte, 1024)
		n, err := src.reader.Read(b)
		if err != nil {
			if err == io.EOF {
				if n == 0 && nextLine == "" {
					return nil
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

		if strings.HasSuffix(line, "\n") {
			fmt.Fprint(w, "\r"+line)
			line = ""
			continue
		}
		if len(line) > int(width) {
			continue
		}
		fmt.Fprint(w, "\r"+line)
	}
}

func renderString(in string) string {
	// TODO: Implement flow rendering logic

	r, err := glamour.NewTermRenderer(
		glamour.WithColorProfile(lipgloss.ColorProfile()),
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(int(width)), //nolint:gosec
		glamour.WithBaseURL(baseURL),
		glamour.WithPreservedNewLines(),
	)

	if err != nil {
		panic("err on creating renderer")
	}

	

	/////////***********************

	// b = utils.RemoveFrontmatter(b)

	// styleOptions := utils.GlamourStyle(style, isCode)

	// // initialize glamour
	// r, err := glamour.NewTermRenderer(
	// 	glamour.WithColorProfile(lipgloss.ColorProfile()),
	// 	styleOptions,
	// 	glamour.WithWordWrap(int(width)), //nolint:gosec
	// 	glamour.WithBaseURL(baseURL),
	// 	glamour.WithPreservedNewLines(),
	// )
	// if err != nil {
	// 	return fmt.Errorf("unable to create renderer: %w", err)
	// }

	// content := string(b[:n])

	// if isCode {
	// 	content = utils.WrapCodeBlock(string(b), ext)
	// }
	// line = line + string(b[:n])
	// out, err := r.Render(line)
	// rend := "\r" + strings.TrimRight(out, "\n")
	// if err != nil {
	// 	return fmt.Errorf("unable to render markdown: %w", err)
	// }

	// // display
	// switch {
	// case pager || cmd.Flags().Changed("pager"):
	// 	pagerCmd := os.Getenv("PAGER")
	// 	if pagerCmd == "" {
	// 		pagerCmd = "less -r"
	// 	}

	// 	pa := strings.Split(pagerCmd, " ")
	// 	c := exec.Command(pa[0], pa[1:]...) //nolint:gosec
	// 	c.Stdin = strings.NewReader(out)
	// 	c.Stdout = os.Stdout
	// 	if err := c.Run(); err != nil {
	// 		return fmt.Errorf("unable to run command: %w", err)
	// 	}
	// 	return nil
	// case tui || cmd.Flags().Changed("tui"):
	// 	path := ""
	// 	if !isURL(src.URL) {
	// 		path = src.URL
	// 	}
	// 	return runTUI(path, content)
	// default:
	// 	if _, err = fmt.Fprint(w, rend); err != nil {
	// 		return fmt.Errorf("unable to write to writer: %w", err)
	// 	}
	// }
}
