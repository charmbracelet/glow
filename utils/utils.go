// Package utils provides utility functions.
package utils

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/ansi"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
	"github.com/mitchellh/go-homedir"
)

// RemoveFrontmatter removes the front matter header of a markdown file.
func RemoveFrontmatter(content []byte) []byte {
	if frontmatterBoundaries := detectFrontmatter(content); frontmatterBoundaries[0] == 0 {
		return content[frontmatterBoundaries[1]:]
	}
	return content
}

var yamlPattern = regexp.MustCompile(`(?m)^---\r?\n(\s*\r?\n)?`)

func detectFrontmatter(c []byte) []int {
	if matches := yamlPattern.FindAllIndex(c, 2); len(matches) > 1 {
		return []int{matches[0][0], matches[1][1]}
	}
	return []int{-1, -1}
}

// ExpandPath expands tilde and all environment variables from the given path.
func ExpandPath(path string) string {
	s, err := homedir.Expand(path)
	if err == nil {
		return os.ExpandEnv(s)
	}
	return os.ExpandEnv(path)
}

// WrapCodeBlock wraps a string in a code block with the given language.
func WrapCodeBlock(s, language string) string {
	return "```" + language + "\n" + s + "```"
}

var markdownExtensions = []string{
	".md", ".mdown", ".mkdn", ".mkd", ".markdown",
}

// IsMarkdownFile returns whether the filename has a markdown extension.
func IsMarkdownFile(filename string) bool {
	ext := filepath.Ext(filename)

	if ext == "" {
		// By default, assume it's a markdown file.
		return true
	}

	for _, v := range markdownExtensions {
		if strings.EqualFold(ext, v) {
			return true
		}
	}

	// Has an extension but not markdown
	// so assume this is a code file.
	return false
}

// GlamourStyle returns a glamour.TermRendererOption based on the given style.
func GlamourStyle(style string, isCode bool) glamour.TermRendererOption {
	if !isCode {
		if style == "auto" {
			return glamour.WithStandardStyle("dark")
		}
		return glamour.WithStylePath(style)
	}

	// If we are rendering a pure code block, we need to modify the style to
	// remove the indentation.

	var styleConfig ansi.StyleConfig

	switch style {
	case "auto":
		if lipgloss.HasDarkBackground(os.Stdin, os.Stdout) {
			styleConfig = styles.DarkStyleConfig
		} else {
			styleConfig = styles.LightStyleConfig
		}
	case styles.DarkStyle:
		styleConfig = styles.DarkStyleConfig
	case styles.LightStyle:
		styleConfig = styles.LightStyleConfig
	case styles.PinkStyle:
		styleConfig = styles.PinkStyleConfig
	case styles.NoTTYStyle:
		styleConfig = styles.NoTTYStyleConfig
	case styles.DraculaStyle:
		styleConfig = styles.DraculaStyleConfig
	case styles.TokyoNightStyle:
		styleConfig = styles.DraculaStyleConfig
	default:
		return glamour.WithStylesFromJSONFile(style)
	}

	var margin uint
	styleConfig.CodeBlock.Margin = &margin

	return glamour.WithStyles(styleConfig)
}

// SupportsHyperlinks reports whether the terminal is known to support OSC 8
// hyperlinks, based on environment variables.
func SupportsHyperlinks(env []string) bool {
	get := func(key string) string {
		for _, e := range env {
			if k, v, ok := strings.Cut(e, "="); ok && k == key {
				return v
			}
		}
		return ""
	}

	// Explicit opt-out.
	if get("TERM_PROGRAM") == "Apple_Terminal" {
		return false
	}
	// Terminals known to support OSC 8 hyperlinks.
	switch get("TERM_PROGRAM") {
	case "iTerm.app", "WezTerm", "kitty", "alacritty", "Hyper", "ghostty", "vscode", "Tabby", " Rio":
		return true
	}
	if get("WT_SESSION") != "" { // Windows Terminal
		return true
	}
	if get("KONSOLE_VERSION") != "" { // Konsole
		return true
	}
	if get("VTE_VERSION") != "" { // GNOME Terminal, Tilix, etc.
		return true
	}
	if get("TERM") == "xterm-kitty" {
		return true
	}
	return false
}
