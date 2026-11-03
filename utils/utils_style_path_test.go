package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/glamour/v2"
)

// Regression test for charmbracelet/glow#776 and the glow side of
// charmbracelet/glamour#545: a style path from glow.yml that contains a
// tilde or an environment variable must be expanded before the file is
// read, the same way validateStyle already expands it for its existence
// check. Before the fix, both cases failed with "no such file or
// directory" even though the file existed.
func TestStyleConfigFromFileExpandsPath(t *testing.T) {
	dir := t.TempDir()
	styleFile := filepath.Join(dir, "style.json")
	if err := os.WriteFile(styleFile, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Run("env var", func(t *testing.T) {
		t.Setenv("GLOW_TEST_STYLE_DIR", dir)
		if _, err := styleConfigFromFile("$GLOW_TEST_STYLE_DIR/style.json"); err != nil {
			t.Fatalf("styleConfigFromFile with $VAR path failed: %v", err)
		}
		if _, err := styleConfigFor("$GLOW_TEST_STYLE_DIR/style.json", false); err != nil {
			t.Fatalf("styleConfigFor with $VAR path failed: %v", err)
		}
	})

	t.Run("tilde", func(t *testing.T) {
		t.Setenv("HOME", dir)
		if _, err := styleConfigFromFile("~/style.json"); err != nil {
			t.Fatalf("styleConfigFromFile with ~/ path failed: %v", err)
		}
		if _, err := styleConfigFor("~/style.json", false); err != nil {
			t.Fatalf("styleConfigFor with ~/ path failed: %v", err)
		}
	})

	t.Run("plain path still works", func(t *testing.T) {
		if _, err := styleConfigFromFile(styleFile); err != nil {
			t.Fatalf("styleConfigFromFile with plain path failed: %v", err)
		}
	})

	t.Run("missing file still errors", func(t *testing.T) {
		if _, err := styleConfigFromFile(filepath.Join(dir, "does-not-exist.json")); err == nil {
			t.Fatal("expected an error for a missing style file, got nil")
		}
	})
}

// End-to-end through the option both the CLI and the TUI actually use:
// a renderer built with a $VAR style path must render, not fail at
// renderer creation with "no such file or directory".
func TestGlamourStyleExpandsPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "style.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GLOW_TEST_STYLE_DIR", dir)
	r, err := glamour.NewTermRenderer(GlamourStyle("$GLOW_TEST_STYLE_DIR/style.json", false))
	if err != nil {
		t.Fatalf("NewTermRenderer with $VAR style path failed: %v", err)
	}
	out, err := r.Render("# Hello\n")
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(out, "Hello") {
		t.Fatalf("expected rendered output to contain Hello, got %q", out)
	}
}
