package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func testPagerModel(width, height int, cfg Config) pagerModel {
	config = cfg
	common := &commonModel{cfg: cfg, width: width, height: height}
	vp := viewport.New(width, height)
	return pagerModel{
		common:   common,
		viewport: vp,
		state:    pagerStateBrowse,
	}
}

func TestGlamourRender(t *testing.T) {
	savedConfig := config
	t.Cleanup(func() { config = savedConfig })

	t.Run("GlamourEnabled false returns raw markdown", func(t *testing.T) {
		cfg := Config{GlamourEnabled: false}
		m := testPagerModel(80, 24, cfg)

		input := "# Hello\n\nWorld"
		got, err := glamourRender(m, input)
		if err != nil {
			t.Fatalf("glamourRender() error: %v", err)
		}
		if got != input {
			t.Errorf("glamourRender() = %q, want %q", got, input)
		}
	})

	t.Run("markdown file renders non-empty", func(t *testing.T) {
		cfg := Config{
			GlamourEnabled:  true,
			GlamourStyle:    "dark",
			GlamourMaxWidth: 80,
		}
		m := testPagerModel(80, 24, cfg)
		m.currentDocument = markdown{Note: "test.md"}

		got, err := glamourRender(m, "# Hello\n\nWorld")
		if err != nil {
			t.Fatalf("glamourRender() error: %v", err)
		}
		if got == "" {
			t.Error("glamourRender() returned empty string for markdown")
		}
	})

	t.Run("code file wraps in code block", func(t *testing.T) {
		cfg := Config{
			GlamourEnabled:  true,
			GlamourStyle:    "dark",
			GlamourMaxWidth: 80,
		}
		m := testPagerModel(80, 24, cfg)
		m.currentDocument = markdown{Note: "main.go"}

		got, err := glamourRender(m, "package main\n")
		if err != nil {
			t.Fatalf("glamourRender() error: %v", err)
		}
		if got == "" {
			t.Error("glamourRender() returned empty for code file")
		}
	})

	t.Run("ShowLineNumbers adds prefixes", func(t *testing.T) {
		cfg := Config{
			GlamourEnabled:  true,
			GlamourStyle:    "dark",
			GlamourMaxWidth: 80,
			ShowLineNumbers: true,
		}
		m := testPagerModel(80, 24, cfg)
		m.currentDocument = markdown{Note: "test.md"}

		got, err := glamourRender(m, "# Hello")
		if err != nil {
			t.Fatalf("glamourRender() error: %v", err)
		}
		// Line numbers should be present - look for the number 1
		if !strings.Contains(got, "1") {
			t.Errorf("glamourRender() with ShowLineNumbers should contain line numbers, got: %q", got)
		}
	})

	t.Run("code files always get line numbers", func(t *testing.T) {
		cfg := Config{
			GlamourEnabled:  true,
			GlamourStyle:    "dark",
			GlamourMaxWidth: 80,
			ShowLineNumbers: false, // explicitly false
		}
		m := testPagerModel(80, 24, cfg)
		m.currentDocument = markdown{Note: "main.go"}

		got, err := glamourRender(m, "package main\n")
		if err != nil {
			t.Fatalf("glamourRender() error: %v", err)
		}
		// Code files always get line numbers regardless of ShowLineNumbers
		if !strings.Contains(got, "1") {
			t.Errorf("glamourRender() code file should have line numbers, got: %q", got)
		}
	})
}

func TestStatusBarView(t *testing.T) {
	savedConfig := config
	t.Cleanup(func() { config = savedConfig })

	t.Run("browse state shows Note", func(t *testing.T) {
		cfg := Config{}
		m := testPagerModel(80, 24, cfg)
		m.currentDocument = markdown{Note: "myfile.md"}
		m.state = pagerStateBrowse

		var b strings.Builder
		m.statusBarView(&b)
		got := b.String()
		if !strings.Contains(got, "myfile.md") {
			t.Errorf("statusBarView() in browse should contain Note, got: %q", got)
		}
	})

	t.Run("status message state shows message", func(t *testing.T) {
		cfg := Config{}
		m := testPagerModel(80, 24, cfg)
		m.state = pagerStateStatusMessage
		m.statusMessage = "Copied contents"

		var b strings.Builder
		m.statusBarView(&b)
		got := b.String()
		if !strings.Contains(got, "Copied contents") {
			t.Errorf("statusBarView() should contain status message, got: %q", got)
		}
	})

	t.Run("narrow width no panic", func(t *testing.T) {
		cfg := Config{}
		m := testPagerModel(10, 5, cfg)
		m.currentDocument = markdown{Note: "a-very-long-filename-that-exceeds-width.md"}
		m.state = pagerStateBrowse

		var b strings.Builder
		// Should not panic
		m.statusBarView(&b)
	})

	t.Run("zero width no panic", func(t *testing.T) {
		cfg := Config{}
		m := testPagerModel(0, 0, cfg)
		m.state = pagerStateBrowse

		var b strings.Builder
		m.statusBarView(&b)
	})
}

