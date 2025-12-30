package utils

import (
	"testing"
)

func TestToUTF8String(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "plain UTF-8",
			input:    []byte("Hello, World!"),
			expected: "Hello, World!",
		},
		{
			name:     "UTF-8 with BOM",
			input:    []byte{0xEF, 0xBB, 0xBF, 'H', 'e', 'l', 'l', 'o'},
			expected: "Hello",
		},
		{
			name: "UTF-16 LE with BOM",
			input: []byte{
				0xFF, 0xFE, // BOM
				'H', 0x00, 'e', 0x00, 'l', 0x00, 'l', 0x00, 'o', 0x00,
			},
			expected: "Hello",
		},
		{
			name: "UTF-16 BE with BOM",
			input: []byte{
				0xFE, 0xFF, // BOM
				0x00, 'H', 0x00, 'e', 0x00, 'l', 0x00, 'l', 0x00, 'o',
			},
			expected: "Hello",
		},
		{
			name:     "empty input",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "single byte",
			input:    []byte{'A'},
			expected: "A",
		},
		{
			name: "UTF-16 LE with markdown content",
			input: []byte{
				0xFF, 0xFE, // BOM
				'#', 0x00, ' ', 0x00, 'T', 0x00, 'e', 0x00, 's', 0x00, 't', 0x00,
			},
			expected: "# Test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ToUTF8String(tc.input)
			if result != tc.expected {
				t.Errorf("ToUTF8String(%v) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestRemoveFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "no frontmatter",
			input:    []byte("# Hello\n\nWorld"),
			expected: []byte("# Hello\n\nWorld"),
		},
		{
			name:     "with frontmatter",
			input:    []byte("---\ntitle: Test\n---\n# Hello"),
			expected: []byte("# Hello"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := RemoveFrontmatter(tc.input)
			if string(result) != string(tc.expected) {
				t.Errorf("RemoveFrontmatter(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}
