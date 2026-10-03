package main

import (
	"testing"
)

func TestGlowFlags(t *testing.T) {
	tt := []struct {
		args  []string
		check func() bool
	}{
		{
			args: []string{"-p"},
			check: func() bool {
				return pager
			},
		},
		{
			args: []string{"-s", "light"},
			check: func() bool {
				return style == "light"
			},
		},
		{
			args: []string{"-w", "40"},
			check: func() bool {
				return width == 40
			},
		},
	}

	for _, v := range tt {
		err := rootCmd.ParseFlags(v.args)
		if err != nil {
			t.Fatal(err)
		}
		if !v.check() {
			t.Errorf("Parsing flag failed: %s", v.args)
		}
	}
}

func TestBuildStyle(t *testing.T) {
	style := buildStyle("252", "228", "63", true, "39", "39", "35",
		"203", "236", "244", "30", true, "240", "│ ")

	tests := []struct {
		key      string
		field    string
		expected interface{}
	}{
		{"document", "color", "252"},
		{"document", "margin", 2},
		{"h1", "color", "228"},
		{"h1", "background_color", "63"},
		{"h1", "bold", true},
		{"h2", "prefix", "## "},
		{"code", "color", "203"},
		{"code", "background_color", "236"},
		{"link", "color", "30"},
		{"link", "underline", true},
		{"hr", "color", "240"},
		{"block_quote", "indent_token", "│ "},
		{"strong", "bold", true},
		{"emph", "italic", true},
		{"strikethrough", "crossed_out", true},
		{"task", "ticked", "[✓] "},
		{"task", "unticked", "[ ] "},
	}

	for _, tt := range tests {
		section, ok := style[tt.key].(map[string]interface{})
		if !ok {
			t.Errorf("missing style section: %s", tt.key)
			continue
		}
		got, ok := section[tt.field]
		if !ok {
			t.Errorf("missing field %s in section %s", tt.field, tt.key)
			continue
		}
		if got != tt.expected {
			t.Errorf("style[%s].%s = %v, want %v", tt.key, tt.field, got, tt.expected)
		}
	}
}

func TestBuildStyleCustomColors(t *testing.T) {
	style := buildStyle("#FF0000", "#00FF00", "#0000FF", false, "#FF00FF", "#FFFF00", "#00FFFF",
		"#FFA500", "#800080", "#A52A2A", "#FFC0CB", false, "#808080", "> ")

	doc := style["document"].(map[string]interface{})
	if doc["color"] != "#FF0000" {
		t.Errorf("document.color = %v, want #FF0000", doc["color"])
	}

	h1 := style["h1"].(map[string]interface{})
	if h1["color"] != "#00FF00" {
		t.Errorf("h1.color = %v, want #00FF00", h1["color"])
	}
	if h1["bold"] != false {
		t.Errorf("h1.bold = %v, want false", h1["bold"])
	}

	link := style["link"].(map[string]interface{})
	if link["underline"] != false {
		t.Errorf("link.underline = %v, want false", link["underline"])
	}

	bq := style["block_quote"].(map[string]interface{})
	if bq["indent_token"] != "> " {
		t.Errorf("block_quote.indent_token = %v, want '> '", bq["indent_token"])
	}
}
