package mermaid

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

func TestParseNodesAndEdges(t *testing.T) {
	g, err := parse(`flowchart TD
    A[Hard edge] -->|Link text| B(Round edge)
    B --> C{Decision}
    C -->|One| D[Result one]
    C -->|Two| E[Result two]`)
	if err != nil {
		t.Fatal(err)
	}
	if g.dir != "TD" {
		t.Errorf("dir = %q, want TD", g.dir)
	}
	if len(g.nodes) != 5 {
		t.Errorf("nodes = %d, want 5", len(g.nodes))
	}
	if len(g.edges) != 4 {
		t.Errorf("edges = %d, want 4", len(g.edges))
	}
	if g.nodes["B"].label != "Round edge" || !g.nodes["B"].round {
		t.Errorf("B = %+v", g.nodes["B"])
	}
	if g.nodes["C"].label != "Decision" || !g.nodes["C"].round { // non-rect shapes render rounded
		t.Errorf("C = %+v", g.nodes["C"])
	}
	if g.edges[0].label != "Link text" || !g.edges[0].arrow {
		t.Errorf("edge0 = %+v", g.edges[0])
	}
	if g.order[0] != "A" {
		t.Errorf("order0 = %q, want A", g.order[0])
	}
}

func TestParseEdgeForms(t *testing.T) {
	tests := []struct {
		src    string
		arrow  bool
		dotted bool
		thick  bool
	}{
		{"A --> B", true, false, false},
		{"A-->B", true, false, false},
		{"A---B", false, false, false},
		{"A-.->B", true, true, false},
		{"A==>B", true, false, true},
		{"a-b-->c", true, false, false},
		{"a.b-->c", true, false, false},
		{"A --- B", false, false, false},
		{"A -.-> B", true, true, false},
		{"A -.- B", false, true, false},
		{"A ==> B", true, false, true},
		{"A === B", false, false, true},
		{"A -- text --> B", true, false, false},
		{"A -- text --- B", false, false, false},
		{"A-. text .->B", true, true, false},
		{"A -. text .- B", false, true, false},
		{"A == text ==> B", true, false, true},
		{"A -->|text| B", true, false, false},
		{"A ---> B", true, false, false},
	}
	for _, tt := range tests {
		g, err := parse("flowchart TD\n    " + tt.src + "\n")
		if err != nil {
			t.Errorf("%q: %v", tt.src, err)
			continue
		}
		e := g.edges[0]
		if e.arrow != tt.arrow || e.dotted != tt.dotted || e.thick != tt.thick {
			t.Errorf("%q: arrow=%v dotted=%v thick=%v", tt.src, e.arrow, e.dotted, e.thick)
		}
	}
}

func TestParseChainsAndSeparators(t *testing.T) {
	g, err := parse("flowchart LR\n    A --> B --> C; D --> E\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(g.edges) != 3 {
		t.Errorf("edges = %d, want 3", len(g.edges))
	}
	if len(g.nodes) != 5 {
		t.Errorf("nodes = %d, want 5", len(g.nodes))
	}
}

func TestParseQuotedLabels(t *testing.T) {
	g, err := parse(`flowchart TD
    A["Label with ] bracket and \"quotes\""] --> B["other"]`)
	if err != nil {
		t.Fatal(err)
	}
	if g.nodes["A"].label != `Label with ] bracket and "quotes"` {
		t.Errorf("A label = %q", g.nodes["A"].label)
	}
	if g.nodes["B"].label != "other" {
		t.Errorf("B label = %q", g.nodes["B"].label)
	}
}

func TestParseLineBreaks(t *testing.T) {
	g, err := parse("flowchart TD\n    A[line1<br/>line2] --> B\n")
	if err != nil {
		t.Fatal(err)
	}
	if g.nodes["A"].label != "line1\nline2" {
		t.Errorf("label = %q", g.nodes["A"].label)
	}
}

func TestParseSkipsStatements(t *testing.T) {
	src := `flowchart TD
    %% a comment
    A --> B
    classDef special fill:#f9f
    class A special
    style B fill:#2a2
    linkStyle 0 stroke:red
    click A "https://example.com"
    subgraph Sub
    direction LR
    C --> D
    end
`
	g, err := parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.edges) != 2 {
		t.Errorf("edges = %d, want 2", len(g.edges))
	}
	for _, id := range []string{"A", "B", "C", "D"} {
		if _, ok := g.nodes[id]; !ok {
			t.Errorf("missing node %q", id)
		}
	}
}

