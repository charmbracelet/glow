// Package utils provides utility functions.
package utils

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
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
	var styleConfig ansi.StyleConfig
	var useBuiltinStyle bool

	switch style {
	case styles.AutoStyle:
		if lipgloss.HasDarkBackground() {
			styleConfig = styles.DarkStyleConfig
		} else {
			styleConfig = styles.LightStyleConfig
		}
		useBuiltinStyle = true
	case styles.DarkStyle:
		styleConfig = styles.DarkStyleConfig
		useBuiltinStyle = true
	case styles.LightStyle:
		styleConfig = styles.LightStyleConfig
		useBuiltinStyle = true
	case styles.PinkStyle:
		styleConfig = styles.PinkStyleConfig
		useBuiltinStyle = true
	case styles.NoTTYStyle:
		styleConfig = styles.NoTTYStyleConfig
		useBuiltinStyle = true
	case styles.DraculaStyle:
		styleConfig = styles.DraculaStyleConfig
		useBuiltinStyle = true
	case styles.TokyoNightStyle:
		styleConfig = styles.DraculaStyleConfig
		useBuiltinStyle = true
	default:
		return glamour.WithStylesFromJSONFile(style)
	}

	if useBuiltinStyle {
		// Fix link color for better contrast against H1 background.
		// The default link colors (e.g. "30" for dark) have poor contrast
		// against the H1 background color ("63").
		// Using "123" provides better readability.
		linkColor := "123"
		styleConfig.Link.Color = &linkColor
	}

	if isCode {
		// If we are rendering a pure code block, we need to modify the style
		// to remove the indentation.
		var margin uint
		styleConfig.CodeBlock.Margin = &margin
	}

	return glamour.WithStyles(styleConfig)
}
