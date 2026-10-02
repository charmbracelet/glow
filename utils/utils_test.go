package utils

import (
	"testing"

	"charm.land/glamour/v2/ansi"
	"charm.land/glamour/v2/styles"
	"charm.land/x/nerdfont"
)

func TestNerdFontAlertIcons(t *testing.T) {
	t.Cleanup(nerdfont.Reset)

	nerdfont.SetOverride(true)
	cfg, err := styleConfigFor(styles.DarkStyle, false)
	if err != nil {
		t.Fatal(err)
	}
	for kind, want := range map[ansi.AlertKind]string{
		ansi.AlertNote:      nerdFontNote,
		ansi.AlertTip:       nerdFontTip,
		ansi.AlertImportant: nerdFontImportant,
		ansi.AlertWarning:   nerdFontWarning,
		ansi.AlertCaution:   nerdFontCaution,
	} {
		if got := cfg.Alerts.For(kind).Title.Prefix; got != want {
			t.Errorf("expected the %s Nerd Font icon, got %q", kind, got)
		}
	}

	// The default styles themselves must not be modified.
	if got := styles.DarkStyleConfig.Alerts.Note.Title.Prefix; got != "ℹ " {
		t.Errorf("expected the default style to be left alone, got %q", got)
	}

	nerdfont.SetOverride(false)
	cfg, err = styleConfigFor(styles.DarkStyle, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Alerts.Note.Title.Prefix; got != "ℹ " {
		t.Errorf("expected the Unicode fallback, got %q", got)
	}
}

func TestNerdFontAlertIconsPlainStyles(t *testing.T) {
	t.Cleanup(nerdfont.Reset)
	nerdfont.SetOverride(true)

	for _, style := range []string{styles.NoTTYStyle, styles.AsciiStyle} {
		cfg, err := styleConfigFor(style, false)
		if err != nil {
			t.Fatal(err)
		}
		if got := cfg.Alerts.Note.Title.Prefix; got != "" {
			t.Errorf("expected no icon for the %s style, got %q", style, got)
		}
	}
}
