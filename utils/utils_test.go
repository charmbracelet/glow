package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveFrontmatter(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "YAML frontmatter stripped",
			input: "---\ntitle: hello\n---\n# Body",
			want:  "# Body",
		},
		{
			name:  "no frontmatter unchanged",
			input: "# Just a heading\nSome text",
			want:  "# Just a heading\nSome text",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name:  "single delimiter not stripped",
			input: "---\nno closing delimiter",
			want:  "---\nno closing delimiter",
		},
		{
			name:  "frontmatter only at position 0",
			input: "some text\n---\ntitle: hello\n---\nbody",
			want:  "some text\n---\ntitle: hello\n---\nbody",
		},
		{
			name:  "frontmatter with blank line",
			input: "---\n\ntitle: hello\n---\n# Body",
			want:  "# Body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(RemoveFrontmatter([]byte(tt.input)))
			if got != tt.want {
				t.Errorf("RemoveFrontmatter() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsMarkdownFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"md extension", "README.md", true},
		{"mdown extension", "file.mdown", true},
		{"mkdn extension", "file.mkdn", true},
		{"mkd extension", "file.mkd", true},
		{"markdown extension", "file.markdown", true},
		{"go extension", "main.go", false},
		{"txt extension", "notes.txt", false},
		{"rs extension", "lib.rs", false},
		{"no extension", "Makefile", true},
		{"case insensitive MD", "README.MD", true},
		{"case insensitive Md", "file.Md", true},
		{"multi-dot md", "file.tar.md", true},
		{"multi-dot go", "file.test.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsMarkdownFile(tt.filename)
			if got != tt.want {
				t.Errorf("IsMarkdownFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestWrapCodeBlock(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		language string
		want     string
	}{
		{
			name:     "normal wrap",
			s:        "fmt.Println(\"hello\")\n",
			language: "go",
			want:     "```go\nfmt.Println(\"hello\")\n```",
		},
		{
			name:     "empty string",
			s:        "",
			language: "go",
			want:     "```go\n```",
		},
		{
			name:     "empty language",
			s:        "some code\n",
			language: "",
			want:     "```\nsome code\n```",
		},
		{
			name:     "multiline",
			s:        "line1\nline2\nline3\n",
			language: "python",
			want:     "```python\nline1\nline2\nline3\n```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WrapCodeBlock(tt.s, tt.language)
			if got != tt.want {
				t.Errorf("WrapCodeBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "tilde expansion",
			path: "~/foo",
			want: filepath.Join(home, "foo"),
		},
		{
			name: "absolute unchanged",
			path: "/usr/local/bin",
			want: "/usr/local/bin",
		},
		{
			name: "empty string",
			path: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandPath(tt.path)
			if got != tt.want {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}

	t.Run("env var expansion", func(t *testing.T) {
		t.Setenv("GLOW_TEST_DIR", "/tmp/glowtest")
		got := ExpandPath("$GLOW_TEST_DIR/foo")
		want := "/tmp/glowtest/foo"
		if got != want {
			t.Errorf("ExpandPath($GLOW_TEST_DIR/foo) = %q, want %q", got, want)
		}
	})
}

func TestGlamourStyle(t *testing.T) {
	tests := []struct {
		name   string
		style  string
		isCode bool
	}{
		{"dark style", "dark", false},
		{"light style", "light", false},
		{"notty style", "notty", false},
		{"dark style isCode", "dark", true},
		{"light style isCode", "light", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := GlamourStyle(tt.style, tt.isCode)
			if opt == nil {
				t.Errorf("GlamourStyle(%q, %v) returned nil", tt.style, tt.isCode)
			}
		})
	}
}
