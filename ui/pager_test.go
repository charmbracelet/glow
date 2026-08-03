package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestPager(t *testing.T, highPerformance bool, lines int) pagerModel {
	t.Helper()

	config = Config{HighPerformancePager: highPerformance}
	common := commonModel{cfg: config, width: 80, height: 24}
	m := newPagerModel(&common)
	t.Cleanup(func() {
		if m.watcher != nil {
			_ = m.watcher.Close()
		}
	})
	m.setSize(common.width, common.height)
	m.setContent(strings.Repeat("line\n", lines))
	return m
}

func key(k string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

// The viewport binds some of the same keys the pager does. If both act on one
// key press the viewport moves twice while only one of the two movements is
// drawn, which leaves the screen showing lines from two scroll positions.
func TestPagerScrollsOncePerKeyPress(t *testing.T) {
	for _, tc := range []struct {
		key  string
		want int
	}{
		{"j", 1},
		{"d", 11}, // half a page of the 23 line viewport
		{"f", 23}, // a full page
	} {
		t.Run(tc.key, func(t *testing.T) {
			m := newTestPager(t, true, 200)
			m, _ = m.update(key(tc.key))
			if m.viewport.YOffset != tc.want {
				t.Errorf("scrolled to line %d, want %d", m.viewport.YOffset, tc.want)
			}
		})
	}
}

// High performance render commands write to the terminal directly, so only one
// may be in flight at a time: Bubble Tea runs commands in their own goroutines
// and two of them can reach the renderer out of order.
func TestPagerKeepsOneRenderInFlight(t *testing.T) {
	m := newTestPager(t, true, 200)

	m, cmd := m.update(key("j"))
	if cmd == nil {
		t.Fatal("no render command for the first scroll")
	}
	if !m.renderPending {
		t.Fatal("first scroll didn't mark a render as pending")
	}

	m, cmd = m.update(key("j"))
	if cmd != nil {
		t.Error("issued a second render command while the first was in flight")
	}
	if !m.renderStale {
		t.Error("dropped render wasn't recorded as stale")
	}
	if m.viewport.YOffset != 2 {
		t.Errorf("scrolled to line %d, want 2", m.viewport.YOffset)
	}

	// Once the in-flight command lands the dropped movement is redrawn.
	m, cmd = m.update(renderedMsg{m.renderSeq})
	if cmd == nil {
		t.Error("no resync after a dropped render")
	}
	if m.renderStale {
		t.Error("still stale after resyncing")
	}
}

// A render command left over from a document we've since closed must not open
// the gate for the document we're showing now.
func TestPagerIgnoresStaleRenderedMsg(t *testing.T) {
	m := newTestPager(t, true, 200)

	m, _ = m.update(key("j"))
	seq := m.renderSeq
	m.unload()
	m.setContent(strings.Repeat("line\n", 200))

	m, _ = m.update(key("j"))
	m, _ = m.update(renderedMsg{seq})
	if !m.renderPending {
		t.Error("a stale renderedMsg cleared the pending render")
	}
}

// Near the top and bottom of the document the viewport moves by less than it
// was asked to, so the lines it hands back don't match how far it went.
func TestPagerRedrawsInFullAtDocumentEdges(t *testing.T) {
	m := newTestPager(t, true, 30)

	// 31 lines of content in a 23 line viewport: only 8 lines to scroll.
	m, _ = m.update(key("d"))
	if m.viewport.YOffset != 8 || !m.viewport.AtBottom() {
		t.Fatalf("scrolled to line %d, want the bottom at line 8", m.viewport.YOffset)
	}
	if !m.renderPending {
		t.Fatal("no render for a partial scroll")
	}

	// Nothing left to scroll: no movement, no render.
	m.renderPending = false
	m, cmd := m.update(key("d"))
	if cmd != nil || m.renderPending {
		t.Error("rendered a scroll that didn't move the viewport")
	}
}

func TestPagerScrollsWithoutHighPerformanceRendering(t *testing.T) {
	m := newTestPager(t, false, 200)

	m, cmd := m.update(key("d"))
	if cmd != nil {
		t.Error("issued a high performance render command with the feature off")
	}
	if m.viewport.YOffset != 11 {
		t.Errorf("scrolled to line %d, want 11", m.viewport.YOffset)
	}
}
