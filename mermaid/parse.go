// Package mermaid renders a subset of mermaid flowchart diagrams as Unicode
// box-drawing art suitable for terminal display.
//
// Supported syntax: flowchart/graph TD|TB|BT|LR|RL, node shapes
// [rect] (round) ((circle)) {diamond} {{hexagon}} ([stadium]) >flag],
// quoted and unquoted labels, <br/> line breaks, edges --> --- -.-> -.- ==>
// === (arrow style preserved, dotted/thick rendered solid), edge labels via
// |label| and -- label --> forms, chained statements, ";" separators, "%%"
// comments and classDef/class/style/linkStyle/click statements (ignored).
//
// Anything else is rejected and left for the caller to render verbatim.
package mermaid

import (
	"fmt"
	"regexp"
	"strings"
)

// node is a diagram node.
type node struct {
	label string
	round bool // rounded corners (any non-rectangular shape)
}

// edge is a directed connection between two nodes.
type edge struct {
	from, to string
	label    string
	arrow    bool // has an arrowhead
	dotted   bool
	thick    bool
}

type graph struct {
	dir   string // TD, TB, BT, LR or RL (as written)
	order []string
	nodes map[string]node
	edges []edge
}

var headerRe = regexp.MustCompile(`^(?:flowchart|graph)\s*(TD|TB|BT|LR|RL)?\s*$`)

// kwEnd marks the end of a subgraph block.
const kwEnd = "end"

// parse parses mermaid flowchart source into a graph.
func parse(src string) (*graph, error) {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")

	g := &graph{dir: "TD", nodes: map[string]node{}}

	// header: first significant line must declare a flowchart/graph
	i, found := 0, false
	for ; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if t == "" || strings.HasPrefix(t, "%%") {
			continue
		}
		m := headerRe.FindStringSubmatch(t)
		if m == nil {
			return nil, fmt.Errorf("unsupported diagram type: %q", t)
		}
		if m[1] != "" {
			g.dir = m[1]
		}
		i, found = i+1, true
		break
	}
	if !found {
		return nil, fmt.Errorf("missing flowchart header")
	}

	depth := 0 // subgraph nesting
	for ; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		if kw, ok := keyword(line); ok {
			switch kw {
			case "subgraph":
				depth++
			case kwEnd:
				if depth > 0 {
					depth--
				}
			}
			continue
		}
		for _, stmt := range splitStatements(line) {
			if err := g.parseStatement(stmt); err != nil {
				return nil, err
			}
		}
	}
	if len(g.order) == 0 {
		return nil, fmt.Errorf("no nodes found")
	}
	return g, nil
}

// keyword reports whether the line is a statement we must skip, and if it
// is a subgraph opener.
func keyword(line string) (string, bool) {
	for _, kw := range []string{
		"subgraph", "direction", "classDef", "class ", "style ",
		"linkStyle ", "click ", "accTitle:", "accTitle ", "accDescr:", "accDescr ",
	} {
		if strings.HasPrefix(line, kw) {
			return kw, true
		}
	}
	if line == kwEnd {
		return kwEnd, true
	}
	return "", false
}

// splitStatements splits a line on ";" outside of quoted labels.
func splitStatements(line string) []string {
	var out []string
	var cur strings.Builder
	inQ := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case c == '\\' && inQ && i+1 < len(line):
			cur.WriteByte(c)
			i++
			cur.WriteByte(line[i])
		case c == '"':
			inQ = !inQ
			cur.WriteByte(c)
		case c == ';' && !inQ:
			if s := strings.TrimSpace(cur.String()); s != "" {
				out = append(out, s)
			}
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	if s := strings.TrimSpace(cur.String()); s != "" {
		out = append(out, s)
	}
	return out
}

