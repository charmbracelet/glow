package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2/ansi"
)

// testPNG returns the bytes of a small PNG image.
func testPNG(t *testing.T) []byte {
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
	return pngBuf.Bytes()
}

// writeTestDocument writes a markdown document to a temporary directory and
// returns its path.
func writeTestDocument(t *testing.T, content string) string {
	t.Helper()

	mdPath := filepath.Join(t.TempDir(), "test.md")
	if err := os.WriteFile(mdPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return mdPath
}

// writeTestFiles creates a markdown document referencing a PNG image and
// returns the markdown file's path.
func writeTestFiles(t *testing.T) string {
	t.Helper()

	if err := os.WriteFile(filepath.Join(t.TempDir(), "test.png"), testPNG(t), 0o644); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.png"), testPNG(t), 0o644); err != nil {
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

// TestHasRemoteImages checks the detection of images that have to be fetched
// over the network.
func TestHasRemoteImages(t *testing.T) {
	for _, tc := range []struct {
		markdown string
		want     bool
	}{
		{"![alt](https://example.com/img.png)", true},
		{"![](http://example.com/img.png)", true},
		{`<img src="https://example.com/img.png">`, true},
		{"![build][badge]\n\n[badge]: https://example.com/badge.svg", true},
		{"![build][BADGE]\n\n[badge]: https://example.com/badge.svg", true},
		{"![build][badge]\n\n[badge]: badge.svg", false},
		{"[docs]: https://example.com\n\nsee [docs]", false},
		{"![alt](img.png)", false},
		{"[alt](https://example.com)", false},
		{`<img src="img.png">`, false},
		{"no images at all", false},
	} {
		if got := hasRemoteImages(tc.markdown); got != tc.want {
			t.Errorf("%q: expected %v, got %v", tc.markdown, tc.want, got)
		}
	}
}

// TestPagerRemoteImagesDisabled checks that remote images are not fetched by
// default, and that their URL is rendered with a note pointing at the config
// setting that loads them.
func TestPagerRemoteImagesDisabled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("no remote image may be fetched while remote image loading is disabled")
	}))
	defer srv.Close()

	mdPath := writeTestDocument(t, "# Hello\n\n![red]("+srv.URL+"/test.png)\n")

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

	go func() {
		time.Sleep(500 * time.Millisecond)
		p.Send(tea.Quit())
	}()
	if _, err := p.Run(); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	// The note can wrap or be cut off by the terminal, so check for the
	// parts of it that always fit.
	if !strings.Contains(got, "not loaded.") {
		t.Errorf("expected a note about the image not being loaded, got: %q", got)
	}
	if !strings.Contains(got, "loadRemoteImage") {
		t.Errorf("expected a note about the loadRemoteImages setting, got: %q", got)
	}
	if strings.Contains(got, "\x1b_G") {
		t.Errorf("expected no graphics commands, got: %q", got)
	}
	if strings.ContainsRune(got, 0x10EEEE) {
		t.Errorf("expected no placeholders, got: %q", got)
	}
}

// TestGlamourRenderRemoteImages checks how remote images are rendered,
// depending on the config: with loading disabled they are annotated with a
// note, and while they are pending the note is withheld so they aren't
// reported as not loaded.
func TestGlamourRenderRemoteImages(t *testing.T) {
	url := "https://example.com/img.png"
	md := "![alt](" + url + ")"

	newPager := func(loadRemoteImages bool) pagerModel {
		common := &commonModel{
			cfg:           Config{GlamourEnabled: true, Images: true, LoadRemoteImages: loadRemoteImages, GlamourStyle: "dark"},
			styles:        newStyles(true),
			imageProtocol: ansi.ImageProtocolKitty,
		}
		m := newPagerModel(common)
		// No wrap width, so the rendered note stays on a single line and the
		// rendering is deterministic.
		m.setSize(0, 24)
		m.currentDocument = markdown{Note: "test.md", Body: md}
		return m
	}

	// Disabled: the URL is followed by a note pointing at the setting.
	out, _, err := glamourRender(newPager(false), md, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, RemoteImageNotLoadedNote) {
		t.Errorf("expected the not-loaded note, got: %q", out)
	}

	// Enabled: the first pass renders the text without claiming the images
	// weren't loaded; the second pass loads them.
	out, _, err = glamourRender(newPager(true), md, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, url) {
		t.Errorf("expected the image URL to render, got: %q", out)
	}
	if strings.Contains(out, "not loaded") {
		t.Errorf("expected no not-loaded note while the images load, got: %q", out)
	}
}

