package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

var styleOutputPath string

var styleCmd = &cobra.Command{
	Use:   "style",
	Short: "Manage glow color styles",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

var styleInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a custom color style interactively",
	Long: paragraph(fmt.Sprintf(
		"\nWalk through interactive prompts to create a custom %s JSON stylesheet with your preferred colors.\n\nThe generated file can be used with %s.",
		keyword("glamour"),
		keyword("glow -s path/to/style.json"),
	)),
	Example: paragraph("  glow style init\n  glow style init -o ~/mytheme.json"),
	Args:    cobra.NoArgs,
	RunE: func(*cobra.Command, []string) error {
		return runStyleInit()
	},
}

func init() {
	styleInitCmd.Flags().StringVarP(&styleOutputPath, "output", "o", "", "output path for the style JSON (default ~/.config/glow/style.json)")
	styleCmd.AddCommand(styleInitCmd)
}

func runStyleInit() error {
	if styleOutputPath == "" {
		home, err := homedir.Dir()
		if err != nil {
			return fmt.Errorf("unable to determine home directory: %w", err)
		}
		styleOutputPath = filepath.Join(home, ".config", "glow", "style.json")
	}

	fmt.Println()
	fmt.Println("  " + keyword("glow style init"))
	fmt.Println("  " + strings.Repeat("─", 40))
	fmt.Println("  Press Enter to accept the default value shown in brackets.")
	fmt.Println()

	isDark := promptChoose("Base scheme", "1", []string{
		"Dark background",
		"Light background",
	}) == "1"

	docColor := promptColor("Document text color", "252", isDark)
	h1Color := promptColor("H1 text color", "228", isDark)
	h1Bg := promptColor("H1 background color", "63", isDark)
	h1Bold := promptBool("H1 bold", true)
	h2Color := promptColor("H2 text color", "39", isDark)
	h3Color := promptColor("H3 text color", "39", isDark)
	h6Color := promptColor("H6 text color", "35", isDark)
	codeColor := promptColor("Inline code text color", "203", isDark)
	codeBg := promptColor("Inline code background color", "236", isDark)
	codeBlockColor := promptColor("Code block text color", "244", isDark)
	linkColor := promptColor("Link color", "30", isDark)
	linkUnderline := promptBool("Underline links", true)
	hrColor := promptColor("Horizontal rule color", "240", isDark)
	blockQuoteToken := prompt("Blockquote indent token", "│ ")

	style := buildStyle(docColor, h1Color, h1Bg, h1Bold, h2Color, h3Color, h6Color,
		codeColor, codeBg, codeBlockColor, linkColor, linkUnderline, hrColor, blockQuoteToken)

	data, err := json.MarshalIndent(style, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshal style: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(styleOutputPath), 0o700); err != nil {
		return fmt.Errorf("unable to create directory: %w", err)
	}

	if err := os.WriteFile(styleOutputPath, data, 0o600); err != nil {
		return fmt.Errorf("unable to write style file: %w", err)
	}

	fmt.Println()
	fmt.Println("  ✓ Style saved to: " + styleOutputPath)
	fmt.Println("  Use it with: " + keyword("glow -s "+styleOutputPath))
	fmt.Println()

	return nil
}

func prompt(label, defaultVal string) string {
	fmt.Printf("  %s [%s]: ", label, defaultVal)
	var input string
	_, _ = fmt.Scanln(&input)
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func promptColor(label, defaultColor string, isDark bool) string {
	hint := defaultColor
	fmt.Printf("  %s [%s]: ", label, hint)
	var input string
	_, _ = fmt.Scanln(&input)
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultColor
	}
	return input
}

func promptBool(label string, defaultVal bool) bool {
	defaultStr := "y"
	if !defaultVal {
		defaultStr = "n"
	}
	fmt.Printf("  %s? [%s]: ", label, defaultStr)
	var input string
	_, _ = fmt.Scanln(&input)
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultVal
	}
	return input == "y" || input == "yes"
}

