package utils

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
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

// Expands tilde and all environment variables from the given path.
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

func GlamourStyle(style string, isCode bool) glamour.TermRendererOption {
	if !isCode {
		if style == glamour.AutoStyle {
			return glamour.WithAutoStyle()
		} else {
			return glamour.WithStylePath(style)
		}
	}

	// If we are rendering a pure code block, we need to modify the style to
	// remove the indentation.

	var styleConfig ansi.StyleConfig

	switch style {
	case glamour.AutoStyle:
		if lipgloss.HasDarkBackground() {
			styleConfig = glamour.DarkStyleConfig
		} else {
			styleConfig = glamour.LightStyleConfig
		}
	case glamour.DarkStyle:
		styleConfig = glamour.DarkStyleConfig
	case glamour.LightStyle:
		styleConfig = glamour.LightStyleConfig
	case glamour.PinkStyle:
		styleConfig = glamour.PinkStyleConfig
	case glamour.NoTTYStyle:
		styleConfig = glamour.NoTTYStyleConfig
	case glamour.DraculaStyle:
		styleConfig = glamour.DraculaStyleConfig
	default:
		return glamour.WithStylesFromJSONFile(style)
	}

	var margin uint
	styleConfig.CodeBlock.Margin = &margin

	return glamour.WithStyles(styleConfig)
}
