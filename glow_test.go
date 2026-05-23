package main

import (
	"testing"

	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
)

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
		{
			args: []string{"--color", "always"},
			check: func() bool {
				return colorMode == "always"
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

func TestShouldUseNoTTYStyle(t *testing.T) {
	cases := []struct {
		name        string
		mode        string
		isTerminal  bool
		styleFlag   bool
		expectNoTTY bool
	}{
		{name: "auto non-terminal", mode: "auto", isTerminal: false, expectNoTTY: true},
		{name: "always non-terminal", mode: "always", isTerminal: false, expectNoTTY: false},
		{name: "auto terminal", mode: "auto", isTerminal: true, expectNoTTY: false},
		{name: "auto style flag set", mode: "auto", isTerminal: false, styleFlag: true, expectNoTTY: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldUseNoTTYStyle(tc.mode, tc.isTerminal, tc.styleFlag); got != tc.expectNoTTY {
				t.Fatalf("shouldUseNoTTYStyle(%q, %v, %v) = %v, want %v", tc.mode, tc.isTerminal, tc.styleFlag, got, tc.expectNoTTY)
			}
		})
	}
}

func TestResolveStyleForColorMode(t *testing.T) {
	cases := []struct {
		name         string
		mode         string
		currentStyle string
		isTerminal   bool
		styleFlag    bool
		expectStyle  string
	}{
		{name: "auto non-terminal keeps auto when style explicit", mode: "always", currentStyle: "dark", isTerminal: false, styleFlag: true, expectStyle: "dark"},
		{name: "auto terminal keeps auto", mode: "always", currentStyle: styles.AutoStyle, isTerminal: true, expectStyle: styles.AutoStyle},
		{name: "auto mode keeps auto", mode: "auto", currentStyle: styles.AutoStyle, isTerminal: false, expectStyle: styles.AutoStyle},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveStyleForColorMode(tc.mode, tc.currentStyle, tc.isTerminal, tc.styleFlag); got != tc.expectStyle {
				t.Fatalf("resolveStyleForColorMode(%q, %q, %v, %v) = %q, want %q", tc.mode, tc.currentStyle, tc.isTerminal, tc.styleFlag, got, tc.expectStyle)
			}
		})
	}

	want := styles.DarkStyle
	if !lipgloss.HasDarkBackground() {
		want = styles.LightStyle
	}
	if got := resolveStyleForColorMode("always", styles.AutoStyle, false, false); got != want {
		t.Fatalf("resolveStyleForColorMode(always, auto, false, false) = %q, want %q", got, want)
	}
}

func TestValidateColorMode(t *testing.T) {
	if err := validateColorMode("auto"); err != nil {
		t.Fatalf("expected auto to be valid: %v", err)
	}
	if err := validateColorMode("always"); err != nil {
		t.Fatalf("expected always to be valid: %v", err)
	}
	if err := validateColorMode("never"); err == nil {
		t.Fatal("expected never to be invalid")
	}
}
