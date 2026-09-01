package utils

import (
	"testing"

	"charm.land/glamour/v2/ansi"
)

func TestProtocolFromEnvironment(t *testing.T) {
	// Clear the variables the detection looks at so tests don't depend on
	// the environment they run in.
	clearEnv := func(env map[string]string) {
		for _, k := range []string{"TERM", "TERM_PROGRAM", "KITTY_WINDOW_ID", "GHOSTTY_RESOURCES_DIR"} {
			if _, ok := env[k]; !ok {
				t.Setenv(k, "")
			}
		}
	}

	tests := []struct {
		name     string
		env      map[string]string
		protocol ansi.ImageProtocol
		known    bool
	}{
		{
			name:     "kitty via KITTY_WINDOW_ID",
			env:      map[string]string{"KITTY_WINDOW_ID": "1"},
			protocol: ansi.ImageProtocolKitty,
			known:    true,
		},
		{
			name:     "kitty via TERM",
			env:      map[string]string{"TERM": "xterm-kitty"},
			protocol: ansi.ImageProtocolKitty,
			known:    true,
		},
		{
			name:     "ghostty via TERM",
			env:      map[string]string{"TERM": "ghostty"},
			protocol: ansi.ImageProtocolKitty,
			known:    true,
		},
		{
			name:     "wezterm via TERM_PROGRAM",
			env:      map[string]string{"TERM": "xterm-256color", "TERM_PROGRAM": "WezTerm"},
			protocol: ansi.ImageProtocolKitty,
			known:    true,
		},
		{
			name:     "foot",
			env:      map[string]string{"TERM": "foot-extra"},
			protocol: ansi.ImageProtocolSixel,
			known:    true,
		},
		{
			name:     "unknown terminal",
			env:      map[string]string{"TERM": "xterm-256color"},
			protocol: ansi.ImageProtocolNone,
			known:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clearEnv(tc.env)
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			protocol, known := protocolFromEnvironment()
			if protocol != tc.protocol {
				t.Errorf("expected protocol %v, got %v", tc.protocol, protocol)
			}
			if known != tc.known {
				t.Errorf("expected known %v, got %v", tc.known, known)
			}
		})
	}
}

func TestParseQueryResponse(t *testing.T) {
	tests := []struct {
		name     string
		resp     string
		hasKitty bool
		hasSixel bool
	}{
		{
			name:     "kitty",
			resp:     "\x1b_Gi=31;OK\x1b\\\x1b[?62;c",
			hasKitty: true,
		},
		{
			name:     "kitty with other ids",
			resp:     "\x1b_Gi=1;OK\x1b\\\x1b_Gi=31;OK\x1b\\\x1b[?62;c",
			hasKitty: true,
		},
		{
			name:     "sixel only",
			resp:     "\x1b[?62;4;6;9;15;16;17;18;21c",
			hasSixel: true,
		},
		{
			name: "no graphics support",
			resp: "\x1b[?1;2c",
		},
		{
			name: "kitty error response",
			resp: "\x1b_Gi=31;EN\x1b\\\x1b[?62;c",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hasKitty, hasSixel := parseQueryResponse(tc.resp)
			if hasKitty != tc.hasKitty {
				t.Errorf("expected kitty %v, got %v", tc.hasKitty, hasKitty)
			}
			if hasSixel != tc.hasSixel {
				t.Errorf("expected sixel %v, got %v", tc.hasSixel, hasSixel)
			}
		})
	}
}

func TestHasDA1Response(t *testing.T) {
	if !hasDA1Response([]byte("\x1b[?62;4c")) {
		t.Error("expected DA1 response to be detected")
	}
	if hasDA1Response([]byte("\x1b_Gi=31;OK\x1b\\")) {
		t.Error("expected no DA1 response to be detected")
	}
	if hasDA1Response([]byte("garbage")) {
		t.Error("expected no DA1 response to be detected")
	}
}

func TestFileBaseURL(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		// The results must be identical on all platforms, so that relative
		// image references in documents resolve everywhere.
		{"/tmp/docs/test.md", "file:///tmp/docs/"},
		{"C:/Users/x/test.md", "file:///C:/Users/x/"},
	}

	for _, tc := range tests {
		if got := FileBaseURL(tc.path); got != tc.want {
			t.Errorf("FileBaseURL(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestGraphicsQuery(t *testing.T) {
	q := graphicsQuery()
	if want := "\x1b_Gf=24,i=31,s=1,v=1,a=q;AAAA\x1b\\"; q != want+"\x1b[c" {
		t.Errorf("unexpected query %q", q)
	}
}
