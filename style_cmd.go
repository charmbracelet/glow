package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

type theme struct {
	Name        string
	Description string
	Doc         string
	H1          string
	H1Bg        string
	H1Bold      bool
	H2          string
	H3          string
	H6          string
	Code        string
	CodeBg      string
	CodeBlock   string
	Link        string
	LinkUnder   bool
	HR          string
	BqToken     string
}

var themes = []theme{
	{Name: "Catppuccin Mocha", Description: "Dark, warm purple-pink tones", Doc: "252", H1: "228", H1Bg: "63", H1Bold: true, H2: "39", H3: "39", H6: "35", Code: "203", CodeBg: "236", CodeBlock: "244", Link: "30", LinkUnder: true, HR: "240", BqToken: "│ "},
	{Name: "Catppuccin Latte", Description: "Light, warm purple-pink tones", Doc: "234", H1: "228", H1Bg: "63", H1Bold: true, H2: "27", H3: "27", H6: "35", Code: "203", CodeBg: "254", CodeBlock: "242", Link: "36", LinkUnder: true, HR: "249", BqToken: "│ "},
	{Name: "Nord", Description: "Dark, arctic blue tones", Doc: "252", H1: "228", H1Bg: "67", H1Bold: true, H2: "110", H3: "110", H6: "150", Code: "203", CodeBg: "236", CodeBlock: "244", Link: "110", LinkUnder: true, HR: "240", BqToken: "│ "},
	{Name: "Gruvbox Dark", Description: "Dark, retro warm tones", Doc: "223", H1: "228", H1Bg: "88", H1Bold: true, H2: "214", H3: "214", H6: "108", Code: "203", CodeBg: "237", CodeBlock: "244", Link: "109", LinkUnder: true, HR: "240", BqToken: "│ "},
	{Name: "Gruvbox Light", Description: "Light, retro warm tones", Doc: "237", H1: "228", H1Bg: "88", H1Bold: true, H2: "214", H3: "214", H6: "108", Code: "203", CodeBg: "254", CodeBlock: "242", Link: "109", LinkUnder: true, HR: "249", BqToken: "│ "},
	{Name: "Solarized Dark", Description: "Dark, sepia-toned", Doc: "252", H1: "228", H1Bg: "33", H1Bold: true, H2: "37", H3: "37", H6: "64", Code: "203", CodeBg: "236", CodeBlock: "244", Link: "33", LinkUnder: true, HR: "240", BqToken: "│ "},
	{Name: "Solarized Light", Description: "Light, sepia-toned", Doc: "234", H1: "228", H1Bg: "33", H1Bold: true, H2: "37", H3: "37", H6: "64", Code: "203", CodeBg: "254", CodeBlock: "242", Link: "33", LinkUnder: true, HR: "249", BqToken: "│ "},
	{Name: "Tokyo Night", Description: "Dark, vibrant neon", Doc: "252", H1: "228", H1Bg: "99", H1Bold: true, H2: "147", H3: "147", H6: "150", Code: "203", CodeBg: "236", CodeBlock: "244", Link: "147", LinkUnder: true, HR: "240", BqToken: "│ "},
	{Name: "Dracula", Description: "Dark, purple accents", Doc: "252", H1: "228", H1Bg: "99", H1Bold: true, H2: "212", H3: "212", H6: "150", Code: "203", CodeBg: "236", CodeBlock: "244", Link: "212", LinkUnder: true, HR: "240", BqToken: "│ "},
	{Name: "One Dark", Description: "Dark, balanced blue-gray", Doc: "252", H1: "228", H1Bg: "67", H1Bold: true, H2: "75", H3: "75", H6: "150", Code: "203", CodeBg: "236", CodeBlock: "244", Link: "75", LinkUnder: true, HR: "240", BqToken: "│ "},
	{Name: "Random", Description: "Surprise me — auto-generated palette", Doc: "252", H1: "228", H1Bg: "63", H1Bold: true, H2: "39", H3: "39", H6: "35", Code: "203", CodeBg: "236", CodeBlock: "244", Link: "30", LinkUnder: true, HR: "240", BqToken: "│ "},
}

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
	Short: "Create a color style from a theme palette",
	Long: paragraph(fmt.Sprintf(
		"\nChoose from preset color themes or generate a random palette. Preview the result and save it.\n\nThe generated file can be used with %s.",
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

	for {
		fmt.Println()
		fmt.Println("  " + keyword("Pick a theme"))
		fmt.Println("  " + strings.Repeat("─", 50))
		for i, t := range themes {
			fmt.Printf("  %2d) %-20s %s\n", i+1, t.Name, t.Description)
		}
		fmt.Printf("  %2d) %-20s %s\n", 0, "Quit", "exit without saving")
		fmt.Print("\n  Your choice [1]: ")

		var choice int
		input := ""
		_, _ = fmt.Scanln(&input)
		input = strings.TrimSpace(input)
		if input == "" {
			choice = 1
		} else {
			_, err := fmt.Sscanf(input, "%d", &choice)
			if err != nil || choice < 0 || choice > len(themes) {
				fmt.Println("  Invalid choice. Try again.")
				continue
			}
		}
		if choice == 0 {
			fmt.Println("  " + keyword("Bye!"))
			return nil
		}

		t := themes[choice-1]
		if t.Name == "Random" {
			t = randomTheme()
		}

		style := buildStyle(t.Doc, t.H1, t.H1Bg, t.H1Bold, t.H2, t.H3, t.H6,
			t.Code, t.CodeBg, t.CodeBlock, t.Link, t.LinkUnder, t.HR, t.BqToken)

		data, err := json.MarshalIndent(style, "", "  ")
		if err != nil {
			return fmt.Errorf("unable to marshal style: %w", err)
		}

		fmt.Println()
		fmt.Println("  " + keyword(t.Name))
		fmt.Println("  " + strings.Repeat("─", 50))

		showPreview(data)

		fmt.Println()
		fmt.Println("  " + strings.Repeat("─", 50))
		fmt.Print("  [S]ave  [R]andomize  [Q]uit [S]: ")

		var action string
		_, _ = fmt.Scanln(&action)
		action = strings.TrimSpace(strings.ToLower(action))

		switch action {
		case "", "s", "save":
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
		case "r", "random", "rand":
			continue
		case "q", "quit":
			fmt.Println("  " + keyword("Bye!"))
			return nil
		default:
			continue
		}
	}
}

func showPreview(data []byte) {
	sample := "# Hello World\n\nThis is **bold** and *italic* text. Here is `inline code`.\n\n## Code Block\n\n```go\nfunc hello() {\n\tfmt.Println(\"Hello, World!\")\n}\n```\n\n> A wise blockquote once said...\n\n---\n\n[Link to somewhere](https://example.com)"

	r, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes(data),
		glamour.WithWordWrap(60),
	)
	if err != nil {
		fmt.Println("  (preview unavailable)")
		return
	}
	defer r.Close() //nolint:errcheck

	out, err := r.Render(sample)
	if err != nil {
		fmt.Println("  (preview unavailable)")
		return
	}
	fmt.Println(out)
}

