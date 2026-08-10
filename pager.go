package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"mvdan.cc/sh/v3/shell"
)

func runPagerCommand(content string) error {
	pagerCmd := os.Getenv("PAGER")
	if pagerCmd == "" {
		pagerCmd = "less -r"
	}

	fields, err := shell.Fields(pagerCmd, os.Getenv)
	if err != nil || len(fields) == 0 {
		return fmt.Errorf("unable to parse PAGER command: %q", pagerCmd)
	}

	f, err := os.CreateTemp("", "glow-*.md")
	if err != nil {
		return fmt.Errorf("unable to create pager temp file: %w", err)
	}
	name := f.Name()
	defer os.Remove(name)

	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		return fmt.Errorf("unable to write pager temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("unable to close pager temp file: %w", err)
	}

	args := append(append([]string{}, fields[1:]...), name)
	c := exec.Command(fields[0], args...) //nolint:gosec
	c.Stdout = os.Stdout

	// Interactive pagers like `most` read file arguments but need a real TTY
	// for keyboard input. Piping rendered content on stdin breaks them (#153).
	if tty := pagerTTY(); tty != nil {
		defer tty.Close()
		c.Stdin = tty
	}

	var stderr strings.Builder
	c.Stderr = &stderr

	if err := c.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("pager %q failed: %w: %s", fields[0], err, msg)
		}
		return fmt.Errorf("pager %q failed: %w", fields[0], err)
	}
	return nil
}

func pagerTTY() io.ReadCloser {
	for _, path := range []string{"/dev/tty", "CONIN$"} {
		f, err := os.Open(path)
		if err == nil {
			return f
		}
	}
	return nil
}