func TestHelpView(t *testing.T) {
	savedConfig := config
	t.Cleanup(func() { config = savedConfig })

	cfg := Config{}
	m := testPagerModel(80, 24, cfg)

	got := m.helpView()

	if got == "" {
		t.Error("helpView() returned empty string")
	}

	expectedBindings := []string{"g/home", "G/end", "esc"}
	for _, binding := range expectedBindings {
		if !strings.Contains(got, binding) {
			t.Errorf("helpView() should contain %q", binding)
		}
	}
}

func TestSetSize(t *testing.T) {
	savedConfig := config
	t.Cleanup(func() { config = savedConfig })

	t.Run("viewport dimensions correct", func(t *testing.T) {
		cfg := Config{}
		m := testPagerModel(80, 24, cfg)
		m.setSize(100, 30)

		if m.viewport.Width != 100 {
			t.Errorf("viewport.Width = %d, want 100", m.viewport.Width)
		}
		wantHeight := 30 - statusBarHeight
		if m.viewport.Height != wantHeight {
			t.Errorf("viewport.Height = %d, want %d", m.viewport.Height, wantHeight)
		}
	})

	t.Run("showHelp reduces height", func(t *testing.T) {
		cfg := Config{}
		m := testPagerModel(80, 24, cfg)

		m.setSize(80, 40)
		heightWithoutHelp := m.viewport.Height

		m.showHelp = true
		pagerHelpHeight = 0 // reset so it recalculates
		m.setSize(80, 40)
		heightWithHelp := m.viewport.Height

		if heightWithHelp >= heightWithoutHelp {
			t.Errorf("showHelp should reduce viewport height: withHelp=%d, withoutHelp=%d",
				heightWithHelp, heightWithoutHelp)
		}
	})
}

func TestLocalDir(t *testing.T) {
	m := pagerModel{
		currentDocument: markdown{localPath: "/home/user/docs/readme.md"},
	}

	got := m.localDir()
	want := "/home/user/docs"
	if got != want {
		t.Errorf("localDir() = %q, want %q", got, want)
	}
}

