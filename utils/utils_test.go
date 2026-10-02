package utils

import "testing"

func TestRemoveFrontmatter(t *testing.T) {
	for name, tc := range map[string]struct{ in, want string }{
		"none":                    {"# hi\n", "# hi\n"},
		"basic":                   {"---\ntitle: x\n---\n# hi\n", "# hi\n"},
		"crlf":                    {"---\r\ntitle: x\r\n---\r\n# hi\r\n", "# hi\r\n"},
		"closing marker at EOF":   {"---\ntitle: x\n---", ""},
		"closing marker at EOF 2": {"---\ntitle: x\r\n---\r", ""},
		"rule not at start":       {"# hi\n---\nfoo\n---\n", "# hi\n---\nfoo\n---\n"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := string(RemoveFrontmatter([]byte(tc.in))); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
