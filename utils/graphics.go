// Package utils provides utility functions.
package utils

import (
	"bytes"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"charm.land/glamour/v2/ansi"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
	"golang.org/x/term"
)

// Image protocol names as accepted by the GLOW_IMAGE_PROTOCOL environment
// variable.
const (
	ImageProtocolEnv   = "GLOW_IMAGE_PROTOCOL"
	ImageProtocolAuto  = "auto"
	ImageProtocolKitty = "kitty"
	ImageProtocolSixel = "sixel"
	ImageProtocolNone  = "none"
)

// queryTimeout is how long DetectImageProtocol waits for terminal query
// responses before giving up. Terminals respond within a few milliseconds,
// so this only matters for terminals that don't respond at all.
const queryTimeout = 150 * time.Millisecond

// kittyQueryID is the image id used for the kitty graphics query. The
// terminal echoes it back in its response, and it must not collide with real
// image ids. KittyGraphicsOK reports whether a KittyGraphicsEvent is a
// successful response to [GraphicsQuery].
const kittyQueryID = 31

// DetectImageProtocol returns the graphics protocol the terminal attached to
// os.Stdout and os.Stdin supports.
//
// The GLOW_IMAGE_PROTOCOL environment variable can be set to "kitty",
// "sixel", "auto", or "none" to force a protocol and skip detection. With
// "auto" (the default), known terminals are detected via environment
// variables, and everything else is detected by querying the terminal: a
// kitty graphics query followed by a request for the primary device
// attributes, which also reports sixel support. Terminals that answer
// neither are assumed to not support graphics.
func DetectImageProtocol() ansi.ImageProtocol {
	switch p := os.Getenv(ImageProtocolEnv); p {
	case ImageProtocolKitty:
		return ansi.ImageProtocolKitty
	case ImageProtocolSixel:
		return ansi.ImageProtocolSixel
	case ImageProtocolNone:
		return ansi.ImageProtocolNone
	}

	out, in := os.Stdout, os.Stdin
	if !term.IsTerminal(int(out.Fd())) {
		// Piping the output elsewhere: don't emit any graphics sequences.
		return ansi.ImageProtocolNone
	}

	// Terminals we can identify by environment alone. This keeps them fast,
	// as they don't need to be queried.
	if protocol, ok := protocolFromEnvironment(); ok {
		return protocol
	}

	if !term.IsTerminal(int(in.Fd())) {
		// No way to read query responses, e.g. when rendering from stdin.
		return ansi.ImageProtocolNone
	}

	resp, err := queryTerminal(out, in, graphicsQuery(), queryTimeout)
	if err != nil {
		return ansi.ImageProtocolNone
	}
	if hasKitty, hasSixel := parseQueryResponse(resp); hasKitty {
		return ansi.ImageProtocolKitty
	} else if hasSixel {
		return ansi.ImageProtocolSixel
	}
	return ansi.ImageProtocolNone
}

// protocolFromEnvironment reports the graphics protocol of terminals that
// identify themselves via environment variables, and whether the terminal
// is known.
func protocolFromEnvironment() (ansi.ImageProtocol, bool) {
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return ansi.ImageProtocolKitty, true
	}
	termVar := os.Getenv("TERM")
	switch {
	case termVar == "xterm-kitty" || strings.HasPrefix(termVar, "kitty"):
		return ansi.ImageProtocolKitty, true
	case strings.Contains(termVar, "ghostty"), os.Getenv("GHOSTTY_RESOURCES_DIR") != "":
		return ansi.ImageProtocolKitty, true
	case os.Getenv("TERM_PROGRAM") == "WezTerm":
		return ansi.ImageProtocolKitty, true
	case strings.HasPrefix(termVar, "foot"):
		// foot supports sixel but not the kitty graphics protocol.
		return ansi.ImageProtocolSixel, true
	}
	return ansi.ImageProtocolNone, false
}

