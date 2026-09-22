package ui

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
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

// TestViewsFillScreen checks that the views shown between documents (loading,
// error) and the pager before it has been sized fill the terminal height.
// Views that don't cover the whole screen leave whatever the previous frame
// drew on it, e.g. a document's status bar stuck at the bottom after going
// back to the stash.
func TestViewsFillScreen(t *testing.T) {
	const height = 24
	common := commonModel{cfg: Config{GlamourEnabled: true}, styles: newStyles(true), width: 80, height: height}

	check := func(name, view string) {
		t.Helper()
		if got := strings.Count(view, "\n") + 1; got != height {
			t.Errorf("expected the %s view to fill the screen, got %d lines, want %d", name, got, height)
		}
	}

	stash := newStashModel(&common)
	stash.setSize(80, height)
	stash.loaded = true
	check("stash", stash.view())

	stash.viewState = stashStateLoadingDocument
	check("loading", stash.view())

	stash.viewState = stashStateShowingError
	stash.err = errMsg{err: errors.New("boom")}
	check("error", stash.view())

	pager := newPagerModel(&common)
	check("unsized pager", pager.View())
	pager.setSize(80, height)
	check("empty pager", pager.View())
	pager.setContent("hello\nworld")
	check("pager", pager.View())
}

// TestPagerRemoteImagesLoadError checks that a failed loading pass clears the
// loading indicator and reports the error, rather than leaving the indicator
// in the status bar forever.
func TestPagerRemoteImagesLoadError(t *testing.T) {
	common := commonModel{cfg: Config{GlamourEnabled: true}, styles: newStyles(true), width: 80, height: 24}
	m := newPagerModel(&common)
	m.setSize(80, 24)
	m.currentDocument = markdown{Note: "test.md", Body: "![](https://example.com/img.png)"}

	m, _ = m.update(contentRenderedMsg{
		content:             "text without images",
		body:                m.currentDocument.Body,
		remoteImagesPending: true,
	})
	if !m.remoteImagesLoading {
		t.Fatal("expected the remote images to be reported as loading")
	}

	m, cmd := m.update(errMsg{err: errors.New("boom")})
	if m.remoteImagesLoading {
		t.Error("expected the loading indicator to be cleared")
	}
	if strings.Contains(m.View(), "Loading remote images") {
		t.Errorf("expected the loading indicator to be gone, got: %q", m.View())
	}
	if !strings.Contains(m.View(), "boom") {
		t.Errorf("expected the error to be reported, got: %q", m.View())
	}
	if cmd == nil {
		t.Error("expected a status message timeout")
	}
}

// staticViewModel renders a view that never changes, like the loading
// indicator between documents.
type staticViewModel struct{}