// parseStatement parses "A --> B --> C" chains.
func (g *graph) parseStatement(s string) error {
	pos := 0
	var lastID string
	for {
		pos = skipSpaces(s, pos)
		if pos >= len(s) {
			break
		}

		// edge (labeled first: "-- text -->" contains "--")
		if lastID != "" {
			id, np, ok, err := g.parseEdgeTarget(s, pos, lastID)
			if err != nil {
				return err
			}
			if ok {
				lastID, pos = id, np
				continue
			}
		}

		id, np, err := g.parseNode(s, pos)
		if err != nil {
			return err
		}
		if lastID != "" {
			return fmt.Errorf("expected edge between %q and %q", lastID, id)
		}
		lastID, pos = id, np
	}
	if lastID == "" {
		return fmt.Errorf("empty statement")
	}
	return nil
}

// parseEdgeTarget parses an edge operator at pos and the node following it.
// ok is false when there is no edge operator at pos.
func (g *graph) parseEdgeTarget(s string, pos int, from string) (string, int, bool, error) {
	e, np, ok := matchLabeledEdge(s, pos)
	if !ok {
		e, np, ok = scanEdge(s, pos)
		if !ok {
			return "", pos, false, nil
		}
		np = skipPipes(s, np, &e)
	}
	to, np2, err := g.parseNode(s, np)
	if err != nil {
		return "", pos, true, err
	}
	e.from, e.to = from, to
	g.edges = append(g.edges, e)
	return to, np2, true, nil
}

func skipSpaces(s string, pos int) int {
	for pos < len(s) && (s[pos] == ' ' || s[pos] == '\t') {
		pos++
	}
	return pos
}

func (g *graph) nodeRef(id string) string {
	if _, ok := g.nodes[id]; !ok {
		g.nodes[id] = node{label: id}
		g.order = append(g.order, id)
	}
	return id
}

func isIDChar(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
		c == '_' || c == '-' || c == '.'
}

// startsEdge reports whether an edge operator begins at pos: "-" followed
// by "-" or "." (-->, ---, -.->, -.-). Keeps ids like "a-b" parseable
// while letting "A-->B" (no spaces) split correctly.
func startsEdge(s string, pos int) bool {
	return s[pos] == '-' && pos+1 < len(s) && (s[pos+1] == '-' || s[pos+1] == '.')
}

// parseNode parses a node spec at pos: "id", "id[Label]", 'id["Quoted"]',
// "id(Round)", "id((Circle))", "id{Diamond}", "id{{Hex}}", "id([Stadium])",
// "id>Flag]". Returns the node id and the position after the spec.
func (g *graph) parseNode(s string, pos int) (string, int, error) {
	pos = skipSpaces(s, pos)
	start := pos
	for pos < len(s) && isIDChar(s[pos]) && !startsEdge(s, pos) {
		pos++
	}
	if pos == start {
		return "", 0, fmt.Errorf("expected node id at %q", s[start:])
	}
	id := s[start:pos]

	label, round, hasShape, np, err := readShape(s, pos)
	if err != nil {
		return "", 0, err
	}
	pos = np
	if hasShape {
		if _, exists := g.nodes[id]; !exists {
			g.order = append(g.order, id)
		}
		g.nodes[id] = node{label: cleanLabel(label), round: round}
	} else {
		g.nodeRef(id) // bare reference: keep a previous label if any
	}
	return id, pos, nil
}

// readShape parses an optional node shape at pos and returns the raw label
// inner text.
func readShape(s string, pos int) (label string, round, hasShape bool, newPos int, err error) {
	if pos >= len(s) {
		return "", false, false, pos, nil
	}
	switch s[pos] {
	case '[':
		inner, np, ok := readUntil(s, pos, '[', ']')
		if !ok {
			return "", false, false, pos, fmt.Errorf("unterminated [ in %q", s)
		}
		return inner, false, true, np, nil
	case '(':
		inner, np, ok := readUntil(s, pos, '(', ')')
		if !ok {
			return "", false, false, pos, fmt.Errorf("unterminated ( in %q", s)
		}
		// ((circle)) or ([stadium]): strip one more layer
		if strings.HasPrefix(inner, "(") && strings.HasSuffix(inner, ")") ||
			strings.HasPrefix(inner, "[") && strings.HasSuffix(inner, "]") {
			inner = inner[1 : len(inner)-1]
		}
		return inner, true, true, np, nil
	case '{':
		inner, np, ok := readUntil(s, pos, '{', '}')
		if !ok {
			return "", false, false, pos, fmt.Errorf("unterminated { in %q", s)
		}
		if strings.HasPrefix(inner, "{") && strings.HasSuffix(inner, "}") {
			inner = inner[1 : len(inner)-1]
		}
		return inner, true, true, np, nil
	case '>':
		inner, np, ok := readUntil(s, pos, '>', ']')
		if !ok {
			return "", false, false, pos, fmt.Errorf("unterminated > in %q", s)
		}
		return inner, true, true, np, nil
	}
	return "", false, false, pos, nil
}

