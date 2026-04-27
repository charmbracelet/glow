package ui

import (
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glow/v2/utils"
	"github.com/charmbracelet/lipgloss"
)

type streamChunkMsg []byte
type streamErrMsg error

// StreamModel is a minimal Bubble Tea streaming renderer.
type StreamModel struct {
	content  string
	renderer *glamour.TermRenderer
	reader   io.Reader
}

// NewStreamModel creates a streaming markdown renderer.
func NewStreamModel(cfg Config) *StreamModel {
	r, _ := glamour.NewTermRenderer(
		glamour.WithColorProfile(lipgloss.ColorProfile()),
		utils.GlamourStyle(cfg.GlamourStyle, false),
		glamour.WithWordWrap(int(cfg.GlamourMaxWidth)),
		glamour.WithPreservedNewLines(),
	)

	return &StreamModel{
		renderer: r,
		reader:   os.Stdin,
	}
}

// Init starts the input streaming loop.
func (m *StreamModel) Init() tea.Cmd {
	return m.readChunk()
}

// readChunk reads the next input chunk from stdin.
func (m *StreamModel) readChunk() tea.Cmd {
	return func() tea.Msg {
		buf := make([]byte, 4096)

		n, err := m.reader.Read(buf)
		if err != nil {
			return streamErrMsg(err)
		}

		return streamChunkMsg(buf[:n])
	}
}

// Update processes incoming stream data and user input.
func (m *StreamModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case streamChunkMsg:
		m.content += string(msg)

		// continue streaming
		return m, m.readChunk()

	case streamErrMsg:
		return m, tea.Quit

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the current markdown content.
func (m *StreamModel) View() string {
	out, err := m.renderer.Render(m.content)
	if err != nil {
		return fmt.Sprintf("render error: %v", err)
	}
	return out
}