func TestParseDirections(t *testing.T) {
	for _, dir := range []string{"TD", "TB", "BT", "LR", "RL"} {
		g, err := parse("flowchart " + dir + "\n    A --> B\n")
		if err != nil {
			t.Errorf("%s: %v", dir, err)
		}
		if g.dir != dir {
			t.Errorf("dir = %q, want %q", g.dir, dir)
		}
	}
	// bare "graph" defaults to TD
	g, err := parse("graph\n    A --> B\n")
	if err != nil {
		t.Fatal(err)
	}
	if g.dir != "TD" {
		t.Errorf("default dir = %q", g.dir)
	}
}

func TestParseUnsupported(t *testing.T) {
	for _, src := range []string{
		"sequenceDiagram\n    A->>B: hi\n",
		"hello world\n",
		"%% only a comment\n",
		"",
		"flowchart TD\n",
		"flowchart TD\n    A[unterminated\n",
		"flowchart TD\n    A -->\n",
		"flowchart TD\n    A B\n",
	} {
		if _, err := parse(src); err == nil {
			t.Errorf("expected error for %q", src)
		}
	}
}

func TestRenderSimple(t *testing.T) {
	out, err := Render("flowchart TD\n    A --> B\n")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"┌─┐",
		"│A│",
		"└─┘",
		" │ ",
		" │ ",
		" ▼ ",
		"┌─┐",
		"│B│",
		"└─┘",
	}, "\n")
	if out != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestRenderBranches(t *testing.T) {
	out, err := Render(`flowchart TD
    A -->|yes| B
    A -->|no| C`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"yes", "▼", "│A│", "│B│", "│C│", "┌─┐"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	lines := strings.Split(out, "\n")
	w := utf8.RuneCountInString(lines[0])
	for i, l := range lines {
		if utf8.RuneCountInString(l) != w {
			t.Errorf("line %d width %d, want %d:\n%s", i, utf8.RuneCountInString(l), w, out)
		}
	}
}

func TestRenderLR(t *testing.T) {
	out, err := Render("flowchart LR\n    A --> B --> C\n")
	if err != nil {
		t.Fatal(err)
	}
	// all three boxes on the same text lines
	lines := strings.Split(out, "\n")
	if !strings.Contains(lines[1], "│A│") || !strings.Contains(lines[1], "│B│") || !strings.Contains(lines[1], "│C│") {
		t.Errorf("expected boxes side by side:\n%s", out)
	}
}

func TestRenderCycle(t *testing.T) {
	out, err := Render("graph TD\n    A --> B\n    B --> C\n    C --> A\n    C --> D\n")
	if err != nil {
		t.Fatal(err)
	}
	// back edge C-->A is dropped; the rest must still render
	for _, want := range []string{"│A│", "│B│", "│C│", "│D│", "▼"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestRenderShapeVariants(t *testing.T) {
	out, err := Render(`flowchart TD
    a[rect] --> b(round)
    b --> c((circle))
    c --> d{diamond}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"│rect│", "╭", "╯", "│circle│", "│diamond│"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestRenderLabelWrapping(t *testing.T) {
	out, err := Render("flowchart TD\n    A[this is a very long label that must wrap] --> B\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"│ this is a very long│", "│label that must wrap│"} {
		if !strings.Contains(out, want) {
			t.Errorf("wrapped label missing %q:\n%s", want, out)
		}
	}
}

func TestRenderWideRunes(t *testing.T) {
	out, err := Render("flowchart TD\n    A[開始] -->|確認| B[終了]\n")
	if err != nil {
		t.Fatal(err)
	}
	// wide runes must emit contiguously (no gaps) so box borders align;
	// edge-label rows may overhang past the box, so only border rows are
	// width-compared
	if !strings.Contains(out, "│開始│") {
		t.Errorf("box label not contiguous:\n%s", out)
	}
	if !strings.Contains(out, "│終了│") {
		t.Errorf("box label not contiguous:\n%s", out)
	}
	w := 0
	for _, l := range strings.Split(out, "\n") {
		if !strings.ContainsAny(l, "┌└") {
			continue
		}
		if lw := runewidth.StringWidth(l); w == 0 {
			w = lw
		} else if lw != w {
			t.Errorf("border row width %d, want %d:\n%s", lw, w, out)
		}
	}
	for _, want := range []string{"開", "始", "確", "認", "終", "了"} {
		if !strings.Contains(out, want) {
			t.Errorf("wide rune %q lost:\n%s", want, out)
		}
	}
}

func TestRenderLongEdgeLabel(t *testing.T) {
	out, err := Render("flowchart TD\n    A -->|a rather long edge label| B\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "a rather long edge label") {
		t.Errorf("long edge label clipped:\n%s", out)
	}
}

func TestRenderFanOutLabels(t *testing.T) {
	out, err := Render("flowchart TD\n    A -->|label one| B\n    A -->|label two| C\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"label one", "label two"} {
		if !strings.Contains(out, want) {
			t.Errorf("fan-out label %q missing:\n%s", want, out)
		}
	}
}
