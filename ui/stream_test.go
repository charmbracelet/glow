package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/glow/v2/utils"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// fakeReader simulates streaming stdin deterministically.
type fakeReader struct {
	data []byte
	i    int
}

func (r *fakeReader) Read(p []byte) (int, error) {
	if r.i >= len(r.data) {
		time.Sleep(10 * time.Millisecond)
		return 0, ioEOF{}
	}

	n := copy(p, r.data[r.i:])
	r.i += n
	return n, nil
}

type ioEOF struct{}

func (ioEOF) Error() string { return "EOF" }
func TestStreamModel_AccumulatesChunks(t *testing.T) {
	input := []byte("hello ")
	input = append(input, []byte("world")...)

	cfg := Config{
		GlamourStyle:    "auto",
		GlamourMaxWidth: 80,
	}

	r, _ := glamour.NewTermRenderer(
		glamour.WithColorProfile(lipgloss.ColorProfile()),
		utils.GlamourStyle(cfg.GlamourStyle, false),
		glamour.WithWordWrap(int(cfg.GlamourMaxWidth)),
		glamour.WithPreservedNewLines(),
	)

	m := &StreamModel{
		content:  "",
		renderer: r,
		reader:   &fakeReader{data: input},
	}

	// Init should start streaming
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected init command")
	}

	// simulate stream loop manually
	for i := 0; i < 10; i++ {
		msg := cmd()

		switch v := msg.(type) {
		case streamChunkMsg:
			_, next := m.Update(v)
			cmd = next

		case streamErrMsg:
			return // expected EOF
		}
	}

	if !strings.Contains(m.content, "hello") || !strings.Contains(m.content, "world") {
		t.Fatalf("expected streamed content, got: %q", m.content)
	}
}
