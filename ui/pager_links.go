package ui

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type followableLink struct {
	Href     string
	Path     string
	Fragment string
	Label    string

	ResolvedPath string
	ResolvedNote string
}

type rawLink struct {
	href  string
	label string
}

func followableLinksForDocument(rootDir, currentFilePath, markdown string) ([]followableLink, error) {
	raw := extractRawLinks(markdown)

	out := make([]followableLink, 0, len(raw))
	for _, l := range raw {
		link, ok, err := resolveFollowableLink(rootDir, currentFilePath, l.href)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if strings.TrimSpace(l.label) == "" {
			continue
		}
		link.Label = l.label
		out = append(out, link)
	}
	return out, nil
}

func splitFragment(href string) (path, frag string) {
	path, frag, ok := strings.Cut(href, "#")
	if ok {
		return path, frag
	}
	return href, ""
}

func isAbsoluteOrUNCPath(path string) bool {
	if strings.HasPrefix(path, "/") {
		return true
	}
	if strings.HasPrefix(path, `\\`) {
		return true
	}
	if len(path) >= 2 {
		c0 := path[0]
		if ((c0 >= 'a' && c0 <= 'z') || (c0 >= 'A' && c0 <= 'Z')) && path[1] == ':' {
			return true
		}
	}
	return filepath.IsAbs(path)
}

func isFollowableHref(href string) bool {
	href = strings.TrimSpace(href)
	href = strings.Trim(href, "<>")
	hrefLower := strings.ToLower(href)

	if strings.Contains(href, "://") || strings.HasPrefix(hrefLower, "mailto:") {
		return false
	}

	path, _ := splitFragment(href)
	if isAbsoluteOrUNCPath(path) {
		return false
	}
	pathLower := strings.ToLower(path)

	return strings.HasSuffix(pathLower, ".md") || strings.HasSuffix(pathLower, ".markdown")
}

func extractRawLinks(markdown string) []rawLink {
	source := []byte(markdown)
	parser := goldmark.New().Parser()
	doc := parser.Parse(text.NewReader(source))

	var out []rawLink
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		link, ok := n.(*ast.Link)
		if !ok {
			return ast.WalkContinue, nil
		}

		href := strings.TrimSpace(string(link.Destination))
		if href == "" {
			return ast.WalkContinue, nil
		}

		var b strings.Builder
		_ = ast.Walk(link, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			if t, ok := child.(*ast.Text); ok {
				b.Write(t.Segment.Value(source))
			}
			return ast.WalkContinue, nil
		})

		out = append(out, rawLink{
			href:  href,
			label: strings.TrimSpace(b.String()),
		})

		return ast.WalkContinue, nil
	})

	return out
}

func resolveFollowableLink(rootDir, currentFilePath, href string) (followableLink, bool, error) {
	href = strings.TrimSpace(href)
	href = strings.Trim(href, "<>")

	if !isFollowableHref(href) {
		return followableLink{}, false, nil
	}

	path, frag := splitFragment(href)
	path = strings.TrimSpace(path)
	if path == "" {
		return followableLink{}, false, nil
	}

	if strings.Contains(path, "%") {
		if decoded, err := url.PathUnescape(path); err == nil {
			path = decoded
		}
	}

	base := filepath.Dir(currentFilePath)
	resolved := filepath.Clean(filepath.Join(base, path))

	rootAbs, err := filepath.Abs(rootDir)
	if err != nil {
		return followableLink{}, false, fmt.Errorf("abs root dir: %w", err)
	}
	resAbs, err := filepath.Abs(resolved)
	if err != nil {
		return followableLink{}, false, fmt.Errorf("abs resolved path: %w", err)
	}

	if rootEval, err := filepath.EvalSymlinks(rootAbs); err == nil {
		rootAbs = rootEval
	}
	if resEval, err := filepath.EvalSymlinks(resAbs); err == nil {
		resAbs = resEval
	}

	rel, err := filepath.Rel(rootAbs, resAbs)
	if err != nil {
		return followableLink{}, false, nil
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return followableLink{}, false, nil
	}

	info, statErr := os.Stat(resAbs)
	if statErr != nil {
		return followableLink{}, false, nil
	}
	if !info.Mode().IsRegular() {
		return followableLink{}, false, nil
	}

	return followableLink{
		Href:         href,
		Path:         path,
		Fragment:     frag,
		ResolvedPath: resAbs,
		ResolvedNote: stripAbsolutePath(resAbs, rootAbs),
	}, true, nil
}
