package ui

import "testing"

func TestExceedsMaxDepth(t *testing.T) {
	tests := []struct {
		name     string
		relPath  string
		maxDepth int
		want     bool
	}{
		{"unlimited", "a/b/c/README.md", -1, false},
		{"root file within limit", "README.md", 0, false},
		{"nested file exceeds root-only limit", "docs/README.md", 0, true},
		{"nested file within limit", "docs/README.md", 1, false},
		{"deeply nested file exceeds limit", "a/b/c/README.md", 1, true},
		{"deeply nested file within limit", "a/b/c/README.md", 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := exceedsMaxDepth(tt.relPath, tt.maxDepth); got != tt.want {
				t.Errorf("exceedsMaxDepth(%q, %d) = %v, want %v", tt.relPath, tt.maxDepth, got, tt.want)
			}
		})
	}
}
