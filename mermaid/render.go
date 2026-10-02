package mermaid

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

const (
	gapX    = 4 // min horizontal gap between boxes
	gapY    = 3 // min vertical gap between ranks (2 routing lanes)
	laneGap = 2 // min cross-axis gap between boxes in the same rank (LR)
	wrapAt  = 20
)

// direction bits of a cell, in abstract (rank-axis, cross-axis) space.
const (
	bitFwd  = 1 << iota // towards higher ranks
	bitBack             // towards lower ranks
	bitCPos             // cross axis +
	bitCNeg             // cross axis -
)

type cell struct {
	bits uint8
	ch   rune // explicit rune (box border, arrowhead, label); wins over bits
}

// box is a rendered node.
type box struct {
	lines []string
	w, h  int
	a, c  int // top-left in abstract space (rank axis, cross axis)
	round bool
}

// crossExt is the box extent along the cross axis, alongExt along the rank
// axis. Text is always horizontal, so the extents swap with orientation.
func (b *box) crossExt(td bool) int {
	if td {
		return b.w
	}
	return b.h
}

func (b *box) alongExt(td bool) int {
	if td {
		return b.h
	}
	return b.w
}

// canvas is a character grid addressed in abstract space (a = rank axis,
// c = cross axis) and mapped to (x, y) according to orientation.
type canvas struct {
	td         bool // top-down (rank axis is y); otherwise left-right
	maxA, maxC int
	cells      []cell
}

func newCanvas(maxA, maxC int, td bool) *canvas {
	return &canvas{td: td, maxA: maxA, maxC: maxC, cells: make([]cell, maxA*maxC)}
}

func (cv *canvas) at(a, c int) *cell {
	return &cv.cells[a*cv.maxC+c]
}

// grow enlarges the canvas so writes past the bounds (edge labels) fit.
func (cv *canvas) grow(a, c int) {
	na, nc := cv.maxA, cv.maxC
	if a >= na {
		na = a + 1
	}
	if c >= nc {
		nc = c + 1
	}
	if na == cv.maxA && nc == cv.maxC {
		return
	}
	cells := make([]cell, na*nc)
	for i := 0; i < cv.maxA; i++ {
		copy(cells[i*nc:i*nc+cv.maxC], cv.cells[i*cv.maxC:(i+1)*cv.maxC])
	}
	cv.cells, cv.maxA, cv.maxC = cells, na, nc
}

func (cv *canvas) set(a, c int, bit uint8) {
	if a < 0 || c < 0 {
		return
	}
	cv.grow(a, c)
	cell := cv.at(a, c)
	if cell.ch != 0 {
		return // never draw through boxes/labels/arrowheads
	}
	cell.bits |= bit
}

func (cv *canvas) setRune(a, c int, r rune, force bool) {
	if a < 0 || c < 0 {
		return
	}
	cv.grow(a, c)
	cell := cv.at(a, c)
	if cell.ch != 0 && !force {
		return
	}
	cell.bits = 0
	cell.ch = r
}

// setScreen writes a rune at screen coordinates (text always runs along
// screen x, so boxes and labels are drawn in screen space).
func (cv *canvas) setScreen(x, y int, r rune, force bool) {
	if cv.td {
		cv.setRune(y, x, r, force)
	} else {
		cv.setRune(x, y, r, force)
	}
}

// Render parses src and renders the flowchart as box-drawing art.
// ponytail: no terminal-width awareness — wide diagrams overflow the pager
// instead of being re-laid-out; add width-driven direction flip if that
// ever becomes a real complaint.
func Render(src string) (string, error) {
	g, err := parse(src)
	if err != nil {
		return "", err
	}
	return g.render(), nil
}

