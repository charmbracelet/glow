package main

import (
	"bytes"
	"errors"
	"net"
	"testing"
)

func TestGlowSources(t *testing.T) {
	tt := []string{
		".",
		"README.md",
		"github.com/charmbracelet/glow",
		"github://charmbracelet/glow",
		"https://github.com/charmbracelet/glow",
	}

	for _, v := range tt {
		t.Run(v, func(t *testing.T) {
			buf := &bytes.Buffer{}
			err := executeArg(rootCmd, v, buf)
			if err != nil {
				// Check for network issues.
				var netErr *net.DNSError
				if errors.As(err, &netErr) {
					t.Logf("Error during execution (args: %s): %v", v, err)
					t.Skip("Test uses network. Are you connected to the Internet?")
				}
				t.Errorf("Error during execution (args: %s): %v", v, err)
			}
			if buf.Len() == 0 {
				t.Errorf("Output buffer should not be empty (args: %s)", v)
			}
		})
	}
}

func TestGlowFlags(t *testing.T) {
	tt := []struct {
		args  []string
		check func() bool
	}{
		{
			args: []string{"-p"},
			check: func() bool {
				return pager
			},
		},
		{
			args: []string{"-s", "light"},
			check: func() bool {
				return style == "light"
			},
		},
		{
			args: []string{"-w", "40"},
			check: func() bool {
				return width == 40
			},
		},
	}

	for _, v := range tt {
		err := rootCmd.ParseFlags(v.args)
		if err != nil {
			t.Fatal(err)
		}
		if !v.check() {
			t.Errorf("Parsing flag failed: %s", v.args)
		}
	}
}
