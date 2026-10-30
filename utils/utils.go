// Package utils provides utility functions.
package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/ansi"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/term"
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

const (
	codeBlockStartMarker = "\x1b]glow:cb-start\x07"
	codeBlockEndMarker   = "\x1b]glow:cb-end\x07"
)

// GlamourStyle returns a glamour.TermRendererOption based on the given style.
// When width is provided, horizontal rules are dynamically scaled to match the viewport.
func GlamourStyle(style string, isCode bool, width ...int) glamour.TermRendererOption {
	return func(tr *glamour.TermRenderer) error {
		styleConfig, err := styleConfigFor(style, isCode)
		if err != nil {
			return err
		}
		if !isCode {
			w := 0
			if len(width) > 0 {
				w = width[0]
			}
			applyDynamicHorizontalRule(&styleConfig, w)

			// Mark code blocks for background styling.
			styleConfig.CodeBlock.BlockPrefix = codeBlockStartMarker + styleConfig.CodeBlock.BlockPrefix
			styleConfig.CodeBlock.BlockSuffix = styleConfig.CodeBlock.BlockSuffix + codeBlockEndMarker

			if TextSizingEnabled() {
				AddHeadingSizeMarkers(&styleConfig)
			}
		}
		if err := glamour.WithStyles(styleConfig)(tr); err != nil {
			return fmt.Errorf("glamour: error applying styles: %w", err)
		}
		return nil
	}
}

// ApplyCodeBlockBackground wraps rendered code block lines with a subtle dark background.
func ApplyCodeBlockBackground(s string, isDark ...bool) string {
	if !strings.Contains(s, codeBlockStartMarker) {
		return s
	}

	bgCode := "233"
	if envBg := os.Getenv("GLOW_CODEBLOCK_BG"); envBg != "" {
		bgCode = envBg
	} else if len(isDark) > 0 && !isDark[0] {
		bgCode = "254"
	}
	bgSeq := "\x1b[48;5;" + bgCode + "m"

	lines := strings.Split(s, "\n")
	var result []string
	inCodeBlock := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		hasStart := strings.Contains(line, codeBlockStartMarker)
		hasEnd := strings.Contains(line, codeBlockEndMarker)

		if hasStart {
			inCodeBlock = true
			line = strings.ReplaceAll(line, codeBlockStartMarker, "")

			// Prepend a top padding line with background color to balance the bottom padding.
			prefix := ""
			content := line
			if strings.HasPrefix(line, "  ") {
				prefix = "  "
				content = line[2:]
			}
			width := lipgloss.Width(content)
			if width > 0 {
				topPadding := prefix + bgSeq + strings.Repeat(" ", width) + "\x1b[0m"
				result = append(result, topPadding)
			}
		}

		if inCodeBlock {
			line = strings.ReplaceAll(line, "\x1b[m", "\x1b[m"+bgSeq)
			line = strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+bgSeq)

			prefix := ""
			content := line
			if strings.HasPrefix(line, "  ") {
				prefix = "  "
				content = line[2:]
			}

			line = prefix + bgSeq + content + "\x1b[0m"
		}

		if hasEnd {
			inCodeBlock = false
			line = strings.ReplaceAll(line, codeBlockEndMarker, "")
		}

		result = append(result, line)

		// Ensure code block has an external bottom margin (blank line) if followed by non-empty line.
		if hasEnd && i+1 < len(lines) {
			if strings.TrimSpace(lines[i+1]) != "" {
				result = append(result, "")
			}
		}
	}

	return strings.Join(result, "\n")
}

// applyDynamicHorizontalRule replaces the default short dashes (--------) with
// a smooth full-width line matching the document viewport width.
func applyDynamicHorizontalRule(cfg *ansi.StyleConfig, width int) {
	if width <= 0 {
		width = defaultRenderWidth()
	}

	var margin int
	if cfg.Document.Margin != nil {
		margin = int(*cfg.Document.Margin)
	}
	ruleWidth := width - margin*2
	if ruleWidth < 10 {
		ruleWidth = 10
	}

	// Only replace if the format is empty or the default short dash.
	trimFmt := strings.TrimSpace(cfg.HorizontalRule.Format)
	if trimFmt == "" || trimFmt == "--------" {
		cfg.HorizontalRule.Format = "\n" + strings.Repeat("─", ruleWidth) + "\n"
	}

	if cfg.HorizontalRule.Color == nil || *cfg.HorizontalRule.Color == "" {
		dimColor := "240"
		cfg.HorizontalRule.Color = &dimColor
	}
}

func defaultRenderWidth() int {
	fd := int(os.Stdout.Fd())
	if term.IsTerminal(fd) {
		w, _, err := term.GetSize(fd)
		if err == nil && w > 0 {
			return w
		}
	}
	return 80
}

func styleConfigFor(style string, isCode bool) (ansi.StyleConfig, error) {
	if !isCode {
		if style == "auto" {
			style = styles.DarkStyle
		}
		if cfg, ok := styles.DefaultStyles[style]; ok {
			return *cfg, nil
		}
		return styleConfigFromFile(style)
	}

	// If we are rendering a pure code block, we need to modify the style to
	// remove the indentation.
	switch style {
	case "auto":
		if lipgloss.HasDarkBackground(os.Stdin, os.Stdout) {
			style = styles.DarkStyle
		} else {
			style = styles.LightStyle
		}
	case styles.TokyoNightStyle:
		style = styles.DraculaStyle
	case styles.DarkStyle, styles.LightStyle, styles.PinkStyle,
		styles.NoTTYStyle, styles.DraculaStyle:
	default:
		return styleConfigFromFile(style)
	}

	styleConfig := *styles.DefaultStyles[style]

	var margin uint
	styleConfig.CodeBlock.Margin = &margin

	return styleConfig, nil
}

func styleConfigFromFile(path string) (ansi.StyleConfig, error) {
	jsonBytes, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return ansi.StyleConfig{}, fmt.Errorf("glamour: error reading file: %w", err)
	}

	var styleConfig ansi.StyleConfig
	if err := json.Unmarshal(jsonBytes, &styleConfig); err != nil {
		return ansi.StyleConfig{}, fmt.Errorf("glamour: error parsing style file: %w", err)
	}
	return styleConfig, nil
}
