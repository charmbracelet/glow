package utils

import (
	"strings"

	"charm.land/glow/v3/mermaid"
)

// ReplaceMermaidBlocks renders fenced ```mermaid blocks as terminal
// diagrams. Blocks that fail to parse are left untouched.
func ReplaceMermaidBlocks(md string) string {
	lines := strings.Split(md, "\n")
	var out []string
	for i := 0; i < len(lines); {
		kind, info, ok := openFence(lines[i])
		if !ok {
			out = append(out, lines[i])
			i++
			continue
		}
		// find the closing fence (or EOF)
		j := i + 1
		for ; j < len(lines); j++ {
			if closesFence(lines[j], kind) {
				break
			}
		}
		if info == "mermaid" && j < len(lines) {
			if art, err := mermaid.Render(strings.Join(lines[i+1:j], "\n")); err == nil {
				out = append(out, "```")
				out = append(out, strings.Split(art, "\n")...)
				out = append(out, "```")
				i = j + 1
				continue
			}
		}
		out = append(out, lines[i:min(j+1, len(lines))]...)
		i = j + 1
	}
	return strings.Join(out, "\n")
}

// openFence reports whether line opens a code fence, returning the fence
// character, the trimmed info string, and whether it is a fence at all.
// ponytail: fences indented >3 spaces (deep list nesting) are not detected.
func openFence(line string) (byte, string, bool) {
	indent := 0
	for indent < len(line) && line[indent] == ' ' {
		indent++
	}
	if indent > 3 {
		return 0, "", false
	}
	t := line[indent:]
	if len(t) < 3 || t[0] != '`' && t[0] != '~' {
		return 0, "", false
	}
	kind := t[0]
	n := 0
	for n < len(t) && t[n] == kind {
		n++
	}
	if n < 3 {
		return 0, "", false
	}
	return kind, strings.TrimSpace(t[n:]), true
}

// closesFence reports whether line closes a fence of the given kind.
func closesFence(line string, kind byte) bool {
	t := strings.TrimLeft(line, " ")
	return strings.HasPrefix(t, strings.Repeat(string(kind), 3))
}
