package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
)

func testStashModel(numMarkdowns, perPage int) stashModel {
	initSections()

	common := &commonModel{
		cfg:    Config{},
		width:  80,
		height: 40,
	}

	si := textinput.New()
	si.Prompt = "Find:"

	s := []section{
		sections[documentsSection],
	}
	// Set PerPage so we control pagination
	s[0].paginator.PerPage = perPage

	mds := make([]*markdown, numMarkdowns)
	for i := range mds {
		mds[i] = &markdown{
			Note: string(rune('a' + i%26)),
		}
		mds[i].buildFilterValue()
	}

	m := stashModel{
		common:      common,
		filterInput: si,
		sections:    s,
		markdowns:   mds,
	}

	// Set total pages based on markdowns
	if numMarkdowns > 0 {
		m.paginator().SetTotalPages(numMarkdowns)
	} else {
		m.paginator().SetTotalPages(1)
	}

	return m
}

func TestMoveCursorUp(t *testing.T) {
	t.Run("at top of first page stays 0", func(t *testing.T) {
		m := testStashModel(10, 5)
		m.setCursor(0)
		m.paginator().Page = 0

		m.moveCursorUp()

		if m.cursor() != 0 {
			t.Errorf("cursor = %d, want 0", m.cursor())
		}
	})

	t.Run("middle decrements", func(t *testing.T) {
		m := testStashModel(10, 5)
		m.setCursor(3)

		m.moveCursorUp()

		if m.cursor() != 2 {
			t.Errorf("cursor = %d, want 2", m.cursor())
		}
	})

	t.Run("top of page 2 goes to prev page", func(t *testing.T) {
		m := testStashModel(10, 5)
		m.paginator().Page = 1
		m.setCursor(0)

		m.moveCursorUp()

		if m.paginator().Page != 0 {
			t.Errorf("page = %d, want 0", m.paginator().Page)
		}
		// Cursor should be at last item of previous page
		if m.cursor() < 0 {
			t.Errorf("cursor = %d, should be >= 0", m.cursor())
		}
	})
}

func TestMoveCursorDown(t *testing.T) {
	t.Run("middle increments", func(t *testing.T) {
		m := testStashModel(10, 5)
		m.setCursor(2)

		m.moveCursorDown()

		if m.cursor() != 3 {
			t.Errorf("cursor = %d, want 3", m.cursor())
		}
	})

	t.Run("bottom of non-last page goes to next page", func(t *testing.T) {
		m := testStashModel(10, 5)
		m.setCursor(4) // last item on page (0-indexed, perPage=5)

		m.moveCursorDown()

		if m.paginator().Page != 1 {
			t.Errorf("page = %d, want 1", m.paginator().Page)
		}
		if m.cursor() != 0 {
			t.Errorf("cursor = %d, want 0", m.cursor())
		}
	})

	t.Run("bottom of last page stays", func(t *testing.T) {
		m := testStashModel(5, 5)
		m.setCursor(4) // last item, only one page

		m.moveCursorDown()

		if m.cursor() != 4 {
			t.Errorf("cursor = %d, want 4", m.cursor())
		}
	})
}

func TestUpdatePagination(t *testing.T) {
	t.Run("correct page count", func(t *testing.T) {
		m := testStashModel(10, 5)
		m.updatePagination()

		if m.paginator().TotalPages < 1 {
			t.Errorf("TotalPages = %d, want >= 1", m.paginator().TotalPages)
		}
	})

	t.Run("empty markdowns gives 1 page", func(t *testing.T) {
		m := testStashModel(0, 5)
		m.updatePagination()

		if m.paginator().TotalPages != 1 {
			t.Errorf("TotalPages = %d, want 1", m.paginator().TotalPages)
		}
	})
}

