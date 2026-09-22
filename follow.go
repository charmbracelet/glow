package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glow/v2/utils"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
)

const (
	// followDebounce coalesces the burst of write events a single save
	// produces before reading the file.
	followDebounce = 100 * time.Millisecond
	// followIdleFlush renders a trailing block that has no terminating blank
	// line yet, once the file has been quiet for this long.
	followIdleFlush = 8 * time.Second
)

// fenceState tracks whether a scan position is inside a fenced code block.
type fenceState struct {
	open   bool
	char   byte
	length int
	line   string
}

// trimIndent removes up to three leading spaces. Four or more means an
// indented code block, which cannot open or close a fence.
func trimIndent(line []byte) []byte {
	for i := 0; i < 3 && len(line) > 0 && line[0] == ' '; i++ {
		line = line[1:]
	}
	if len(line) > 0 && line[0] == ' ' {
		return nil
	}
	return line
}

// openingFence reports whether line opens a fenced code block, returning the
// fence character and length.
func openingFence(line []byte) (byte, int, bool) {
	trimmed := trimIndent(line)
	if len(trimmed) < 3 {
		return 0, 0, false
	}
	ch := trimmed[0]
	if ch != '`' && ch != '~' {
		return 0, 0, false
	}
	n := 0
	for n < len(trimmed) && trimmed[n] == ch {
		n++
	}
	if n < 3 {
		return 0, 0, false
	}
	// the info string of a backtick fence cannot contain backticks
	if ch == '`' && bytes.IndexByte(trimmed[n:], '`') >= 0 {
		return 0, 0, false
	}
	return ch, n, true
}

// closesFence reports whether line closes the fence described by st.
func closesFence(line []byte, st fenceState) bool {
	trimmed := trimIndent(line)
	n := 0
	for n < len(trimmed) && trimmed[n] == st.char {
		n++
	}
	if n < st.length {
		return false
	}
	return len(bytes.TrimSpace(trimmed[n:])) == 0
}

// lastBoundary scans the complete lines of data, starting in fence state st,
// and returns the offset just past the last blank line that sits outside a
// fenced code block (0 if there is none), along with the fence state after
// the last complete line.
func lastBoundary(data []byte, st fenceState) (int, fenceState) {
	boundary, pos := 0, 0
	for {
		nl := bytes.IndexByte(data[pos:], '\n')
		if nl < 0 {
			break
		}
		line := data[pos : pos+nl]
		pos += nl + 1
		switch {
		case st.open:
			if closesFence(line, st) {
				st = fenceState{}
			}
		default:
			if ch, n, ok := openingFence(line); ok {
				st = fenceState{open: true, char: ch, length: n, line: string(line)}
			} else if len(bytes.TrimSpace(line)) == 0 {
				boundary = pos
			}
		}
	}
	return boundary, st
}

// follower tails a file and renders appended markdown in block-safe chunks.
type follower struct {
	path   string
	w      io.Writer
	render func(string) (string, error)
	isCode bool
	ext    string

	offset    int64
	rewritten bool
	pending   []byte
	st        fenceState
	// reopenLine is the opening fence line of a code block that a forced
	// flush rendered before its closing fence arrived; it is prepended to
	// the next chunk so the remainder still renders as code.
	reopenLine string
}

// emit renders a chunk and appends it to the output.
func (f *follower) emit(chunk []byte) error {
	content := string(chunk)
	if f.reopenLine != "" && !f.isCode {
		content = f.reopenLine + "\n" + content
	}
	if f.isCode {
		content = utils.WrapCodeBlock(content, f.ext)
	}
	out, err := f.render(content)
	if err != nil {
		return fmt.Errorf("unable to render markdown: %w", err)
	}
	out = strings.Trim(out, "\n")
	if out == "" {
		return nil
	}
	if _, err := fmt.Fprintf(f.w, "%s\n\n", out); err != nil {
		return fmt.Errorf("unable to write to writer: %w", err)
	}
	return nil
}

// flushComplete renders everything up to the last safe block boundary and
// keeps the remainder pending.
func (f *follower) flushComplete() error {
	var boundary int
	if f.isCode {
		boundary = bytes.LastIndexByte(f.pending, '\n') + 1
	} else {
		boundary, _ = lastBoundary(f.pending, f.st)
	}
	if boundary <= 0 {
		return nil
	}
	chunk := f.pending[:boundary]
	f.pending = append([]byte(nil), f.pending[boundary:]...)
	err := f.emit(chunk)
	// a boundary is always outside a fence, so any reopened fence is closed
	f.st = fenceState{}
	f.reopenLine = ""
	return err
}