func promptChoose(label, defaultVal string, options []string) string {
	fmt.Println("  " + label + ":")
	for i, opt := range options {
		fmt.Printf("    %d) %s\n", i+1, opt)
	}
	fmt.Printf("  Enter 1-%d [%s]: ", len(options), defaultVal)
	var input string
	_, _ = fmt.Scanln(&input)
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func buildStyle(docColor, h1Color, h1Bg string, h1Bold bool, h2Color, h3Color, h6Color string,
	codeColor, codeBg, codeBlockColor, linkColor string, linkUnderline bool, hrColor, blockQuoteToken string) map[string]interface{} {

	style := map[string]interface{}{
		"document": map[string]interface{}{
			"block_prefix": "\n",
			"block_suffix": "\n",
			"color":        docColor,
			"margin":       2,
		},
		"block_quote": map[string]interface{}{
			"indent":       1,
			"indent_token": blockQuoteToken,
		},
		"paragraph": map[string]interface{}{},
		"list": map[string]interface{}{
			"level_indent": 2,
		},
		"heading": map[string]interface{}{
			"block_suffix": "\n",
			"color":        h2Color,
			"bold":         true,
		},
		"h1": map[string]interface{}{
			"prefix":           " ",
			"suffix":           " ",
			"color":            h1Color,
			"background_color": h1Bg,
			"bold":             h1Bold,
		},
		"h2": map[string]interface{}{
			"prefix": "## ",
		},
		"h3": map[string]interface{}{
			"prefix": "### ",
			"color":  h3Color,
		},
		"h4": map[string]interface{}{
			"prefix": "#### ",
		},
		"h5": map[string]interface{}{
			"prefix": "##### ",
		},
		"h6": map[string]interface{}{
			"prefix": "###### ",
			"color":  h6Color,
			"bold":   false,
		},
		"text": map[string]interface{}{},
		"strikethrough": map[string]interface{}{
			"crossed_out": true,
		},
		"emph": map[string]interface{}{
			"italic": true,
		},
		"strong": map[string]interface{}{
			"bold": true,
		},
		"hr": map[string]interface{}{
			"color":  hrColor,
			"format": "\n--------\n",
		},
		"item": map[string]interface{}{
			"block_prefix": "• ",
		},
		"enumeration": map[string]interface{}{
			"block_prefix": ". ",
		},
		"task": map[string]interface{}{
			"ticked":   "[✓] ",
			"unticked": "[ ] ",
		},
		"link": map[string]interface{}{
			"color":     linkColor,
			"underline": linkUnderline,
		},
		"link_text": map[string]interface{}{
			"color": "35",
			"bold":  true,
		},
		"image": map[string]interface{}{
			"color":     "212",
			"underline": true,
		},
		"image_text": map[string]interface{}{
			"color":  "243",
			"format": "Image: {{.text}} →",
		},
		"code": map[string]interface{}{
			"prefix":           "\u00a0",
			"suffix":           "\u00a0",
			"color":            codeColor,
			"background_color": codeBg,
		},
		"code_block": map[string]interface{}{
			"color":  codeBlockColor,
			"margin": 2,
			"chroma": map[string]interface{}{
				"text":                  map[string]interface{}{"color": "#C4C4C4"},
				"error":                 map[string]interface{}{"color": "#F1F1F1", "background_color": "#F05B5B"},
				"comment":               map[string]interface{}{"color": "#676767"},
				"comment_preproc":       map[string]interface{}{"color": "#FF875F"},
				"keyword":               map[string]interface{}{"color": "#00AAFF"},
				"keyword_reserved":      map[string]interface{}{"color": "#FF5FD2"},
				"keyword_namespace":     map[string]interface{}{"color": "#FF5F87"},
				"keyword_type":          map[string]interface{}{"color": "#6E6ED8"},
				"operator":              map[string]interface{}{"color": "#EF8080"},
				"punctuation":           map[string]interface{}{"color": "#E8E8A8"},
				"name":                  map[string]interface{}{"color": "#C4C4C4"},
				"name_builtin":          map[string]interface{}{"color": "#FF8EC7"},
				"name_tag":              map[string]interface{}{"color": "#B083EA"},
				"name_attribute":        map[string]interface{}{"color": "#7A7AE6"},
				"name_class":            map[string]interface{}{"color": "#F1F1F1", "underline": true, "bold": true},
				"name_decorator":        map[string]interface{}{"color": "#FFFF87"},
				"name_function":         map[string]interface{}{"color": "#00D787"},
				"literal_number":        map[string]interface{}{"color": "#6EEFC0"},
				"literal_string":        map[string]interface{}{"color": "#C69669"},
				"literal_string_escape": map[string]interface{}{"color": "#AFFFD7"},
				"generic_deleted":       map[string]interface{}{"color": "#FD5B5B"},
				"generic_emph":          map[string]interface{}{"italic": true},
				"generic_inserted":      map[string]interface{}{"color": "#00D787"},
				"generic_strong":        map[string]interface{}{"bold": true},
				"generic_subheading":    map[string]interface{}{"color": "#777777"},
				"background":            map[string]interface{}{"background_color": "#373737"},
			},
		},
		"table":           map[string]interface{}{},
		"definition_list": map[string]interface{}{},
		"definition_term": map[string]interface{}{},
		"definition_description": map[string]interface{}{
			"block_prefix": "\n🠶 ",
		},
		"html_block": map[string]interface{}{},
		"html_span":  map[string]interface{}{},
	}

	return style
}
