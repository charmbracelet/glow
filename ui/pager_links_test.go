package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFollowableLinksForDocument_CommonFormatsAndSafety(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	outside := filepath.Join(base, "outside")

	mustMkdirAll(t, filepath.Join(root, "docs"))
	mustMkdirAll(t, outside)

	currentFilePath := filepath.Join(root, "current.md")
	mustWriteFile(t, currentFilePath, "# Current\n")

	targetMD := filepath.Join(root, "docs", "target.md")
	targetMarkdown := filepath.Join(root, "docs", "target.markdown")
	spaceNameMD := filepath.Join(root, "docs", "SPACE NAME.md")
	mustWriteFile(t, targetMD, "# Target\n")
	mustWriteFile(t, targetMarkdown, "# Target Markdown\n")
	mustWriteFile(t, spaceNameMD, "# Space Name\n")

	outsideMD := filepath.Join(outside, "outside.md")
	mustWriteFile(t, outsideMD, "# Outside\n")

	// A directory with a markdown-looking name should not be followable.
	dirLooksLikeMD := filepath.Join(root, "docs", "dir.md")
	mustMkdirAll(t, dirLooksLikeMD)

	rootAbs := absEvalSymlinks(t, root)

	type wantLink struct {
		Label        string
		ResolvedPath string
		ResolvedNote string
		Fragment     string
	}

	targetAbs := absEvalSymlinks(t, targetMD)
	targetMarkdownAbs := absEvalSymlinks(t, targetMarkdown)
	spaceNameAbs := absEvalSymlinks(t, spaceNameMD)

	cases := []struct {
		name  string
		md    string
		want  []wantLink
		setup func(t *testing.T)
	}{
		{
			name: "inline_relative_md",
			md:   "See [Target](docs/target.md).\n",
			want: []wantLink{{
				Label:        "Target",
				ResolvedPath: targetAbs,
				ResolvedNote: stripAbsolutePath(targetAbs, rootAbs),
			}},
		},
		{
			name: "reference_relative_md",
			md:   "See [Target][id].\n\n[id]: docs/target.md\n",
			want: []wantLink{{
				Label:        "Target",
				ResolvedPath: targetAbs,
				ResolvedNote: stripAbsolutePath(targetAbs, rootAbs),
			}},
		},
		{
			name: "collapsed_reference_relative_md",
			md:   "[Target][]\n\n[Target]: docs/target.md\n",
			want: []wantLink{{
				Label:        "Target",
				ResolvedPath: targetAbs,
				ResolvedNote: stripAbsolutePath(targetAbs, rootAbs),
			}},
		},
		{
			name: "relative_md_with_fragment",
			md:   "See [Target](docs/target.md#section).\n",
			want: []wantLink{{
				Label:        "Target",
				ResolvedPath: targetAbs,
				ResolvedNote: stripAbsolutePath(targetAbs, rootAbs),
				Fragment:     "section",
			}},
		},
		{
			name: "destination_in_angle_brackets",
			md:   "See [Target](<docs/target.md>).\n",
			want: []wantLink{{
				Label:        "Target",
				ResolvedPath: targetAbs,
				ResolvedNote: stripAbsolutePath(targetAbs, rootAbs),
			}},
		},
		{
			name: "url_escaped_path_is_unescaped",
			md:   "See [Space](docs/SPACE%20NAME.md).\n",
			want: []wantLink{{
				Label:        "Space",
				ResolvedPath: spaceNameAbs,
				ResolvedNote: stripAbsolutePath(spaceNameAbs, rootAbs),
			}},
		},
		{
			name: "relative_markdown_extension",
			md:   "See [Target](docs/target.markdown).\n",
			want: []wantLink{{
				Label:        "Target",
				ResolvedPath: targetMarkdownAbs,
				ResolvedNote: stripAbsolutePath(targetMarkdownAbs, rootAbs),
			}},
		},
		{
			name: "empty_label_is_ignored",
			md:   "See [](docs/target.md).\n",
			want: nil,
		},
		{
			name: "image_is_ignored",
			md:   "![Alt](docs/target.md)\n",
			want: nil,
		},
		{
			name: "autolink_relative_is_ignored",
			md:   "See <docs/target.md>.\n",
			want: nil,
		},
		{
			name: "bare_path_is_ignored",
			md:   "docs/target.md\n",
			want: nil,
		},
		{
			name: "external_http_is_ignored",
			md:   "See [Ext](https://example.com/docs/target.md).\n",
			want: nil,
		},
		{
			name: "mailto_is_ignored",
			md:   "See [Mail](mailto:test@example.com).\n",
			want: nil,
		},
		{
			name: "non_markdown_extension_is_ignored",
			md:   "See [Txt](docs/target.txt).\n",
			want: nil,
		},
		{
			name: "missing_file_is_ignored",
			md:   "See [Missing](docs/missing.md).\n",
			want: nil,
		},
		{
			name: "directory_is_ignored_even_if_md_suffix",
			md:   "See [Dir](docs/dir.md).\n",
			want: nil,
		},
		{
			name: "absolute_unix_path_is_ignored",
			md:   "See [Abs](/etc/passwd).\n",
			want: nil,
		},
		{
			name: "windows_drive_absolute_is_ignored",
			md:   `See [WinAbs](C:\\Windows\\system32\\a.md).`,
			want: nil,
		},
		{
			name: "windows_unc_absolute_is_ignored",
			md:   `See [UNC](\\\\server\\share\\a.md).`,
			want: nil,
		},
		{
			name: "root_escape_via_dotdot_is_ignored",
			md:   "See [Escape](../outside/outside.md).\n",
			want: nil,
		},
		{
			name: "root_escape_via_symlink_is_ignored",
			md:   "See [Escape](escape/outside.md).\n",
			want: nil,
			setup: func(t *testing.T) {
				t.Helper()
				escape := filepath.Join(root, "escape")
				if err := os.Symlink(outside, escape); err != nil {
					t.Skipf("symlink not supported: %v", err)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(t)
			}

			got, err := followableLinksForDocument(root, currentFilePath, tc.md)
			if err != nil {
				t.Fatalf("followableLinksForDocument returned error: %v", err)
			}

			if len(got) != len(tc.want) {
				t.Fatalf("expected %d links, got %d: %+v", len(tc.want), len(got), got)
			}

			for i := range tc.want {
				if got[i].Label != tc.want[i].Label {
					t.Fatalf("link[%d] label: expected %q, got %q", i, tc.want[i].Label, got[i].Label)
				}
				if got[i].ResolvedPath != tc.want[i].ResolvedPath {
					t.Fatalf("link[%d] resolved path: expected %q, got %q", i, tc.want[i].ResolvedPath, got[i].ResolvedPath)
				}
				if got[i].ResolvedNote != tc.want[i].ResolvedNote {
					t.Fatalf("link[%d] resolved note: expected %q, got %q", i, tc.want[i].ResolvedNote, got[i].ResolvedNote)
				}
				if got[i].Fragment != tc.want[i].Fragment {
					t.Fatalf("link[%d] fragment: expected %q, got %q", i, tc.want[i].Fragment, got[i].Fragment)
				}
			}
		})
	}
}

func absEvalSymlinks(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("abs %q: %v", path, err)
	}
	if eval, err := filepath.EvalSymlinks(abs); err == nil {
		return eval
	}
	return abs
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdirall %q: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path, contents string) {
	t.Helper()
	mustMkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("writefile %q: %v", path, err)
	}
}
