package main

import (
	"os"
	"path/filepath"
	"strings"
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

func TestValidateStyle(t *testing.T) {
	// Built-in styles should pass through unchanged.
	for _, name := range []string{"auto", "dark", "light", "notty"} {
		got, err := validateStyle(name)
		if err != nil {
			t.Errorf("validateStyle(%q) returned error: %v", name, err)
		}
		if got != name {
			t.Errorf("validateStyle(%q) = %q, want %q", name, got, name)
		}
	}

	// Custom paths with a leading ~ should be expanded to the user's home
	// directory. This is the regression covered by #713.
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot determine home dir: %v", err)
	}
	dir := t.TempDir()
	rel, err := filepath.Rel(home, dir)
	if err != nil || strings.HasPrefix(rel, "..") {
		t.Skip("temp dir is not under the user's home directory")
	}
	stylePath := filepath.Join(dir, "custom.json")
	if err := os.WriteFile(stylePath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	tildePath := "~/" + filepath.ToSlash(filepath.Join(rel, "custom.json"))

	got, err := validateStyle(tildePath)
	if err != nil {
		t.Fatalf("validateStyle(%q) returned error: %v", tildePath, err)
	}
	if got == tildePath || strings.HasPrefix(got, "~") {
		t.Errorf("validateStyle(%q) = %q, expected ~ to be expanded", tildePath, got)
	}
	if got != stylePath {
		t.Errorf("validateStyle(%q) = %q, want %q", tildePath, got, stylePath)
	}

	// A missing custom style should surface an error mentioning the
	// expanded path (not the raw ~ form), so users can see what was looked
	// up on disk.
	missing := "~/does-not-exist-" + filepath.Base(dir) + ".json"
	_, err = validateStyle(missing)
	if err == nil {
		t.Fatalf("validateStyle(%q) returned no error", missing)
	}
	if strings.Contains(err.Error(), "~") {
		t.Errorf("error message %q still contains a raw tilde", err.Error())
	}
}
