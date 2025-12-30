// Package utils provides utility functions.
package utils

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// UTF-8 BOM bytes.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// ToUTF8String converts bytes to a UTF-8 string, handling UTF-16 LE/BE with BOM
// and UTF-8 with BOM. If no BOM is detected, the bytes are assumed to be UTF-8.
func ToUTF8String(data []byte) string {
	if len(data) < 2 {
		return string(data)
	}

	// UTF-16 LE BOM: 0xFF 0xFE
	if data[0] == 0xFF && data[1] == 0xFE {
		decoder := unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM).NewDecoder()
		result, _, err := transform.Bytes(decoder, data)
		if err == nil {
			return string(result)
		}
		// Fall through to return as-is on error
	}

	// UTF-16 BE BOM: 0xFE 0xFF
	if data[0] == 0xFE && data[1] == 0xFF {
		decoder := unicode.UTF16(unicode.BigEndian, unicode.ExpectBOM).NewDecoder()
		result, _, err := transform.Bytes(decoder, data)
		if err == nil {
			return string(result)
		}
		// Fall through to return as-is on error
	}

	// UTF-8 BOM: 0xEF 0xBB 0xBF - just strip it
	if bytes.HasPrefix(data, utf8BOM) {
		return string(data[3:])
	}

	return string(data)
}

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
		if style == styles.AutoStyle {
			return glamour.WithAutoStyle()
		}
		return glamour.WithStylePath(style)
	}

	// If we are rendering a pure code block, we need to modify the style to
	// remove the indentation.

	var styleConfig ansi.StyleConfig

	switch style {
	case styles.AutoStyle:
		if lipgloss.HasDarkBackground() {
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