// readUntil returns the content between the delimiter pair starting at pos
// (s[pos] must be open), respecting quotes and nesting of the same pair.
func readUntil(s string, pos int, open, closer byte) (string, int, bool) {
	depth := 0
	inQ := false
	for i := pos; i < len(s); i++ {
		c := s[i]
		if inQ {
			switch c {
			case '\\':
				i++
			case '"':
				inQ = false
			}
			continue
		}
		switch c {
		case '"':
			inQ = true
		case open:
			depth++
		case closer:
			depth--
			if depth == 0 {
				return s[pos+1 : i], i + 1, true
			}
		}
	}
	return "", pos, false
}

// cleanLabel strips quotes, unescapes and converts <br/> to newlines.
var brRe = regexp.MustCompile(`(?i)<br\s*/?>`)

func cleanLabel(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
		s = strings.ReplaceAll(s, `\"`, `"`)
		s = strings.ReplaceAll(s, `\\`, `\`)
	}
	s = brRe.ReplaceAllString(s, "\n")
	return s
}

// matchLabeledEdge matches "X-- text -->Y" edge forms at pos. The closing
// token must match the opening one in kind (solid/dotted/thick).
var labeledEdgeRe = regexp.MustCompile(`^(--|-\.|==)\s+(.+?)\s+(-{2,}>|-{3,}|\.{1,2}->|\.-|={2,}>|={3,})`)

func matchLabeledEdge(s string, pos int) (e edge, npos int, ok bool) {
	m := labeledEdgeRe.FindStringSubmatch(s[pos:])
	if m == nil {
		return e, pos, false
	}
	kind, closing := m[1], m[3]
	matched := kind == "--" && closing[0] == '-' ||
		kind == "-." && closing[0] == '.' ||
		kind == "==" && closing[0] == '='
	if !matched {
		return e, pos, false
	}
	e.label = cleanLabel(m[2])
	e.arrow = strings.HasSuffix(closing, ">")
	e.dotted, e.thick = kind == "-.", kind == "=="
	return e, pos + len(m[0]), true
}

// scanEdge scans an unlabeled edge token ("-->", "---", "-.->", "==>", ...)
// at pos.
func scanEdge(s string, pos int) (edge, int, bool) {
	var e edge
	j := pos
	switch s[j] {
	case '-':
		if j+1 < len(s) && s[j+1] == '.' {
			for j < len(s) && (s[j] == '.' || s[j] == '-') {
				j++
			}
			if j-pos < 3 {
				return e, pos, false
			}
			e.dotted = true
		} else {
			for j < len(s) && s[j] == '-' {
				j++
			}
			if j-pos < 2 {
				return e, pos, false
			}
		}
	case '=':
		for j < len(s) && s[j] == '=' {
			j++
		}
		if j-pos < 2 {
			return e, pos, false
		}
		e.thick = true
	default:
		return e, pos, false
	}
	if j < len(s) && s[j] == '>' {
		e.arrow = true
		j++
	}
	return e, j, true
}

// skipPipes consumes an optional "|label|" suffix after an edge token.
func skipPipes(s string, pos int, e *edge) int {
	if pos >= len(s) || s[pos] != '|' {
		return pos
	}
	end := strings.IndexByte(s[pos+1:], '|')
	if end < 0 {
		return pos
	}
	e.label = cleanLabel(s[pos+1 : pos+1+end])
	return pos + end + 2
}