// GraphicsQuery returns a kitty graphics query that can be sent to the
// terminal to detect support for the kitty graphics protocol. Send it via
// tea.Raw and check the response with KittyGraphicsOK. It is the same query
// DetectImageProtocol uses.
// See https://sw.kovidgoyal.net/kitty/graphics-protocol/#checking-for-support
func GraphicsQuery() string {
	return graphicsQuery()
}

// KittyGraphicsOK reports whether the given kitty graphics event is a
// successful response to [GraphicsQuery].
func KittyGraphicsOK(optionsID int, payload []byte) bool {
	return optionsID == kittyQueryID && bytes.HasPrefix(payload, []byte("OK"))
}

// FileBaseURL returns a file:// URL for the directory containing the given
// file path, suitable as a glamour base URL, so that relative image
// references resolve correctly on all platforms.
func FileBaseURL(path string) string {
	dir := filepath.ToSlash(filepath.Dir(path))
	u := url.URL{Scheme: "file", Path: "/" + strings.TrimPrefix(dir, "/")}
	return u.String() + "/"
}

// graphicsQuery returns a kitty graphics query followed by a request for the
// primary device attributes. Terminals supporting the kitty graphics
// protocol must answer the graphics query immediately, and every terminal
// answers the device attributes request, which terminates the wait and
// reports sixel support.
func graphicsQuery() string {
	opts := kitty.Options{
		Action: kitty.Query,
		ID:     kittyQueryID,
		// One 1x1 RGB pixel, base64 encoded as the payload.
		ImageWidth:   1,
		ImageHeight:  1,
		Transmission: kitty.Direct,
		Format:       kitty.RGB,
	}
	return xansi.KittyGraphics([]byte("AAAA"), opts.Options()...) +
		xansi.RequestPrimaryDeviceAttributes
}

// queryTerminal writes query to out, puts in into raw mode, and reads back
// the terminal's response until the primary device attributes response
// arrives. It gives up after the given timeout.
func queryTerminal(out, in *os.File, query string, timeout time.Duration) (string, error) {
	fd := int(in.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer term.Restore(fd, oldState) //nolint:errcheck

	if _, err := out.WriteString(query); err != nil {
		return "", err
	}

	type result struct {
		response string
		err      error
	}
	done := make(chan result, 1)

	go func() {
		var buf bytes.Buffer
		b := make([]byte, 256)
		for {
			n, err := in.Read(b)
			if n > 0 {
				buf.Write(b[:n])
				// The device attributes response terminates the query.
				if hasDA1Response(buf.Bytes()) {
					done <- result{response: buf.String()}
					return
				}
			}
			if err != nil {
				done <- result{err: err}
				return
			}
		}
	}()

	select {
	case r := <-done:
		return r.response, r.err
	case <-time.After(timeout):
		return "", nil
	}
}

// hasDA1Response reports whether data contains a primary device attributes
// response, i.e. ESC [ ? ... c.
func hasDA1Response(data []byte) bool {
	for i := 0; i+3 < len(data); i++ {
		if data[i] == 0x1b && data[i+1] == '[' && data[i+2] == '?' &&
			bytes.ContainsRune(data[i+3:], 'c') {
			return true
		}
	}
	return false
}

// parseQueryResponse reports whether the terminal's query response indicates
// support for the kitty graphics protocol and sixel graphics.
func parseQueryResponse(resp string) (hasKitty, hasSixel bool) {
	// The kitty graphics protocol responds with _Gi=<id>;OK.
	for _, part := range strings.Split(resp, "\x1b_G") {
		options, payload, _ := strings.Cut(part, ";")
		if strings.Contains(options, "i="+strconv.Itoa(kittyQueryID)) &&
			strings.HasPrefix(payload, "OK") {
			hasKitty = true
		}
	}
	// DA1 responses are ESC [ ? <attrs> c, where attribute 4 indicates sixel
	// support.
	for _, part := range strings.Split(resp, "\x1b[?") {
		attrs, ok := strings.CutSuffix(part, "c")
		if !ok {
			continue
		}
		for _, attr := range strings.Split(attrs, ";") {
			if attr == "4" {
				hasSixel = true
			}
		}
	}
	return hasKitty, hasSixel
}
