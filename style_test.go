package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"
)

// A custom style path from the config file has to be expanded, otherwise the
// style file is looked up at a literal "~" path and cannot be found.
func TestValidateStyleExpandsPath(t *testing.T) {
	dir := t.TempDir()
	styleFile := filepath.Join(dir, "style.json")
	if err := os.WriteFile(styleFile, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GLOW_TEST_STYLE_DIR", dir)

	style, err := validateStyle(filepath.Join("$GLOW_TEST_STYLE_DIR", "style.json"))
	if err != nil {
		t.Fatalf("expected the style to be found: %v", err)
	}
	if style != styleFile {
		t.Fatalf("expected the expanded path %q, got %q", styleFile, style)
	}
}

// A path that starts with a tilde is expanded to the home directory.
func TestValidateStyleExpandsTilde(t *testing.T) {
	expanded, err := homedir.Expand("~/glow-style-test.json")
	if err != nil {
		t.Skip("the home directory cannot be determined")
	}

	style, err := validateStyle("~/glow-style-test.json")
	if err == nil {
		t.Fatal("expected an error for a style that does not exist")
	}
	if style != expanded {
		t.Fatalf("expected the expanded path %q, got %q", expanded, style)
	}
	if strings.HasPrefix(style, "~") {
		t.Fatalf("expected the tilde to be expanded, got %q", style)
	}
}
