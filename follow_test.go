package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestLastBoundary(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		boundary int
		open     bool
	}{
		{
			name:     "no blank line",
			data:     "a paragraph\nstill going",
			boundary: 0,
		},
		{
			name:     "blank line ends paragraph",
			data:     "para\n\nmore",
			boundary: len("para\n\n"),
		},
		{
			name:     "blank line inside fence is not a boundary",
			data:     "```\ncode\n\nmore code\n",
			boundary: 0,
			open:     true,
		},
		{
			name:     "boundary after closed fence",
			data:     "```go\ncode\n```\n\ntail",
			boundary: len("```go\ncode\n```\n\n"),
		},
		{
			name:     "short closing fence does not close",
			data:     "````\n```\n\n",
			boundary: 0,
			open:     true,
		},
		{
			name:     "tilde fence ignores backticks",
			data:     "~~~\n```\n\n~~~\n\n",
			boundary: len("~~~\n```\n\n~~~\n\n"),
		},
		{
			name:     "indented fence marker is code not fence",
			data:     "    ```\n\nafter",
			boundary: len("    ```\n\n"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			boundary, st := lastBoundary([]byte(tc.data), fenceState{})
			if boundary != tc.boundary {
				t.Errorf("boundary = %d, want %d", boundary, tc.boundary)
			}
			if st.open != tc.open {
				t.Errorf("fence open = %v, want %v", st.open, tc.open)
			}
		})
	}
}

// newTestFollower returns a follower whose renderer is the identity function,
// so output inspection sees the exact markdown each chunk rendered.
func newTestFollower(isCode bool) (*follower, *bytes.Buffer) {
	var buf bytes.Buffer
	return &follower{
		w:      &buf,
		render: func(s string) (string, error) { return s, nil },
		isCode: isCode,
		ext:    ".txt",
	}, &buf
}

func TestFlushCompleteHoldsPartialBlock(t *testing.T) {
	f, buf := newTestFollower(false)
	f.pending = []byte("done\n\npartial")
	if err := f.flushComplete(); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "done\n\n" {
		t.Errorf("output = %q, want %q", got, "done\n\n")
	}
	if got := string(f.pending); got != "partial" {
		t.Errorf("pending = %q, want %q", got, "partial")
	}
}

func TestFlushCompleteWaitsForFenceClose(t *testing.T) {
	f, buf := newTestFollower(false)
	f.pending = []byte("```\ncode\n\n")
	if err := f.flushComplete(); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("output = %q, want empty while fence is open", buf.String())
	}

	f.pending = append(f.pending, []byte("```\n\n")...)
	if err := f.flushComplete(); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); !strings.Contains(got, "code") {
		t.Errorf("output = %q, want the closed fence rendered", got)
	}
}

func TestFlushAllReopensFence(t *testing.T) {
	f, buf := newTestFollower(false)
	f.pending = []byte("```go\nfirst half\n")
	if err := f.flushAll(); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); !strings.Contains(got, "first half") {
		t.Errorf("output = %q, want forced flush of open fence", got)
	}
	if f.reopenLine != "```go" {
		t.Errorf("reopenLine = %q, want %q", f.reopenLine, "```go")
	}

	buf.Reset()
	f.pending = []byte("second half\n```\n\n")
	if err := f.flushComplete(); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.HasPrefix(got, "```go\n") {
		t.Errorf("output = %q, want chunk prefixed with reopened fence", got)
	}
	if f.reopenLine != "" || f.st.open {
		t.Errorf("fence state not cleared after close: reopen=%q open=%v", f.reopenLine, f.st.open)
	}
}

func TestFlushCompletePlainTextFlushesWholeLines(t *testing.T) {
	f, buf := newTestFollower(true)
	f.pending = []byte("one\ntwo\npart")
	if err := f.flushComplete(); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, "one\ntwo") {
		t.Errorf("output = %q, want complete lines flushed", got)
	}
	if strings.Contains(got, "part") {
		t.Errorf("output = %q, must not contain the partial line", got)
	}
	if got := string(f.pending); got != "part" {
		t.Errorf("pending = %q, want %q", got, "part")
	}
}