func (staticViewModel) Init() tea.Cmd                         { return nil }
func (m staticViewModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (staticViewModel) View() tea.View {
	v := tea.NewView("static")
	v.AltScreen = true
	return v
}

// syncBuffer is a buffer that can be read while the program writes to it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// TestResizeRedrawsUnchangedView checks that a window resize redraws the
// screen even when the view doesn't change, e.g. while a document loads. The
// renderer must not skip the screen erase queued by the resize, or the
// terminal keeps whatever the previous frame drew, like a document's status
// bar left stuck at the bottom (fixed in bubbletea v2.0.9, see
// charmbracelet/bubbletea#1755).
func TestResizeRedrawsUnchangedView(t *testing.T) {
	var out syncBuffer
	p := tea.NewProgram(staticViewModel{},
		tea.WithOutput(&out),
		tea.WithInput(strings.NewReader("")),
		tea.WithWindowSize(80, 24),
	)

	done := make(chan struct{})
	go func() {
		defer close(done)

		// Wait for the initial frame, then resize to the same dimensions so
		// that only the resize can redraw the screen.
		time.Sleep(250 * time.Millisecond)
		before := strings.Count(out.String(), "static")

		p.Send(tea.WindowSizeMsg{Width: 80, Height: 24})
		time.Sleep(250 * time.Millisecond)

		if after := strings.Count(out.String(), "static"); after <= before {
			t.Errorf("expected the screen to be redrawn on resize, got no redraw (frames before=%d, after=%d)",
				before, after)
		}
		p.Send(tea.Quit())
	}()
	if _, err := p.Run(); err != nil {
		t.Fatal(err)
	}
	<-done
}

// testSVG is a small SVG document used in the SVG tests.
const testSVG = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="40" height="20">
	<rect width="40" height="20" fill="red"/>
	<circle cx="20" cy="10" r="8" fill="blue"/>
</svg>
`

// TestPagerKittySVGImages runs the pager on a document with an SVG image and
// checks that the vector image is rasterized and displayed like other
// images.
func TestPagerKittySVGImages(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.svg"), []byte(testSVG), 0o644); err != nil {
		t.Fatal(err)
	}
	mdPath := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdPath, []byte("# Hello\n\n![red](test.svg)\n\nBye!\n"), 0o644); err != nil {
		t.Fatal(err)
	}

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
	// The SVG must be rasterized and transmitted as a PNG, and displayed via
	// unicode placeholders.
	if !strings.Contains(got, "\x1b_G") {
		t.Errorf("expected the SVG to be transmitted, got: %q", got)
	}
	if !strings.ContainsRune(got, 0x10EEEE) {
		t.Errorf("expected the SVG to be displayed via placeholders, got: %q", got)
	}
	// Displayed images don't render their URL next to them.
	if strings.Contains(got, "test.svg") {
		t.Errorf("expected the SVG URL not to be rendered next to the image, got: %q", got)
	}
}

// kittyPlaceholder is the unicode placeholder cell that anchors an image to
// the text grid.
const kittyPlaceholder = '\U0010EEEE'

// escapeSequencePattern matches ANSI SGR, OSC 8 hyperlink, and graphics
// protocol sequences.
var escapeSequencePattern = regexp.MustCompile(`\x1b\]8;[^\x07\x9c]*(\x07|\x9c)|\x1b\[[0-9;:]*[a-zA-Z]|\x1b_G[^\x1b]*\x1b\\|\x1bP[^\x1b]*\x1b\\`)

// visibleText returns the text a terminal would display for the given
// rendered output, i.e. with all escape sequences removed.
func visibleText(s string) string {
	return escapeSequencePattern.ReplaceAllString(s, "")
}

// testBadgeSVG is a badge in the shape of the ones served by shields.io: a
// background split in two, and text drawn in scaled groups.
const testBadgeSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="94" height="20">` +
	`<g><rect width="49" height="20" fill="#555"/>` +
	`<rect x="49" width="45" height="20" fill="#007ec6"/></g>` +
	`<g fill="#fff" text-anchor="middle" font-size="110">` +
	`<g transform="scale(.1)"><text x="255" y="140">release</text></g>` +
	`<g transform="scale(.1)"><text x="705" y="140">v3.0.0</text></g></g></svg>`

// testPathBadgeSVG is a badge whose text is drawn as path outlines, and which
// uses arcs, like the badges served by godoc.org.
const testPathBadgeSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="90" height="20">` +
	`<path d="M2 0h26v20H2a2 2 0 01-2-2V2a2 2 0 012-2z" fill="#5C5C5C"/>` +
	`<path d="M87.99 0H28v20h59.99a2 2 0 002-2V2a2 2 0 00-2-2z" fill="#007D9C"/>` +
	`<path d="M35 14v-3c0-2 2-3 4-3l1 0V14H35z" fill="#FAFAFA"/></svg>`

// TestPagerKittyBadges runs the pager on a README-style list of badges and
// checks that they are rasterized, drawn in the text flow, and rendered next
// to each other on one line, the way a browser renders them.
func TestPagerKittyBadges(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"release.svg": testBadgeSVG,
		"doc.svg":     testPathBadgeSVG,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mdPath := filepath.Join(dir, "test.md")
	body := "<p>\n" +
		"  <a href=\"https://example.com\"><img src=\"release.svg\" alt=\"release\"></a>\n" +
		"  <a href=\"https://example.com\"><img src=\"doc.svg\" alt=\"doc\"></a>\n" +
		"</p>\n"
	if err := os.WriteFile(mdPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		GlamourEnabled:  true,
		GlamourStyle:    "dark",
		GlamourMaxWidth: 80,
		Images:          true,
		ImageProtocol:   "kitty",
	}
	common := commonModel{cfg: cfg, styles: newStyles(true), width: 80, height: 24}
	common.imageProtocol = ansi.ImageProtocolKittyPlaceholders
	pager := newPagerModel(&common)
	pager.setSize(80, 24)
	pager.currentDocument = markdown{Note: "test.md", localPath: mdPath, Body: body}

	content, graphics, err := glamourRender(pager, body, false)
	if err != nil {
		t.Fatal(err)
	}

	// The two badges are rasterized and transmitted, and their placeholder
	// grids are placed on the same line, separated by a space, the way a
	// browser renders them.
	if len(graphics) != 4 {
		t.Errorf("expected transmit and place commands for both badges, got %d", len(graphics))
	}
	gridLines := 0
	for _, line := range strings.Split(content, "\n") {
		if !strings.ContainsRune(line, kittyPlaceholder) {
			continue
		}
		gridLines++

		grid := strings.TrimLeft(visibleText(line), " ")
		if !strings.HasPrefix(grid, string(kittyPlaceholder)) {
			t.Errorf("expected the line to start with a badge, got: %q", grid)
		}
		if got := strings.Count(grid, string(kittyPlaceholder)); got != 19 {
			t.Errorf("expected 19 placeholder cells for both badges, got %d", got)
		}
		if !strings.Contains(grid, string(kittyPlaceholder)+" ") {
			t.Errorf("expected a space between the badges, got: %q", grid)
		}
	}
	if gridLines != 1 {
		t.Fatalf("expected both badges on one line, got %d lines", gridLines)
	}
}