// flushAll renders everything pending, including a trailing block that has no
// terminating blank line yet. If that leaves a fence open, the next chunk
// reopens it.
func (f *follower) flushAll() error {
	if len(f.pending) == 0 {
		return nil
	}
	chunk := f.pending
	f.pending = nil
	if !f.isCode {
		_, st := lastBoundary(chunk, f.st)
		err := f.emit(chunk)
		f.st = st
		if st.open {
			f.reopenLine = st.line
		} else {
			f.reopenLine = ""
		}
		return err
	}
	return f.emit(chunk)
}

// renderWhole renders the file from the beginning, as on startup or after the
// file was rewritten.
func (f *follower) renderWhole() error {
	raw, err := os.ReadFile(f.path)
	if err != nil {
		return fmt.Errorf("unable to read file: %w", err)
	}
	f.offset = int64(len(raw))
	f.pending = utils.RemoveFrontmatter(raw)
	f.st = fenceState{}
	f.reopenLine = ""
	return f.flushAll()
}

// printDivider separates a rewritten file's fresh render from prior output.
func (f *follower) printDivider() error {
	out, err := f.render("---")
	if err != nil {
		return fmt.Errorf("unable to render markdown: %w", err)
	}
	if out = strings.Trim(out, "\n"); out == "" {
		return nil
	}
	if _, err := fmt.Fprintf(f.w, "%s\n\n", out); err != nil {
		return fmt.Errorf("unable to write to writer: %w", err)
	}
	return nil
}

// readNew ingests whatever the file gained since the last read. A shrunken or
// replaced file is treated like tail -f treats truncation: print a divider
// and render the whole file again.
func (f *follower) readNew() error {
	fi, err := os.Stat(f.path)
	if err != nil {
		// the file is momentarily gone (e.g. mid atomic save); wait for it
		// to reappear
		return nil
	}
	if f.rewritten || fi.Size() < f.offset {
		f.rewritten = false
		if err := f.printDivider(); err != nil {
			return err
		}
		return f.renderWhole()
	}
	if fi.Size() == f.offset {
		return nil
	}
	file, err := os.Open(f.path)
	if err != nil {
		return fmt.Errorf("unable to read file: %w", err)
	}
	defer file.Close() //nolint:errcheck
	if _, err := file.Seek(f.offset, io.SeekStart); err != nil {
		return fmt.Errorf("unable to read file: %w", err)
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("unable to read file: %w", err)
	}
	f.offset += int64(len(data))
	f.pending = append(f.pending, data...)
	return f.flushComplete()
}

// runFollow renders path and then appends newly written content as it
// arrives, until interrupted.
func runFollow(path string, w io.Writer) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("unable to resolve path: %w", err)
	}

	isCode := !utils.IsMarkdownFile(abs)
	r, err := glamour.NewTermRenderer(
		glamour.WithColorProfile(lipgloss.ColorProfile()),
		utils.GlamourStyle(style, isCode),
		glamour.WithWordWrap(int(width)),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return fmt.Errorf("unable to create renderer: %w", err)
	}

	f := &follower{
		path:   abs,
		w:      w,
		render: r.Render,
		isCode: isCode,
		ext:    filepath.Ext(abs),
	}
	if err := f.renderWhole(); err != nil {
		return err
	}

	// watch the parent directory rather than the file itself so the watch
	// survives editors that save atomically via rename
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("unable to watch file: %w", err)
	}
	defer watcher.Close() //nolint:errcheck
	if err := watcher.Add(filepath.Dir(abs)); err != nil {
		return fmt.Errorf("unable to watch file: %w", err)
	}

	debounce := time.NewTimer(followDebounce)
	debounce.Stop()
	idle := time.NewTimer(followIdleFlush)
	idle.Stop()

	for {
		select {
		case ev, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if filepath.Clean(ev.Name) != abs {
				continue
			}
			if ev.Op&fsnotify.Create != 0 {
				f.rewritten = true
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				debounce.Reset(followDebounce)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			return fmt.Errorf("unable to watch file: %w", err)
		case <-debounce.C:
			if err := f.readNew(); err != nil {
				return err
			}
			if len(f.pending) > 0 {
				idle.Reset(followIdleFlush)
			} else {
				idle.Stop()
			}
		case <-idle.C:
			if err := f.flushAll(); err != nil {
				return err
			}
		}
	}
}
