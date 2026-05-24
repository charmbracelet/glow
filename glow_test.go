package main

import (
	"testing"
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

func TestResolveTUIStyle(t *testing.T) {
	cases := []struct {
		name       string
		envStyle   string
		cliStyle   string
		cliChanged bool
		want       string
	}{
		{"cli flag wins over env", "dark", "light", true, "light"},
		{"env wins when valid and no cli", "dark", "auto", false, "dark"},
		{"cli wins when env unset", "", "light", false, "light"},
		{"cli wins when env unknown", "bogus", "light", false, "light"},
		{"env auto preserved without cli", "auto", "light", false, "auto"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveTUIStyle(tc.envStyle, tc.cliStyle, tc.cliChanged)
			if got != tc.want {
				t.Errorf("resolveTUIStyle(%q, %q, %v) = %q, want %q",
					tc.envStyle, tc.cliStyle, tc.cliChanged, got, tc.want)
			}
		})
	}
}