func (g *graph) render() string {
	rev := g.dir == "BT" || g.dir == "RL"
	td := g.dir == "TD" || g.dir == "TB" || g.dir == "BT"

	// effective edge direction: reversed layouts swap from/to
	edges := make([]edge, 0, len(g.edges))
	for _, e := range g.edges {
		if rev {
			e.from, e.to = e.to, e.from
		}
		edges = append(edges, e)
	}

	rank := ranks(g.order, edges)
	maxRank := 0
	for _, r := range rank {
		if r > maxRank {
			maxRank = r
		}
	}
	rows := make([][]string, maxRank+1)
	for _, id := range g.order { // first-appearance order within each rank
		rows[rank[id]] = append(rows[rank[id]], id)
	}

	boxes := map[string]*box{}
	for _, id := range g.order {
		n := g.nodes[id]
		label := n.label
		if label == "" {
			label = id
		}
		lines := wrapLabel(label, wrapAt)
		w, h := 0, len(lines)+2 // padding + borders
		for _, l := range lines {
			if lw := runewidth.StringWidth(l); lw > w-2 {
				w = lw + 2
			}
		}
		boxes[id] = &box{lines: lines, w: w, h: h, round: n.round}
	}

	// place: cross positions within each rank, ranks stacked along the axis
	for r, row := range rows {
		cross := 0
		for _, id := range row {
			boxes[id].c = cross
			cross += boxes[id].crossExt(td) + laneGap
		}
		if r == 0 {
			continue
		}
		along := 0
		for _, id := range rows[r-1] {
			if v := boxes[id].a + boxes[id].alongExt(td) + gapY; v > along {
				along = v
			}
		}
		for _, id := range row {
			boxes[id].a = along
		}
	}
	maxAlong, maxCross := 0, 0
	for _, b := range boxes {
		if v := b.a + b.alongExt(td); v > maxAlong {
			maxAlong = v
		}
		if v := b.c + b.crossExt(td); v > maxCross {
			maxCross = v
		}
	}

	cv := newCanvas(maxAlong, maxCross, td)
	slots := map[int]int{} // label stagger per lane

	// draw boxes (in screen space: text runs along screen x)
	for _, id := range g.order {
		b := boxes[id]
		at := func(i, j int, r rune, force bool) {
			if td {
				cv.setScreen(b.c+j, b.a+i, r, force)
			} else {
				cv.setScreen(b.a+j, b.c+i, r, force)
			}
		}
		for i := 0; i < b.h; i++ {
			for j := 0; j < b.w; j++ {
				at(i, j, ' ', false)
			}
		}
		corners := [4]rune{'┌', '┐', '└', '┘'}
		if b.round {
			corners = [4]rune{'╭', '╮', '╰', '╯'}
		}
		at(0, 0, corners[0], true)
		at(0, b.w-1, corners[1], true)
		at(b.h-1, 0, corners[2], true)
		at(b.h-1, b.w-1, corners[3], true)
		for j := 1; j < b.w-1; j++ {
			at(0, j, '─', true)
			at(b.h-1, j, '─', true)
		}
		for i := 1; i < b.h-1; i++ {
			at(i, 0, '│', true)
			at(i, b.w-1, '│', true)
			l := b.lines[i-1]
			off := (b.w - 2 - runewidth.StringWidth(l) + 1) / 2
			col := 0
			for _, r := range l {
				at(i, 1+off+col, r, true)
				col += runewidth.RuneWidth(r)
			}
		}
	}

	// route edges
	for _, e := range edges {
		src, dst := boxes[e.from], boxes[e.to]
		if e.from == e.to {
			continue // ponytail: self-loops not drawn yet
		}
		if rank[e.to] < rank[e.from] {
			continue // ponytail: back edges of cycles are dropped; forward
			// edges still convey the flow. Proper detour routing later.
		}
		// equal ranks are impossible: the ranking DP guarantees every kept
		// edge goes to a strictly higher rank

		var arrow rune
		switch {
		case !e.arrow:
		case td && !rev:
			arrow = '▼'
		case td && rev:
			arrow = '▲'
		case !td && !rev:
			arrow = '▶'
		default:
			arrow = '◀'
		}

		drawRankEdge(cv, slots, src, dst, e.label, arrow, td)
	}
	return cv.string()
}

// drawRankEdge draws an edge to a higher-rank box: along the rank axis to
// the mid lane, across, then along again. slots staggers the labels of
// edges sharing a lane so fan-outs don't overwrite each other.
func drawRankEdge(cv *canvas, slots map[int]int, src, dst *box, label string, arrow rune, td bool) {
	srcA := src.a + src.alongExt(td) // first free cell after src
	dstA := dst.a - 1                // last free cell before dst
	sc, dc := src.c+src.crossExt(td)/2, dst.c+dst.crossExt(td)/2
	lane := (srcA + dstA) / 2

	path := line(srcA, sc, lane, sc)
	if sc != dc {
		path = append(path, line(lane, sc, lane, dc)[1:]...)
		path = append(path, line(lane, dc, dstA, dc)[1:]...)
	} else {
		path = append(path, line(lane, sc, dstA, sc)[1:]...)
	}
	drawPath(cv, path)

	// TD labels line up left-to-right along the shared branch line so each
	// stays attached to its own branch — a label staggered next to the
	// trunk reads as labeling the wrong branch. LR labels stack beside the
	// vertical lane segment.
	if td {
		start := min(sc, dc) + 1 + slots[lane]
		slots[lane] += runewidth.StringWidth(label) + 1
		drawLabel(cv, lane, start, label, true)
	} else {
		n := slots[lane]
		slots[lane] = n + 1
		if sc != dc {
			drawLabel(cv, lane+1, min(sc, dc)+n, label, false)
		} else {
			// short horizontal stub: label sits on the line, arrow keeps its cell
			drawLabel(cv, srcA, min(sc+n, dst.c+dst.crossExt(td)-1), label, false)
		}
	}
	if arrow != 0 {
		end := path[len(path)-1]
		cv.setRune(end.a, end.c, arrow, true)
	}
}