// TestPagerLoadRemoteImages checks that with remote image loading enabled the
// document's remote images are fetched and rendered.
func TestPagerLoadRemoteImages(t *testing.T) {
	var requests atomic.Int32
	png := testPNG(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(png)
	}))
	defer srv.Close()

	mdPath := writeTestDocument(t, "# Hello\n\n![red]("+srv.URL+"/test.png)\n\nBye!\n")

	cfg := Config{
		GlamourEnabled:   true,
		GlamourStyle:     "dark",
		GlamourMaxWidth:  80,
		Images:           true,
		ImageProtocol:    "kitty",
		LoadRemoteImages: true,
		Path:             mdPath,
	}

	m := newModel(cfg, "")

	var out bytes.Buffer
	p := tea.NewProgram(m,
		tea.WithOutput(&out),
		tea.WithInput(strings.NewReader("")),
		tea.WithWindowSize(80, 24),
	)

	go func() {
		// Leave room for the second rendering pass that fetches the image.
		time.Sleep(time.Second)
		p.Send(tea.Quit())
	}()
	if _, err := p.Run(); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	if requests.Load() == 0 {
		t.Error("expected the remote image to be fetched")
	}
	// The image may only be transmitted by the second rendering pass, which
	// fetched it.
	if !strings.Contains(got, "\x1b_G") {
		t.Errorf("expected the remote image to be transmitted, got: %q", got)
	}
	if !strings.ContainsRune(got, 0x10EEEE) {
		t.Errorf("expected the image to be displayed via placeholders, got: %q", got)
	}
	if strings.Contains(got, "loadRemoteImages") {
		t.Errorf("expected no not-loaded note, got: %q", got)
	}
}

// TestPagerRemoteImagesLoading checks that the pager reports the remote
// images as loading, and only displays the results of the current fetching
// pass.
func TestPagerRemoteImagesLoading(t *testing.T) {
	common := commonModel{cfg: Config{GlamourEnabled: true}, styles: newStyles(true), width: 80, height: 24}
	m := newPagerModel(&common)
	m.setSize(80, 24)
	m.currentDocument = markdown{Note: "test.md", Body: "![](https://example.com/img.png)"}

	m, cmd := m.update(contentRenderedMsg{
		content:             "text without images",
		body:                m.currentDocument.Body,
		remoteImagesPending: true,
	})
	if !m.remoteImagesLoading {
		t.Error("expected the remote images to be reported as loading")
	}
	if cmd == nil {
		t.Error("expected a command that fetches the remote images")
	}
	if !strings.Contains(m.View(), "Loading remote images") {
		t.Errorf("expected a loading indicator, got: %q", m.View())
	}

	// Results of an older pass, e.g. one superseded by a resize, are stale.
	m, _ = m.update(remoteImagesLoadedMsg{content: "stale", token: m.remoteImagesPass - 1})
	if !m.remoteImagesLoading {
		t.Error("expected the stale result to be ignored")
	}
	if got := m.viewport.GetContent(); got != "text without images" {
		t.Errorf("expected the text to stay on screen, got: %q", got)
	}

	// The result of the current pass replaces the text-only content.
	m, _ = m.update(remoteImagesLoadedMsg{content: "text with images", token: m.remoteImagesPass})
	if m.remoteImagesLoading {
		t.Error("expected the remote images to be reported as loaded")
	}
	if got := m.viewport.GetContent(); got != "text with images" {
		t.Errorf("expected the rendered images on screen, got: %q", got)
	}
	if strings.Contains(m.View(), "Loading remote images") {
		t.Errorf("expected the loading indicator to be gone, got: %q", m.View())
	}
}
