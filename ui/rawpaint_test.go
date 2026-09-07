package ui

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestRawPainterPaint(t *testing.T) {
	var buf bytes.Buffer
	p := &rawPainter{w: &buf}

	stale := p.paint("stale frame")
	fresh := p.paint("fresh\nframe")

	fresh()
	stale()

	got := buf.String()
	if strings.Contains(got, "stale") {
		t.Errorf("stale frame was painted over a newer one: %q", got)
	}
	if !strings.Contains(got, "\x1b[?2026h\x1b[H\x1b[2Jfresh\r\nframe\x1b[?2026l") {
		t.Errorf("unexpected frame format: %q", got)
	}
}

func TestPaintModelRendersDocument(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("# Hello\n\nSome text here.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		GlamourEnabled:  true,
		GlamourStyle:    "dark",
		GlamourMaxWidth: 80,
		Path:            path,
	}

	var buf bytes.Buffer
	painter := &rawPainter{w: &buf}
	var pm tea.Model = paintModel{inner: newModel(cfg, ""), painter: painter}

	msgCh := make(chan tea.Msg, 64)
	var exec func(cmd tea.Cmd)
	exec = func(cmd tea.Cmd) {
		if cmd == nil {
			return
		}
		go func() {
			switch msg := cmd().(type) {
			case tea.BatchMsg:
				for _, c := range msg {
					exec(c)
				}
			case nil:
			default:
				msgCh <- msg
			}
		}()
	}

	exec(pm.Init())
	msgCh <- tea.WindowSizeMsg{Width: 80, Height: 24}

	deadline := time.Now().Add(3 * time.Second)
	for {
		// Frames are painted asynchronously by the paint command, so poll
		// the buffer instead of only checking after processed messages.
		painter.mu.Lock()
		done := strings.Contains(buf.String(), "Some text")
		painter.mu.Unlock()
		if done {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for content; buf=%q", buf.String())
		}

		select {
		case msg := <-msgCh:
			im, cmd := pm.Update(msg)
			pm = im
			exec(cmd)
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func TestPaintModelZeroSizeSubstitution(t *testing.T) {
	var buf bytes.Buffer
	painter := &rawPainter{w: &buf}
	pm := paintModel{inner: newModel(Config{}, "body"), painter: painter}

	im, _ := pm.Update(tea.WindowSizeMsg{Width: 0, Height: 0})
	m := im.(paintModel).inner.(model)
	if m.common.width != 80 || m.common.height != 24 {
		t.Errorf("expected fallback size 80x24, got %dx%d", m.common.width, m.common.height)
	}
}