// line returns the cells from (a0,c0) to (a1,c1) along one axis, inclusive.
func line(a0, c0, a1, c1 int) []struct{ a, c int } {
	a, c := a0, c0
	da, dc := 0, 0
	if a1 > a0 {
		da = 1
	} else if a1 < a0 {
		da = -1
	}
	if c1 > c0 {
		dc = 1
	} else if c1 < c0 {
		dc = -1
	}
	n := max(abs(a1-a0), abs(c1-c0)) + 1
	path := make([]struct{ a, c int }, 0, n)
	for {
		path = append(path, struct{ a, c int }{a, c})
		if a == a1 && c == c1 {
			return path
		}
		a, c = a+da, c+dc
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// drawPath marks the cells of a path with the direction bits of their
// path neighbours; junctions between edges merge into the right glyph.
func drawPath(cv *canvas, path []struct{ a, c int }) {
	for i, p := range path {
		if i > 0 {
			cv.set(p.a, p.c, dirBit(p, path[i-1]))
		}
		if i < len(path)-1 {
			cv.set(p.a, p.c, dirBit(p, path[i+1]))
		}
	}
}

// dirBit is the connection direction from p towards q.
func dirBit(p, q struct{ a, c int }) uint8 {
	switch {
	case q.a > p.a:
		return bitFwd
	case q.a < p.a:
		return bitBack
	case q.c > p.c:
		return bitCPos
	default:
		return bitCNeg
	}
}

// drawLabel writes a label left-to-right along screen x starting next to
// (a, c), without touching existing runes. td: beside a horizontal segment;
// !td: beside a vertical one. Columns advance by display width, not bytes.
func drawLabel(cv *canvas, a, c int, label string, td bool) {
	if label == "" {
		return
	}
	col := 0
	for _, r := range label {
		if td {
			cv.setRune(a, c+col, r, false)
		} else {
			cv.setRune(a+col, c, r, false)
		}
		col += runewidth.RuneWidth(r)
	}
}

// ranks assigns each node a rank via longest-path layering, with cycles
// broken during DFS.
func ranks(order []string, edges []edge) map[string]int {
	adj := map[string][]string{}
	for _, e := range edges {
		adj[e.from] = append(adj[e.from], e.to)
	}
	const (
		white = iota
		gray
		black
	)
	state := map[string]int{}
	var topo []string
	var dfs func(string)
	dfs = func(n string) {
		state[n] = gray
		for _, m := range adj[n] {
			if state[m] == white {
				dfs(m)
			}
		}
		state[n] = black
		topo = append(topo, n)
	}
	for _, id := range order {
		if state[id] == white {
			dfs(id)
		}
	}
	rank := map[string]int{}
	for i := len(topo) - 1; i >= 0; i-- { // topological order
		n := topo[i]
		for _, e := range edges {
			if e.to == n {
				if r := rank[e.from] + 1; r > rank[n] {
					rank[n] = r
				}
			}
		}
	}
	return rank
}

func wrapLabel(s string, at int) []string {
	var out []string
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		cur := words[0]
		for _, w := range words[1:] {
			if runewidth.StringWidth(cur)+1+runewidth.StringWidth(w) <= at {
				cur += " " + w
			} else {
				out = append(out, cur)
				cur = w
			}
		}
		out = append(out, cur)
	}
	return out
}

func (cv *canvas) string() string {
	h, w := cv.maxA, cv.maxC
	if !cv.td {
		h, w = cv.maxC, cv.maxA
	}
	var sb strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var cell *cell
			if cv.td {
				cell = cv.at(y, x)
			} else {
				cell = cv.at(x, y)
			}
			if cell.ch != 0 {
				sb.WriteRune(cell.ch)
				// ponytail: a wide rune also covers the next terminal cell;
				// the canvas keeps it empty, so skip it to keep columns
				// aligned (combining/zero-width runes unsupported)
				if runewidth.RuneWidth(cell.ch) > 1 {
					x++
				}
				continue
			}
			// map abstract bits to screen directions
			var n, s, e, w bool
			if cv.td {
				n, s, e, w = cell.bits&bitBack != 0, cell.bits&bitFwd != 0, cell.bits&bitCPos != 0, cell.bits&bitCNeg != 0
			} else {
				n, s, e, w = cell.bits&bitCNeg != 0, cell.bits&bitCPos != 0, cell.bits&bitFwd != 0, cell.bits&bitBack != 0
			}
			sb.WriteRune(glyph(n, s, e, w))
		}
		sb.WriteByte('\n')
	}
	return strings.TrimRight(sb.String(), "\n")
}

func glyph(n, s, e, w bool) rune {
	switch {
	case n && s && e && w:
		return '┼'
	case n && s && e:
		return '├'
	case n && s && w:
		return '┤'
	case n && e && w:
		return '┴'
	case s && e && w:
		return '┬'
	case n && s:
		return '│'
	case e && w:
		return '─'
	case n && e:
		return '└'
	case n && w:
		return '┘'
	case s && e:
		return '┌'
	case s && w:
		return '┐'
	case n, s:
		return '│'
	case e, w:
		return '─'
	}
	return ' '
}