func TestFindMatches(t *testing.T) {
	content := "Hello World\nfoo bar\nHello again\nbaz\nhello lower"

	t.Run("basic match", func(t *testing.T) {
		matches := findMatches(content, "Hello")
		// Case-insensitive: should match lines 0, 2, 4
		if len(matches) != 3 {
			t.Errorf("findMatches() returned %d matches, want 3", len(matches))
		}
		if matches[0] != 0 || matches[1] != 2 || matches[2] != 4 {
			t.Errorf("findMatches() = %v, want [0, 2, 4]", matches)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		matches := findMatches(content, "hello")
		if len(matches) != 3 {
			t.Errorf("findMatches() returned %d matches, want 3", len(matches))
		}
	})

	t.Run("no matches", func(t *testing.T) {
		matches := findMatches(content, "xyz")
		if len(matches) != 0 {
			t.Errorf("findMatches() returned %d matches, want 0", len(matches))
		}
	})

	t.Run("empty query matches all lines", func(t *testing.T) {
		matches := findMatches(content, "")
		if len(matches) != 5 {
			t.Errorf("findMatches() returned %d matches, want 5", len(matches))
		}
	})

	t.Run("single line match", func(t *testing.T) {
		matches := findMatches(content, "baz")
		if len(matches) != 1 {
			t.Errorf("findMatches() returned %d matches, want 1", len(matches))
		}
		if matches[0] != 3 {
			t.Errorf("findMatches()[0] = %d, want 3", matches[0])
		}
	})
}

func TestInInputMode(t *testing.T) {
	t.Run("browse state no query", func(t *testing.T) {
		m := pagerModel{state: pagerStateBrowse}
		if m.inInputMode() {
			t.Error("inInputMode() should be false in browse with no query")
		}
	})

	t.Run("search state", func(t *testing.T) {
		m := pagerModel{state: pagerStateSearch}
		if !m.inInputMode() {
			t.Error("inInputMode() should be true in search state")
		}
	})

	t.Run("jump state", func(t *testing.T) {
		m := pagerModel{state: pagerStateJumpToLine}
		if !m.inInputMode() {
			t.Error("inInputMode() should be true in jump state")
		}
	})

	t.Run("browse with active search query", func(t *testing.T) {
		m := pagerModel{state: pagerStateBrowse, searchQuery: "foo"}
		if !m.inInputMode() {
			t.Error("inInputMode() should be true when searchQuery is set")
		}
	})

	t.Run("status message no query", func(t *testing.T) {
		m := pagerModel{state: pagerStateStatusMessage}
		if m.inInputMode() {
			t.Error("inInputMode() should be false in status message state with no query")
		}
	})
}

func TestPageIndicator(t *testing.T) {
	savedConfig := config
	t.Cleanup(func() { config = savedConfig })

	t.Run("multi-page document shows indicator", func(t *testing.T) {
		cfg := Config{}
		m := testPagerModel(80, 10, cfg)
		// Set content with many lines
		lines := strings.Repeat("line\n", 50)
		m.viewport.SetContent(lines)

		var b strings.Builder
		m.statusBarView(&b)
		got := b.String()
		if !strings.Contains(got, "pg") {
			t.Errorf("statusBarView() should contain page indicator for multi-page doc, got: %q", got)
		}
	})

	t.Run("single-page document no indicator", func(t *testing.T) {
		cfg := Config{}
		m := testPagerModel(80, 50, cfg)
		m.viewport.SetContent("short content")

		var b strings.Builder
		m.statusBarView(&b)
		got := b.String()
		// "pg" should not appear for single-page content
		if strings.Contains(got, " pg ") {
			t.Errorf("statusBarView() should not contain page indicator for single-page doc")
		}
	})
}

func TestSearchMatchCounter(t *testing.T) {
	savedConfig := config
	t.Cleanup(func() { config = savedConfig })

	cfg := Config{}
	m := testPagerModel(80, 24, cfg)
	m.searchQuery = "test"
	m.searchMatches = []int{0, 5, 10}
	m.searchIndex = 1

	var b strings.Builder
	m.statusBarView(&b)
	got := b.String()
	if !strings.Contains(got, "2/3") {
		t.Errorf("statusBarView() should contain match counter '2/3', got: %q", got)
	}
}

func TestClearSearch(t *testing.T) {
	m := pagerModel{
		searchQuery:   "test",
		searchMatches: []int{1, 2, 3},
		searchIndex:   1,
	}

	m.clearSearch()

	if m.searchQuery != "" {
		t.Errorf("clearSearch() should clear searchQuery, got %q", m.searchQuery)
	}
	if m.searchMatches != nil {
		t.Errorf("clearSearch() should clear searchMatches, got %v", m.searchMatches)
	}
	if m.searchIndex != -1 {
		t.Errorf("clearSearch() should set searchIndex to -1, got %d", m.searchIndex)
	}
}

func TestHelpViewContainsSearchAndJump(t *testing.T) {
	savedConfig := config
	t.Cleanup(func() { config = savedConfig })

	cfg := Config{}
	m := testPagerModel(80, 24, cfg)

	got := m.helpView()

	expectedBindings := []string{"/", "n/N", ":", "←/→"}
	for _, binding := range expectedBindings {
		if !strings.Contains(got, binding) {
			t.Errorf("helpView() should contain %q", binding)
		}
	}
}

func TestArrowKeyPaging(t *testing.T) {
	savedConfig := config
	t.Cleanup(func() { config = savedConfig })

	cfg := Config{}
	m := testPagerModel(80, 10, cfg)
	// 50 lines of content, viewport height 10 → multiple pages
	m.viewport.SetContent(strings.Repeat("line\n", 50))

	t.Run("right arrow pages forward", func(t *testing.T) {
		m.viewport.GotoTop()
		before := m.viewport.YOffset

		m.handleBrowseKeys(tea.KeyMsg{Type: tea.KeyRight})

		if m.viewport.YOffset <= before {
			t.Errorf("right arrow should page forward: before=%d, after=%d", before, m.viewport.YOffset)
		}
	})

	t.Run("left arrow pages backward", func(t *testing.T) {
		// Start partway down
		m.viewport.SetYOffset(20)
		before := m.viewport.YOffset

		m.handleBrowseKeys(tea.KeyMsg{Type: tea.KeyLeft})

		if m.viewport.YOffset >= before {
			t.Errorf("left arrow should page backward: before=%d, after=%d", before, m.viewport.YOffset)
		}
	})

	t.Run("left arrow at top stays at top", func(t *testing.T) {
		m.viewport.GotoTop()

		m.handleBrowseKeys(tea.KeyMsg{Type: tea.KeyLeft})

		if m.viewport.YOffset != 0 {
			t.Errorf("left arrow at top should stay at 0, got %d", m.viewport.YOffset)
		}
	})
}