func randomTheme() theme {
	colors := make([]string, 10)
	for i := range colors {
		colors[i] = fmt.Sprintf("%d", rand.Intn(256))
	}
	return theme{
		Name:      "Random",
		Doc:       colors[0],
		H1:        colors[1],
		H1Bg:      colors[2],
		H1Bold:    true,
		H2:        colors[3],
		H3:        colors[4],
		H6:        colors[5],
		Code:      colors[6],
		CodeBg:    colors[7],
		CodeBlock: colors[8],
		Link:      colors[9],
		LinkUnder: true,
		HR:        colors[0],
		BqToken:   "│ ",
	}
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

func buildStyle(docColor, h1Color, h1Bg string, h1Bold bool, h2Color, h3Color, h6Color string,
	codeColor, codeBg, codeBlockColor, linkColor string, linkUnderline bool, hrColor, blockQuoteToken string) map[string]interface{} {

	return map[string]interface{}{
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
		"text":          map[string]interface{}{},
		"strikethrough": map[string]interface{}{"crossed_out": true},
		"emph":          map[string]interface{}{"italic": true},
		"strong":        map[string]interface{}{"bold": true},
		"hr": map[string]interface{}{
			"color":  hrColor,
			"format": "\n--------\n",
		},
		"item":        map[string]interface{}{"block_prefix": "• "},
		"enumeration": map[string]interface{}{"block_prefix": ". "},
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
		"table":                  map[string]interface{}{},
		"definition_list":        map[string]interface{}{},
		"definition_term":        map[string]interface{}{},
		"definition_description": map[string]interface{}{"block_prefix": "\n🠶 "},
		"html_block":             map[string]interface{}{},
		"html_span":              map[string]interface{}{},
	}
}
