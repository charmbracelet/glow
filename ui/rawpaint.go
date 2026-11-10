package ui

import (
	"io"
	"os"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/term"
)

// rawPainter paints frames directly to the terminal. It is used when kitty
// text sizing support is detected: bubbletea's default renderer parses each
// frame into a cell buffer which discards OSC 66 sequences along with their
// payload text, so the program runs with [tea.WithoutRenderer] and this
// painter drives the terminal instead.
type rawPainter struct {
	w     io.Writer
	tty   *os.File
	state *term.State
	mouse bool
	done  chan struct{}

	mu      sync.Mutex
	gen     int
	painted int
}

func newRawPainter(w io.Writer, tty *os.File, state *term.State, mouse bool) *rawPainter {
	return &rawPainter{w: w, tty: tty, state: state, mouse: mouse, done: make(chan struct{})}
}

func (p *rawPainter) windowSize() tea.WindowSizeMsg {
	if p.tty != nil {
		if w, h, err := term.GetSize(int(p.tty.Fd())); err == nil && w > 0 && h > 0 {
			return tea.WindowSizeMsg{Width: w, Height: h}
		}
	}
	return tea.WindowSizeMsg{Width: 80, Height: 24}
}

func (p *rawPainter) start() {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, _ = io.WriteString(p.w, "\x1b[?1049h\x1b[?25l")
	if p.mouse {
		_, _ = io.WriteString(p.w, "\x1b[?1002h\x1b[?1006h")
	}
}

func (p *rawPainter) stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.mouse {
		_, _ = io.WriteString(p.w, "\x1b[?1002l\x1b[?1006l")
	}
	_, _ = io.WriteString(p.w, "\x1b[?25h\x1b[?1049l")
	if p.tty != nil {
		if p.state != nil {
			_ = term.Restore(int(p.tty.Fd()), p.state)
		}
		_ = p.tty.Close()
	}
	select {
	case <-p.done:
	default:
		close(p.done)
	}
}

// watchResize sends the initial window size to the program and reports
// terminal resizes. Without a renderer bubbletea never initializes the tty,
// so its own size reporting never happens.
func (p *rawPainter) watchResize(prog *tea.Program) {
	ch := make(chan os.Signal, 1)
	winchNotify(ch)
	go func() {
		prog.Send(p.windowSize())
		for {
			select {
			case <-p.done:
				winchStop(ch)
				return
			case <-ch:
				prog.Send(p.windowSize())
			}
		}
	}()
}

func (p *rawPainter) paint(frame string) tea.Cmd {
	p.mu.Lock()
	p.gen++
	gen := p.gen
	p.mu.Unlock()

	return func() tea.Msg {
		p.mu.Lock()
		defer p.mu.Unlock()
		if gen <= p.painted {
			return nil
		}
		p.painted = gen

		var b strings.Builder
		b.WriteString("\x1b[?2026h\x1b[H\x1b[2J")
		b.WriteString(strings.ReplaceAll(frame, "\n", "\r\n"))
		b.WriteString("\x1b[?2026l")
		_, _ = io.WriteString(p.w, b.String())
		return nil
	}
}

// paintModel wraps the root model, painting a frame after every processed
// message. Bubbletea's renderer never calls View when it is disabled, so the
// wrapper drives painting itself.
type paintModel struct {
	inner   tea.Model
	painter *rawPainter
}

func (m paintModel) Init() tea.Cmd {
	m.painter.start()
	return tea.Batch(m.inner.Init(), m.painter.paint(m.inner.View().Content))
}

func (m paintModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok && (ws.Width <= 0 || ws.Height <= 0) {
		msg = m.painter.windowSize()
	}
	inner, cmd := m.inner.Update(msg)
	m.inner = inner
	return m, tea.Batch(cmd, m.painter.paint(inner.View().Content))
}

func (m paintModel) View() tea.View {
	return m.inner.View()
}
