package main

import (
	"os"
	"testing"
)

func TestValidateStyle(t *testing.T) {
	// Create a temporary file to use as a style
	tmpFile, err := os.CreateTemp("", "glow-style-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// Set an environment variable to point to it
	os.Setenv("MY_GLOW_STYLE", tmpFile.Name())
	defer os.Unsetenv("MY_GLOW_STYLE")

	styleToTest := "$MY_GLOW_STYLE"
	styleVar, err := validateStyle(styleToTest)
	if err != nil {
		t.Fatalf("validateStyle failed: %v", err)
	}

	if styleVar != tmpFile.Name() {
		t.Errorf("Style was NOT expanded: %s, expected: %s", styleVar, tmpFile.Name())
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
