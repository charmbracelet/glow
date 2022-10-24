package main

import (
	"bytes"
	"os"
	"reflect"
	"testing"

	"github.com/charmbracelet/glow/utils"
)

func TestGlowSources(t *testing.T) {
	tt := []string{
		".",
		"README.md",
		"github.com/charmbracelet/glow",
		"https://github.com/charmbracelet/glow",
	}

	for _, v := range tt {
		buf := &bytes.Buffer{}
		err := executeArg(rootCmd, v, buf)

		if err != nil {
			t.Errorf("Error during execution (args: %s): %v", v, err)
		}
		if buf.Len() == 0 {
			t.Errorf("Output buffer should not be empty (args: %s)", v)
		}
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

func TestPagerCommandValues(t *testing.T) {
	tt := []struct {
		pagerEnvVar     string
		expectedCommand []string
	}{
		{
			pagerEnvVar:     "C:\\Program Files\\Git\\usr\\bin\\less.exe",
			expectedCommand: []string{"C:\\Program Files\\Git\\usr\\bin\\less.exe"},
		},
		{
			pagerEnvVar:     "usr/local/bin",
			expectedCommand: []string{"usr/local/bin"},
		},
		{
			pagerEnvVar:     "",
			expectedCommand: []string{"less", "-r"},
		},
	}

	for _, v := range tt {
		pager := "PAGER"
		os.Setenv(pager, v.pagerEnvVar)
		command := utils.GetPagerCommand(pager)
		if !reflect.DeepEqual(command, v.expectedCommand) {
			t.Errorf("Expected: %s Actual %s (pagerEnvVar %s)", v.expectedCommand, command, v.pagerEnvVar)
		}
	}
}