func TestFilterMarkdowns(t *testing.T) {
	t.Run("no filter returns all", func(t *testing.T) {
		m := testStashModel(5, 5)
		m.filterState = unfiltered

		cmd := filterMarkdowns(m)
		msg := cmd()

		filtered, ok := msg.(filteredMarkdownMsg)
		if !ok {
			t.Fatalf("expected filteredMarkdownMsg, got %T", msg)
		}
		if len(filtered) != 5 {
			t.Errorf("len(filtered) = %d, want 5", len(filtered))
		}
	})

	t.Run("fuzzy match", func(t *testing.T) {
		initSections()
		common := &commonModel{cfg: Config{}, width: 80, height: 40}
		si := textinput.New()
		si.Prompt = "Find:"
		si.SetValue("a")

		mds := []*markdown{
			{Note: "apple", filterValue: "apple"},
			{Note: "banana", filterValue: "banana"},
			{Note: "avocado", filterValue: "avocado"},
		}

		m := stashModel{
			common:      common,
			filterInput: si,
			filterState: filtering,
			markdowns:   mds,
			sections:    []section{sections[documentsSection]},
		}

		cmd := filterMarkdowns(m)
		msg := cmd()

		filtered, ok := msg.(filteredMarkdownMsg)
		if !ok {
			t.Fatalf("expected filteredMarkdownMsg, got %T", msg)
		}
		if len(filtered) == 0 {
			t.Error("expected some fuzzy matches, got 0")
		}
	})

	t.Run("no matches returns empty", func(t *testing.T) {
		initSections()
		common := &commonModel{cfg: Config{}, width: 80, height: 40}
		si := textinput.New()
		si.Prompt = "Find:"
		si.SetValue("zzzzzzz")

		mds := []*markdown{
			{Note: "apple", filterValue: "apple"},
			{Note: "banana", filterValue: "banana"},
		}

		m := stashModel{
			common:      common,
			filterInput: si,
			filterState: filtering,
			markdowns:   mds,
			sections:    []section{sections[documentsSection]},
		}

		cmd := filterMarkdowns(m)
		msg := cmd()

		filtered, ok := msg.(filteredMarkdownMsg)
		if !ok {
			t.Fatalf("expected filteredMarkdownMsg, got %T", msg)
		}
		if len(filtered) != 0 {
			t.Errorf("len(filtered) = %d, want 0", len(filtered))
		}
	})
}

func TestSelectedMarkdown(t *testing.T) {
	t.Run("valid cursor returns correct markdown", func(t *testing.T) {
		m := testStashModel(5, 5)
		m.setCursor(2)

		md := m.selectedMarkdown()
		if md == nil {
			t.Fatal("selectedMarkdown() returned nil")
		}
		if md != m.markdowns[2] {
			t.Error("selectedMarkdown() returned wrong markdown")
		}
	})

	t.Run("empty list returns nil", func(t *testing.T) {
		m := testStashModel(0, 5)

		md := m.selectedMarkdown()
		if md != nil {
			t.Errorf("selectedMarkdown() = %v, want nil", md)
		}
	})
}

func TestGetVisibleMarkdowns(t *testing.T) {
	t.Run("not filtering returns markdowns", func(t *testing.T) {
		m := testStashModel(5, 5)
		m.filterState = unfiltered

		got := m.getVisibleMarkdowns()
		if len(got) != 5 {
			t.Errorf("len(getVisibleMarkdowns()) = %d, want 5", len(got))
		}
	})

	t.Run("filtering returns filteredMarkdowns", func(t *testing.T) {
		m := testStashModel(5, 5)
		m.filterState = filtering
		m.filteredMarkdowns = []*markdown{
			{Note: "filtered1"},
			{Note: "filtered2"},
		}

		got := m.getVisibleMarkdowns()
		if len(got) != 2 {
			t.Errorf("len(getVisibleMarkdowns()) = %d, want 2", len(got))
		}
	})
}

func TestMarkdownIndex(t *testing.T) {
	tests := []struct {
		name    string
		page    int
		cursor  int
		perPage int
		want    int
	}{
		{"page 0 cursor 2 perPage 5", 0, 2, 5, 2},
		{"page 1 cursor 3 perPage 5", 1, 3, 5, 8},
		{"page 0 cursor 0 perPage 10", 0, 0, 10, 0},
		{"page 2 cursor 1 perPage 3", 2, 1, 3, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := testStashModel(20, tt.perPage)
			m.paginator().Page = tt.page
			m.paginator().PerPage = tt.perPage
			m.setCursor(tt.cursor)

			got := m.markdownIndex()
			if got != tt.want {
				t.Errorf("markdownIndex() = %d, want %d", got, tt.want)
			}
		})
	}
}
