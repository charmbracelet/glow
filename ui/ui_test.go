package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/ansi"
)

// writeTestFiles creates a markdown document referencing a PNG image and
// returns the markdown file's path.
func writeTestFiles(t *testing.T) string {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 16, 8))
	for x := 0; x < 16; x++ {
		for y := 0; y < 8; y++ {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(t.TempDir(), "test.png"), pngBuf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.png"), pngBuf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	mdPath := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdPath, []byte("# Hello\n\n![red](test.png)\n\nBye!\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return mdPath
}

// TestPagerKittyImages runs the pager on a document with an image and checks
// that the image is transmitted out-of-band before the content, which
// references it via unicode placeholders, is displayed.
func TestPagerKittyImages(t *testing.T) {
	mdPath := writeTestFiles(t)

	cfg := Config{
		GlamourEnabled:  true,
		GlamourStyle:    "dark",
		GlamourMaxWidth: 80,
		Images:          true,
		ImageProtocol:   "kitty",
		Path:            mdPath,
	}

	m := newModel(cfg, "")

	var out bytes.Buffer
	p := tea.NewProgram(m,
		tea.WithOutput(&out),
		tea.WithInput(strings.NewReader("")),
		tea.WithWindowSize(80, 24),
	)

	quit := make(chan struct{})
	go func() {
		// Give the program time to render, then shut it down.
		time.Sleep(500 * time.Millisecond)
		p.Send(tea.Quit())
		close(quit)
	}()
	if _, err := p.Run(); err != nil {
		t.Fatal(err)
	}
	<-quit

	got := out.String()

	// The image must have been transmitted with a virtual placement.
	transmit := strings.Index(got, "\x1b_G")
	if transmit < 0 {
		t.Fatalf("no kitty graphics commands in output: %q", got)
	}
	if !strings.Contains(got, "a=p") || !strings.Contains(got, "U=1") {
		t.Errorf("expected a virtual placement command, got: %q", got)
	}

	// The content must reference the image via unicode placeholders, which
	// the terminal replaces with the image as the content scrolls.
	placeholder := strings.IndexRune(got, 0x10EEEE)
	if placeholder < 0 {
		t.Fatalf("no unicode placeholders in output: %q", got)
	}

	// The transmission must be written to the terminal before the
	// placeholders are displayed, otherwise the terminal can't resolve them
	// until the next repaint.
	if transmit > placeholder {
		t.Errorf("expected graphics commands (%d) before placeholders (%d)", transmit, placeholder)
	}
}

// TestPagerNoImages checks that with images disabled the output contains
// neither graphics commands nor placeholders.
func TestPagerNoImages(t *testing.T) {
	mdPath := writeTestFiles(t)

	cfg := Config{
		GlamourEnabled:  true,
		GlamourStyle:    "dark",
		GlamourMaxWidth: 80,
		Images:          false,
		Path:            mdPath,
	}

	m := newModel(cfg, "")

	var out bytes.Buffer
	p := tea.NewProgram(m,
		tea.WithOutput(&out),
		tea.WithInput(strings.NewReader("")),
		tea.WithWindowSize(80, 24),
	)

	go func() {
		time.Sleep(500 * time.Millisecond)
		p.Send(tea.Quit())
	}()
	if _, err := p.Run(); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	if strings.Contains(got, "\x1b_G") {
		t.Errorf("expected no graphics commands, got: %q", got)
	}
	if strings.ContainsRune(got, 0x10EEEE) {
		t.Errorf("expected no placeholders, got: %q", got)
	}
}

// TestCommonModelImageProtocol checks the initial protocol resolution from
// the config.
func TestCommonModelImageProtocol(t *testing.T) {
	for _, tc := range []struct {
		protocol string
		images   bool
		want     ansi.ImageProtocol
	}{
		{"kitty", true, ansi.ImageProtocolKittyPlaceholders},
		{"none", true, ansi.ImageProtocolNone},
		{"auto", true, ansi.ImageProtocolNone},
		{"kitty", false, ansi.ImageProtocolNone},
	} {
		m := newModel(Config{Images: tc.images, ImageProtocol: tc.protocol}, "")
		if got := m.(model).common.imageProtocol; got != tc.want {
			t.Errorf("protocol %q, images %v: expected %v, got %v", tc.protocol, tc.images, tc.want, got)
		}
	}
}
